//go:build e2e
// +build e2e

package redis_cluster_streams_length_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	redis "github.com/kedacore/keda/v2/tests/scalers/redis/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "redis-cluster-streams-length-test"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	redisNamespace            = fmt.Sprintf("%s-redis-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	jobName                   = fmt.Sprintf("%s-job", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	redisPassword             = "admin"
	redisHost                 = fmt.Sprintf("%s-headless", testName)
	minReplicaCount           = 1
	maxReplicaCount           = 5
)

type templateData struct {
	TestNamespace             string
	RedisNamespace            string
	DeploymentName            string
	JobName                   string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	MinReplicaCount           int
	MaxReplicaCount           int
	RedisPassword             string
	RedisPasswordBase64       string
	RedisHost                 string
	ItemsToWrite              int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: redis-worker
        image: ghcr.io/kedacore/tests-redis-streams:latest
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["consumer"]
        env:
        - name: REDIS_MODE
          value: CLUSTER
        - name: REDIS_HOSTS
          value: {{.RedisHost}}.{{.RedisNamespace}}
        - name: REDIS_PORTS
          value: "6379"
        - name: REDIS_STREAM_NAME
          value: my-stream
        - name: REDIS_STREAM_CONSUMER_GROUP_NAME
          value: consumer-group-1
        - name: REDIS_PASSWORD
          value: {{.RedisPassword}}
        - name: DELETE_MESSAGES
          value: "1"
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  password: {{.RedisPasswordBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: redis-cluster-streams
    metadata:
      hostsFromEnv: REDIS_HOSTS
      portsFromEnv: REDIS_PORTS
      stream: my-stream
      streamLength: "15"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	insertJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: redis
        image: ghcr.io/kedacore/tests-redis-streams:latest
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["producer"]
        env:
        - name: REDIS_MODE
          value: CLUSTER
        - name: REDIS_HOSTS
          value: {{.RedisHost}}.{{.RedisNamespace}}
        - name: REDIS_PORTS
          value: "6379"
        - name: REDIS_STREAM_NAME
          value: my-stream
        - name: REDIS_PASSWORD
          value: {{.RedisPassword}}
        - name: NUM_MESSAGES
          value: "{{.ItemsToWrite}}"
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestScaler(t *testing.T) {
	// Create kubernetes resources for PostgreSQL server
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		redis.RemoveCluster(t, testName, redisNamespace)
	})

	// Create Redis Cluster
	redis.InstallCluster(t, kc, testName, redisNamespace, redisPassword)

	// Create kubernetes resources for testing
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

var data = templateData{
	TestNamespace:             testNamespace,
	RedisNamespace:            redisNamespace,
	DeploymentName:            deploymentName,
	ScaledObjectName:          scaledObjectName,
	MinReplicaCount:           minReplicaCount,
	MaxReplicaCount:           maxReplicaCount,
	TriggerAuthenticationName: triggerAuthenticationName,
	SecretName:                secretName,
	JobName:                   jobName,
	RedisPassword:             redisPassword,
	RedisPasswordBase64:       base64.StdEncoding.EncodeToString([]byte(redisPassword)),
	RedisHost:                 redisHost,
	ItemsToWrite:              100,
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

//go:build e2e
// +build e2e

package redis_cluster_streams_lag_test

import (
	"encoding/base64"
	"fmt"
	"testing"
  "time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	redis "github.com/kedacore/keda/v2/tests/scalers/redis/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "redis-cluster-streams-lag-test"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
  activationNamespace       = fmt.Sprintf("%s-activation-ns", testName)
	redisNamespace            = fmt.Sprintf("%s-redis-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	jobName                   = fmt.Sprintf("%s-job", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	redisPassword             = "admin"
	redisHost                 = fmt.Sprintf("%s-headless", testName)
	minReplicaCount           = 0
	maxReplicaCount           = 4
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
      consumerGroup: consumer-group-1
      lagCount: "4"
      activationTargetLag: "3"
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

	// Create Redis Cluster
	redis.InstallCluster(t, kc, testName, redisNamespace, redisPassword)

	// Create kubernetes resources for testing
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
  DeleteKubernetesResources(t, testNamespace, data, templates)

  CreateKubernetesResources(t, kc, testNamespace, activationData, templates)
  testActivationValue(t, kc, activationData)
  DeleteKubernetesResources(t, testNamespace, activationData, templates)
	// cleanup
	redis.RemoveCluster(t, testName, redisNamespace)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func testActivationValue(t *testing.T, kc *kubernetes.Clientset, activationData templateData) {
  t.Log("--- testing activation value ---")
  KubectlApplyWithTemplate(t, activationData, "insertJobTemplate", insertJobTemplate)

  time.Sleep(time.Duration(60 * time.Second))
  AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
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

var activationData = templateData{
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
	ItemsToWrite:              1,
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

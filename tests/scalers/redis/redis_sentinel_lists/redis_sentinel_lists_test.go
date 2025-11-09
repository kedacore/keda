//go:build e2e
// +build e2e

package redis_sentinel_lists_test

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
	testName = "redis-sentinel-lists-test"
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
	redisList                 = "queue"
	redisHost                 = fmt.Sprintf("%s-headless", testName)
	minReplicaCount           = 0
	maxReplicaCount           = 2
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
	RedisList                 string
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
  replicas: 0
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
        image: ghcr.io/kedacore/tests-redis-sentinel-lists:latest
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["read"]
        env:
        - name: REDIS_ADDRESSES
          value: {{.RedisHost}}.{{.RedisNamespace}}:26379
        - name: LIST_NAME
          value: {{.RedisList}}
        - name: REDIS_PASSWORD
          value: {{.RedisPassword}}
        - name: REDIS_SENTINEL_PASSWORD
          value: {{.RedisPassword}}
        - name: REDIS_SENTINEL_MASTER
          value: mymaster
        - name: READ_PROCESS_TIME
          value: "100"
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
  - parameter: sentinelPassword
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
  triggers:
  - type: redis-sentinel
    metadata:
      addressesFromEnv: REDIS_ADDRESSES
      listName: {{.RedisList}}
      sentinelMaster: mymaster
      listLength: "5"
      activationListLength: "10"
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
        image: ghcr.io/kedacore/tests-redis-sentinel-lists:latest
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["write"]
        env:
        - name: REDIS_ADDRESSES
          value: {{.RedisHost}}.{{.RedisNamespace}}:26379
        - name: REDIS_PASSWORD
          value: {{.RedisPassword}}
        - name: REDIS_SENTINEL_PASSWORD
          value: {{.RedisPassword}}
        - name: REDIS_SENTINEL_MASTER
          value: mymaster
        - name: LIST_NAME
          value: {{.RedisList}}
        - name: NO_LIST_ITEMS_TO_WRITE
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
		redis.RemoveSentinel(t, testName, redisNamespace)
	})

	// Create Redis Sentinel
	redis.InstallSentinel(t, kc, testName, redisNamespace, redisPassword)

	// Create kubernetes resources for testing
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.ItemsToWrite = 400
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
	RedisList:                 redisList,
	RedisHost:                 redisHost,
	ItemsToWrite:              0,
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

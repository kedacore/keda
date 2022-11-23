//go:build e2e
// +build e2e

package rabbitmq_queue_amqp_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	. "github.com/kedacore/keda/v2/tests/scalers/rabbitmq"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "rmq-queue-amqp-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	rmqNamespace     = fmt.Sprintf("%s-rmq", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = "hello"
	user             = fmt.Sprintf("%s-user", testName)
	password         = fmt.Sprintf("%s-password", testName)
	vhost            = "/"
	connectionString = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local", user, password, rmqNamespace)
	messageCount     = 100
)

const (
	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 4
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.QueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '10'
        activationValue: '5'
`
)

type templateData struct {
	TestNamespace                string
	DeploymentName               string
	ScaledObjectName             string
	SecretName                   string
	QueueName                    string
	Connection, Base64Connection string
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	RMQInstall(t, kc, rmqNamespace, user, password, vhost)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	testScaling(t, kc)

	testActivationValue(t, kc)

	// cleanup
	t.Log("--- cleaning up ---")
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	RMQUninstall(t, kc, rmqNamespace, user, password, vhost)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			SecretName:       secretName,
			QueueName:        queueName,
			Connection:       connectionString,
			Base64Connection: base64.StdEncoding.EncodeToString([]byte(connectionString)),
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaling(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 1),
		"replica count should be 4 after 1 minute")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func testActivationValue(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation value ---")
	messagesToQueue := 3
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messagesToQueue)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

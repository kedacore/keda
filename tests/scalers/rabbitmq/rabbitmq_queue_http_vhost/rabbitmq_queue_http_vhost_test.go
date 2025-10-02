//go:build e2e
// +build e2e

package rabbitmq_queue_http_vhost_test

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
var _ = godotenv.Load("../../../.env")

const (
	testName = "rmq-queue-http-vhost-test"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	rmqNamespace         = fmt.Sprintf("%s-rmq", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	secretName           = fmt.Sprintf("%s-secret", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	queueName            = "hello"
	user                 = fmt.Sprintf("%s-user", testName)
	password             = fmt.Sprintf("%s-password", testName)
	vhost                = fmt.Sprintf("%s-vhost", testName)
	connectionString     = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/%s", user, password, rmqNamespace, vhost)
	httpConnectionString = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/%s", user, password, rmqNamespace, vhost)
	messageCount         = 100
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
        protocol: http
        mode: QueueLength
        value: '10'
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
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		RMQUninstall(t, rmqNamespace, user, password, vhost, WithoutOAuth())
	})

	// Create kubernetes resources
	RMQInstall(t, kc, rmqNamespace, user, password, vhost, WithoutOAuth())
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	testScaling(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			SecretName:       secretName,
			QueueName:        queueName,
			Connection:       connectionString,
			Base64Connection: base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaling(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount, 0)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 3),
		"replica count should be 4 after 3 minute")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

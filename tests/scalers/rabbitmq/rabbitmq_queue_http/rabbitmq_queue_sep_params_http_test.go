//go:build e2e
// +build e2e

package rabbitmq_queue_http_test

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
	testName       = "rmq-queue-http-sep-par-test"
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
	name: {{.SecretName}}
	namespace: {{.TestNamespace}}
data:
	rabbitmq-password: {{.RabbitMQPasswordBase64}}
	rabbit-username: {{.RabbitMQUserBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
	name: keda-trigger-auth-rabbitmq-secret
	namespace: {{.TestNamespace}}
spec:
	secretTargetRef:
	- parameter: username
		name: {{.SecretName}}
		key: rabbit-username
	- parameter: password
		name: {{.SecretName}}
		key: rabbitmq-password
	`
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

	connectionString     = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpConnectionString = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
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
        protocol: http
        mode: QueueLength
        value: '10'
	  authenticationRef:
        name: keda-trigger-auth-rabbitmq-secret
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
			Base64Connection: base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
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

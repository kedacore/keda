//go:build e2e
// +build e2e

package rabbitmq_queue_http_regex_aad_wi_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	. "github.com/kedacore/keda/v2/tests/scalers/rabbitmq"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "rmq-queue-http-regex-aad-wi-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	rmqNamespace               = fmt.Sprintf("%s-rmq", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	triggerAuthName            = fmt.Sprintf("%s-ta", testName)
	triggerSecretName          = fmt.Sprintf("%s-ta-secret", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	queueName                  = "hello"
	queueRegex                 = "^hell.{1}$"
	user                       = fmt.Sprintf("%s-user", testName)
	password                   = fmt.Sprintf("%s-password", testName)
	vhost                      = "/"
	connectionString           = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpConnectionString       = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpNoAuthConnectionString = fmt.Sprintf("http://rabbitmq.%s.svc.cluster.local/", rmqNamespace)
	rabbitAppClientID          = os.Getenv("TF_AZURE_RABBIT_API_APPLICATION_ID")
	azureADTenantID            = os.Getenv("TF_AZURE_SP_TENANT")
	messageCount               = 100
)

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.TriggerSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  workloadIdentityResource: {{.Base64RabbitAppClientID}}
`
	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload

  secretTargetRef:
    - parameter: workloadIdentityResource
      name: {{.TriggerSecretName}}
      key: workloadIdentityResource
`
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
        vhostName: {{.VHost}}
        host: {{.ConnectionNoAuth}}
        protocol: http
        mode: QueueLength
        value: '10'
        useRegex: 'true'
        operation: sum
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

type templateData struct {
	TestNamespace                              string
	DeploymentName                             string
	TriggerSecretName                          string
	TriggerAuthName                            string
	ScaledObjectName                           string
	SecretName                                 string
	QueueName                                  string
	VHost                                      string
	Connection, Base64Connection               string
	ConnectionNoAuth                           string
	RabbitAppClientID, Base64RabbitAppClientID string
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, rabbitAppClientID, "TF_AZURE_RABBIT_API_APPLICATION_ID env variable is required for rabbitmq workload identity tests")
	require.NotEmpty(t, azureADTenantID, "TF_AZURE_SP_TENANT env variable is required for rabbitmq workload identity tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		RMQUninstall(t, rmqNamespace, user, password, vhost, WithAzureADOAuth(azureADTenantID, rabbitAppClientID))
	})

	// Create kubernetes resources
	RMQInstall(t, kc, rmqNamespace, user, password, vhost, WithAzureADOAuth(azureADTenantID, rabbitAppClientID))
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	testScaling(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			SecretName:              secretName,
			VHost:                   vhost,
			QueueName:               queueRegex,
			Connection:              connectionString,
			Base64Connection:        base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
			TriggerAuthName:         triggerAuthName,
			TriggerSecretName:       triggerSecretName,
			ConnectionNoAuth:        httpNoAuthConnectionString,
			RabbitAppClientID:       rabbitAppClientID,
			Base64RabbitAppClientID: base64.StdEncoding.EncodeToString([]byte(rabbitAppClientID)),
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaling(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount, 0)
	// dummies
	RMQPublishMessages(t, rmqNamespace, connectionString, fmt.Sprintf("%s-1", queueName), messageCount, 0)
	RMQPublishMessages(t, rmqNamespace, connectionString, fmt.Sprintf("%s-%s", queueName, queueName), messageCount, 0)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 2),
		"replica count should be 4 after 2 minute")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

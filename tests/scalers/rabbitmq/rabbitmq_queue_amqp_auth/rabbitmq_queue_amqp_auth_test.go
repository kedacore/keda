//go:build e2e
// +build e2e

package rabbitmq_queue_amqp_auth_test

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
	testName = "rmq-queue-amqp-auth-test"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	rmqNamespace              = fmt.Sprintf("%s-rmq", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	queueName                 = "hello"
	user                      = fmt.Sprintf("%s-user", testName)
	password                  = fmt.Sprintf("%s-password", testName)
	vhost                     = "/"
	NoAuthConnectionString    = fmt.Sprintf("amqp://rabbitmq.%s.svc.cluster.local", rmqNamespace)
	connectionString          = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local", user, password, rmqNamespace)
	messageCount              = 100
)

const (
	scaledObjectAuthFromSecretTemplate = `
---
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
      authenticationRef:
        name: {{.TriggerAuthenticationName}}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: RabbitUsername
    - parameter: password
      name: {{.SecretName}}
      key: RabbitPassword
`
	invalidUsernameAndPasswordTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: Rabbit-Username
    - parameter: password
      name: {{.SecretName}}
      key: Rabbit-Password
`

	invalidPasswordTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: RabbitUsername
    - parameter: password
      name: {{.SecretName}}
      key: Rabbit-Password
`

	scaledObjectAuthFromEnvTemplate = `
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
        usernameFromEnv: RabbitUsername
        passwordFromEnv: RabbitPassword
        mode: QueueLength
        value: '10'
        activationValue: '5'
`
)

type templateData struct {
	TestNamespace                string
	DeploymentName               string
	ScaledObjectName             string
	TriggerAuthenticationName    string
	SecretName                   string
	QueueName                    string
	Username, Base64Username     string
	Password, Base64Password     string
	Connection, Base64Connection string
	FullConnection               string
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

	testAuthFromSecret(t, kc, data)
	testAuthFromEnv(t, kc, data)

	testInvalidPassword(t, kc, data)
	testInvalidUsernameAndPassword(t, kc, data)

	testActivationValue(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:             testNamespace,
			DeploymentName:            deploymentName,
			ScaledObjectName:          scaledObjectName,
			TriggerAuthenticationName: triggerAuthenticationName,
			SecretName:                secretName,
			QueueName:                 queueName,
			Username:                  user,
			Base64Username:            base64.StdEncoding.EncodeToString([]byte(user)),
			Password:                  password,
			Base64Password:            base64.StdEncoding.EncodeToString([]byte(password)),
			Connection:                connectionString,
			Base64Connection:          base64.StdEncoding.EncodeToString([]byte(NoAuthConnectionString)),
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentWithAuthEnvTemplate},
		}
}

func testAuthFromSecret(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	KubectlApplyWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)
	defer KubectlDeleteWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount, 0)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 1),
		"replica count should be 4 after 1 minute")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func testAuthFromEnv(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectAuthFromEnvTemplate", scaledObjectAuthFromEnvTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectAuthFromEnvTemplate", scaledObjectAuthFromEnvTemplate)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount, 0)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 1),
		"replica count should be 4 after 1 minute")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func testInvalidPassword(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing invalid password ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	KubectlApplyWithTemplate(t, data, "invalidPasswordTriggerAuthenticationTemplate", invalidPasswordTriggerAuthenticationTemplate)
	defer KubectlDeleteWithTemplate(t, data, "invalidPasswordTriggerAuthenticationTemplate", invalidPasswordTriggerAuthenticationTemplate)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
}

func testInvalidUsernameAndPassword(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing invalid username and password ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectAuthFromSecretTemplate", scaledObjectAuthFromSecretTemplate)
	KubectlApplyWithTemplate(t, data, "invalidUsernameAndPasswordTriggerAuthenticationTemplate", invalidUsernameAndPasswordTriggerAuthenticationTemplate)
	defer KubectlDeleteWithTemplate(t, data, "invalidUsernameAndPasswordTriggerAuthenticationTemplate", invalidUsernameAndPasswordTriggerAuthenticationTemplate)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
}

func testActivationValue(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation value ---")
	messagesToQueue := 3
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messagesToQueue, 0)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

//go:build e2e
// +build e2e

package default_scaling_strategy_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
	. "github.com/kedacore/keda/v2/tests/scalers/rabbitmq"
)

var _ = godotenv.Load("../../.env") // For loading env variables from .env

const (
	testName = "default-scaling-strategy-test"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	rmqNamespace         = fmt.Sprintf("%s-rmq", testName)
	scaledJobName        = fmt.Sprintf("%s-sj", testName)
	queueName            = "hello"
	user                 = fmt.Sprintf("%s-user", testName)
	password             = fmt.Sprintf("%s-password", testName)
	vhost                = "/"
	connectionString     = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpConnectionString = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	secretName           = fmt.Sprintf("%s-secret", testName)
)

// YAML templates for your Kubernetes resources
const (
	scaledJobTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  RabbitApiHost: {{.Base64Connection}}
---
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScaledJobName}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: sleeper
            image: busybox
            command:
            - sleep
            - "300"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 10
  scalingStrategy:
    strategy: "default"
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.QueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '1'
`
)

type templateData struct {
	ScaledJobName    string
	TestNamespace    string
	QueueName        string
	SecretName       string
	Base64Connection string
}

func TestScalingStrategy(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		RMQUninstall(t, rmqNamespace, user, password, vhost, WithoutOAuth())
	})

	RMQInstall(t, kc, rmqNamespace, user, password, vhost, WithoutOAuth())
	// Publish 0 messges but create the queue
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	testDefaultScaling(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			// Populate fields required in YAML templates
			ScaledJobName:    scaledJobName,
			TestNamespace:    testNamespace,
			QueueName:        queueName,
			Base64Connection: base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
			SecretName:       secretName,
		}, []Template{
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func testDefaultScaling(t *testing.T, kc *kubernetes.Clientset) {
	iterationCount := 20
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 3)
	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, 3, iterationCount, 1),
		"job count should be %d after %d iterations", 3, iterationCount)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 3)
	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, 6, iterationCount, 1),
		"job count should be %d after %d iterations", 6, iterationCount)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 6)
	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, 10, iterationCount, 1),
		"job count should be %d after %d iterations", 10, iterationCount)
}

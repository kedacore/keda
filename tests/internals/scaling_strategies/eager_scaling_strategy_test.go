//go:build e2e
// +build e2e

package eager_scaling_strategy_test

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
	testName = "eager-scaling-strategy-test"
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
    strategy: "eager"
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
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 0, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	testEagerScaling(t, kc)
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

func testEagerScaling(t *testing.T, kc *kubernetes.Clientset) {
	iterationCount := 20
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 4, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 4, iterationCount, 1),
		"job count should be %d after %d iterations", 4, iterationCount)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 4, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 8, iterationCount, 1),
		"job count should be %d after %d iterations", 8, iterationCount)

	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 8, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 10, iterationCount, 1),
		"job count should be %d after %d iterations", 10, iterationCount)
}

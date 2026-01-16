//go:build e2e
// +build e2e

package scaledjob_conditions_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	. "github.com/kedacore/keda/v2/tests/helper"
	. "github.com/kedacore/keda/v2/tests/scalers/rabbitmq"
)

var _ = godotenv.Load("../../.env")

const (
	testName = "scaledjob-conditions-test"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	rmqNamespace         = fmt.Sprintf("%s-rmq", testName)
	scaledJobName        = fmt.Sprintf("%s-sj", testName)
	queueName            = "hello"
	nonExistingQueueName = "not-existing-queue"
	user                 = fmt.Sprintf("%s-user", testName)
	password             = fmt.Sprintf("%s-password", testName)
	vhost                = "/"
	connectionString     = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpConnectionString = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	secretName           = fmt.Sprintf("%s-secret", testName)
)

const (
	scaledJobTrueConditionTemplate = `
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
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: sleeper
            image: docker.io/library/busybox
            command:
            - sleep
            - "10"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 5
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.QueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '1'
`

	scaledJobFalseConditionTemplate = `
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
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: sleeper
            image: docker.io/library/busybox
            command:
            - sleep
            - "10"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 5
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.NonExistingQueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '1'
`

	scaledJobUnknownConditionTemplate = `
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
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: sleeper
            image: docker.io/library/busybox
            command:
            - sleep
            - "10"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 5
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.QueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '1'
    - type: rabbitmq
      metadata:
        queueName: {{.NonExistingQueueName}}
        hostFromEnv: RabbitApiHost
        mode: QueueLength
        value: '1'
`
)

type templateData struct {
	ScaledJobName        string
	TestNamespace        string
	QueueName            string
	NonExistingQueueName string
	SecretName           string
	Base64Connection     string
}

func TestScaledJobConditions(t *testing.T) {
	kc := GetKubernetesClient(t)

	// Setup RabbitMQ
	RMQInstall(t, kc, rmqNamespace, user, password, vhost, WithoutOAuth())
	// Create the existing queue
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 0, 0)
	WaitForAllJobsSuccess(t, kc, rmqNamespace, 60, 1)

	t.Cleanup(func() {
		RMQUninstall(t, rmqNamespace, user, password, vhost, WithoutOAuth())
	})

	t.Run("Test ReadyCondition True and ActiveCondition True", func(t *testing.T) {
		testReadyConditionTrue(t, kc)
	})

	t.Run("Test ReadyCondition False and ActiveCondition False", func(t *testing.T) {
		testReadyConditionFalse(t, kc)
	})

	t.Run("Test ReadyCondition Unknown and ActiveCondition True", func(t *testing.T) {
		testReadyConditionUnknown(t, kc)
	})
}

// testReadyConditionTrue tests that ReadyCondition is True when triggers work correctly and ActiveCondition is True
func testReadyConditionTrue(t *testing.T, kc *kubernetes.Clientset) {
	testNs := fmt.Sprintf("%s-true", testNamespace)
	scaledJobNs := fmt.Sprintf("%s-true", scaledJobName)

	data := templateData{
		ScaledJobName:        scaledJobNs,
		TestNamespace:        testNs,
		QueueName:            queueName,
		NonExistingQueueName: nonExistingQueueName,
		SecretName:           secretName,
		Base64Connection:     base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
	}

	templates := []Template{
		{Name: "scaledJobTemplate", Config: scaledJobTrueConditionTemplate},
	}

	CreateKubernetesResources(t, kc, testNs, data, templates)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNs, data, templates)
	})

	// Publish messages to trigger scaling
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 3, 0)

	// Wait for ScaledJob to be created and check conditions
	assert.True(t, WaitForScaledJobConditions(t, kc, scaledJobNs, testNs, 60, 2,
		func(sj *kedav1alpha1.ScaledJob) bool {
			readyCondition := sj.Status.Conditions.GetReadyCondition()
			activeCondition := sj.Status.Conditions.GetActiveCondition()

			t.Logf("ReadyCondition: Status=%s, Reason=%s, Message=%s",
				readyCondition.Status, readyCondition.Reason, readyCondition.Message)
			t.Logf("ActiveCondition: Status=%s, Reason=%s, Message=%s",
				activeCondition.Status, activeCondition.Reason, activeCondition.Message)

			// Check ReadyCondition is True
			if !readyCondition.IsTrue() {
				t.Logf("ReadyCondition is not True yet")
				return false
			}

			// Check ActiveCondition is True (because we have messages)
			if !activeCondition.IsTrue() {
				t.Logf("ActiveCondition is not True yet")
				return false
			}

			assert.Equal(t, "ScaledJobReady", readyCondition.Reason)
			assert.Equal(t, "ScalerActive", activeCondition.Reason)
			return true
		}),
		"ScaledJob should have ReadyCondition=True and ActiveCondition=True")

	// Verify jobs were created
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobNs, testNs, 3, 20, 1),
		"job count should be 3")
}

// testReadyConditionFalse tests that ReadyCondition is False when all triggers fail and ActiveCondition is False when triggers are not active
func testReadyConditionFalse(t *testing.T, kc *kubernetes.Clientset) {
	testNs := fmt.Sprintf("%s-false", testNamespace)
	scaledJobNs := fmt.Sprintf("%s-false", scaledJobName)

	data := templateData{
		ScaledJobName:        scaledJobNs,
		TestNamespace:        testNs,
		QueueName:            queueName,
		NonExistingQueueName: nonExistingQueueName,
		SecretName:           secretName,
		Base64Connection:     base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
	}

	templates := []Template{
		{Name: "scaledJobTemplate", Config: scaledJobFalseConditionTemplate},
	}

	CreateKubernetesResources(t, kc, testNs, data, templates)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNs, data, templates)
	})

	// Wait for ScaledJob conditions to be set
	assert.True(t, WaitForScaledJobConditions(t, kc, scaledJobNs, testNs, 60, 2,
		func(sj *kedav1alpha1.ScaledJob) bool {
			readyCondition := sj.Status.Conditions.GetReadyCondition()
			activeCondition := sj.Status.Conditions.GetActiveCondition()

			t.Logf("ReadyCondition: Status=%s, Reason=%s, Message=%s",
				readyCondition.Status, readyCondition.Reason, readyCondition.Message)
			t.Logf("ActiveCondition: Status=%s, Reason=%s, Message=%s",
				activeCondition.Status, activeCondition.Reason, activeCondition.Message)

			// Check ReadyCondition is False (non-existing queue causes error)
			if !readyCondition.IsFalse() {
				t.Logf("ReadyCondition is not False yet")
				return false
			}

			// Check ActiveCondition is False (no active triggers due to error)
			if !activeCondition.IsFalse() {
				t.Logf("ActiveCondition is not False yet")
				return false
			}

			assert.Equal(t, "TriggerError", readyCondition.Reason)
			assert.Contains(t, readyCondition.Message, "Triggers defined in ScaledJob are not working correctly")
			assert.Equal(t, "ScalerNotActive", activeCondition.Reason)
			return true
		}),
		"ScaledJob should have ReadyCondition=False and ActiveCondition=False")
}

// testReadyConditionUnknown tests that ReadyCondition is Unknown when some triggers work and some fail, and ActiveCondition is True when at least one trigger is active
func testReadyConditionUnknown(t *testing.T, kc *kubernetes.Clientset) {
	testNs := fmt.Sprintf("%s-unknown", testNamespace)
	scaledJobNs := fmt.Sprintf("%s-unknown", scaledJobName)

	data := templateData{
		ScaledJobName:        scaledJobNs,
		TestNamespace:        testNs,
		QueueName:            queueName,
		NonExistingQueueName: nonExistingQueueName,
		SecretName:           secretName,
		Base64Connection:     base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
	}

	templates := []Template{
		{Name: "scaledJobTemplate", Config: scaledJobUnknownConditionTemplate},
	}

	CreateKubernetesResources(t, kc, testNs, data, templates)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNs, data, templates)
	})

	// Publish messages to the existing queue to make one trigger active
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, 2, 0)

	// Wait for ScaledJob conditions to be set
	assert.True(t, WaitForScaledJobConditions(t, kc, scaledJobNs, testNs, 60, 2,
		func(sj *kedav1alpha1.ScaledJob) bool {
			readyCondition := sj.Status.Conditions.GetReadyCondition()
			activeCondition := sj.Status.Conditions.GetActiveCondition()

			t.Logf("ReadyCondition: Status=%s, Reason=%s, Message=%s",
				readyCondition.Status, readyCondition.Reason, readyCondition.Message)
			t.Logf("ActiveCondition: Status=%s, Reason=%s, Message=%s",
				activeCondition.Status, activeCondition.Reason, activeCondition.Message)

			// Check ReadyCondition is Unknown (one trigger works, one fails)
			if !readyCondition.IsUnknown() {
				t.Logf("ReadyCondition is not Unknown yet")
				return false
			}

			// Check ActiveCondition is True (at least one trigger is active)
			if !activeCondition.IsTrue() {
				t.Logf("ActiveCondition is not True yet")
				return false
			}

			assert.Equal(t, "PartialTriggerError", readyCondition.Reason)
			assert.Contains(t, readyCondition.Message, "Some triggers defined in ScaledJob are not working correctly")
			assert.Equal(t, "ScalerActive", activeCondition.Reason)
			return true
		}),
		"ScaledJob should have ReadyCondition=Unknown and ActiveCondition=True")

	// Verify that at least some jobs were created
	time.Sleep(5 * time.Second)
	jobCount, _ := GetScaledJobCount(kc, scaledJobNs, testNs)
	assert.Greater(t, jobCount, int64(0), "at least one job should be created when one trigger is active")
}

// Helper function to wait for ScaledJob conditions
func WaitForScaledJobConditions(t *testing.T, kc *kubernetes.Clientset, scaledJobName, namespace string,
	iterations, intervalSeconds int, conditionCheck func(*kedav1alpha1.ScaledJob) bool) bool {
	for i := 0; i < iterations; i++ {
		scaledJob, err := GetScaledJob(t, kc, scaledJobName, namespace)
		if err == nil && scaledJob != nil {
			if conditionCheck(scaledJob) {
				return true
			}
		}
		t.Logf("Waiting for ScaledJob conditions... (%d/%d)", i+1, iterations)
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return false
}

// Helper function to get ScaledJob
func GetScaledJob(t *testing.T, kc *kubernetes.Clientset, name, namespace string) (*kedav1alpha1.ScaledJob, error) {
	kedaClient := GetKedaKubernetesClient(t)
	return kedaClient.ScaledJobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// Helper function to get ScaledJob count
func GetScaledJobCount(kc *kubernetes.Clientset, scaledJobName, namespace string) (int64, error) {
	jobList, err := kc.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scaledjob.keda.sh/name=%s", scaledJobName),
	})
	if err != nil {
		return 0, err
	}
	return int64(len(jobList.Items)), nil
}

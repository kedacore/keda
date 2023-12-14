//go:build e2e
// +build e2e

// go test -v -tags e2e ./internals/pause_scaledjob/pause_scaledjob_test.go

package pause_scaledjob_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file

const (
	testName = "pause-scaledjob-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	serviceName           = fmt.Sprintf("%s-service", testName)
	scalerName            = fmt.Sprintf("%s-scaler", testName)
	scaledJobName         = fmt.Sprintf("%s-sj", testName)
	minReplicaCount       = 0
	maxReplicaCount       = 3
	iterationCountInitial = 15
	iterationCountLatter  = 30
)

type templateData struct {
	TestNamespace                    string
	ServiceName                      string
	ScalerName                       string
	ScaledJobName                    string
	MinReplicaCount, MaxReplicaCount int
	MetricThreshold, MetricValue     int
}

const (
	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
    - port: 6000
      targetPort: 6000
  selector:
    app: {{.ScalerName}}
`

	scalerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ScalerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScalerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.ScalerName}}
  template:
    metadata:
      labels:
        app: {{.ScalerName}}
    spec:
      containers:
        - name: scaler
          image: ghcr.io/kedacore/tests-external-scaler-e2e:latest
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
spec:
  pollingInterval: 5
  maxReplicaCount: {{.MaxReplicaCount}}
  minReplicaCount: {{.MinReplicaCount}}
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "15"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  triggers:
    - type: external
      metadata:
        scalerAddress: {{.ServiceName}}.{{.TestNamespace}}:6000
        metricThreshold: "{{.MetricThreshold}}"
        metricValue: "{{.MetricValue}}"
`
)

// Util function
func WaitForJobByFilterCountUntilIteration(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int, listOptions metav1.ListOptions) bool {
	var isTargetAchieved = false

	for i := 0; i < iterations; i++ {
		jobList, _ := kc.BatchV1().Jobs(namespace).List(context.Background(), listOptions)
		count := len(jobList.Items)

		t.Logf("Waiting for job count to hit target. Namespace - %s, Current  - %d, Target - %d",
			namespace, count, target)

		if count == target {
			isTargetAchieved = true
		} else {
			isTargetAchieved = false
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return isTargetAchieved
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	metricValue := 1

	data, templates := getTemplateData(metricValue)

	listOptions := metav1.ListOptions{
		FieldSelector: "status.successful=0",
	}

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, data.MetricThreshold, iterationCountInitial, 1, listOptions),
		"job count should be %d after %d iterations", data.MetricThreshold, iterationCountInitial)

	// test scaling
	testPause(t, kc, listOptions)
	testUnpause(t, kc, data, listOptions)

	testPause(t, kc, listOptions)
	testUnpauseWithBool(t, kc, data, listOptions)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData(metricValue int) (templateData, []Template) {
	return templateData{
			TestNamespace:   testNamespace,
			ScaledJobName:   scaledJobName,
			ScalerName:      scalerName,
			ServiceName:     serviceName,
			MinReplicaCount: minReplicaCount,
			MaxReplicaCount: maxReplicaCount,
			MetricThreshold: 1,
			MetricValue:     metricValue,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func testPause(t *testing.T, kc *kubernetes.Clientset, listOptions metav1.ListOptions) {
	t.Log("--- testing Paused annotation ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused=true --namespace %s", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count does not change as job is paused")

	expectedTarget := 0
	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1, listOptions),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)
}

func testUnpause(t *testing.T, kc *kubernetes.Clientset, data templateData, listOptions metav1.ListOptions) {
	t.Log("--- testing removing Paused annotation ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused- --namespace %s", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count increases from zero as job is no longer paused")

	expectedTarget := data.MetricThreshold
	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1, listOptions),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)
}

func testUnpauseWithBool(t *testing.T, kc *kubernetes.Clientset, data templateData, listOptions metav1.ListOptions) {
	t.Log("--- test setting Paused annotation to false ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused=false --namespace %s --overwrite=true", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count increases from zero as job is no longer paused")

	expectedTarget := data.MetricThreshold
	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1, listOptions),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)
}

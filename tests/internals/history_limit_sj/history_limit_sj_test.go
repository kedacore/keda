//go:build e2e
// +build e2e

// go test -v -tags e2e ./internals/history_limit_sj/history_limit_sj_test.go

package history_limit_sj_test

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
	failedscaledJobName   = fmt.Sprintf("%s-sj-fail", testName)
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
	FailedScaledJobName              string
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
  successfulJobsHistoryLimit: 5
  failedJobsHistoryLimit: 5
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
	failedscaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.FailedScaledJobName}}
  namespace: {{.TestNamespace}}
spec:
  pollingInterval: 5
  maxReplicaCount: {{.MaxReplicaCount}}
  minReplicaCount: {{.MinReplicaCount}}
  successfulJobsHistoryLimit: 5
  failedJobsHistoryLimit: 5
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sh
            - -c
			- exit 1
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
	testSuccessfulJobsHistoryLimit(t, kc, data, listOptions)
	testFailedJobsHistoryLimit(t, kc, data, listOptions)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData(metricValue int) (templateData, []Template) {
	return templateData{
			TestNamespace:       testNamespace,
			ScaledJobName:       scaledJobName,
			FailedScaledJobName: failedscaledJobName,
			ScalerName:          scalerName,
			ServiceName:         serviceName,
			MinReplicaCount:     minReplicaCount,
			MaxReplicaCount:     maxReplicaCount,
			MetricThreshold:     10,
			MetricValue:         metricValue,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
			{Name: "failedscaledJobTemplate", Config: failedscaledJobTemplate},
		}
}

func testSuccessfulJobsHistoryLimit(t *testing.T, kc *kubernetes.Clientset, data templateData, listOptions metav1.ListOptions) {
	t.Log("--- testing successfulJobsHistoryLimit ---")

	// Update list options to filter only successful jobs
	listOptions.FieldSelector = "status.succeeded>0"

	// Wait for jobs to be created
	expectedTarget := data.MetricThreshold
	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1, listOptions),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)

	// Verify that only 5 jobs are retained due to successfulJobsHistoryLimit
	jobList, err := kc.BatchV1().Jobs(testNamespace).List(context.Background(), listOptions)
	assert.NoError(t, err, "failed to list jobs")

	var retainedJobs int
	for _, job := range jobList.Items {
		if job.Name != scaledJobName {
			retainedJobs++
		}
	}

	assert.Equal(t, 5, retainedJobs, "number of retained jobs should be 5")
}

func testFailedJobsHistoryLimit(t *testing.T, kc *kubernetes.Clientset, data templateData, listOptions metav1.ListOptions) {
	t.Log("--- testing failedJobsHistoryLimit ---")

	// Update list options to filter only failed jobs
	listOptions.FieldSelector = "status.failed>0"

	// Wait for jobs to be created
	expectedTarget := data.MetricThreshold

	assert.True(t, WaitForJobByFilterCountUntilIteration(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1, listOptions),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)

	// Verify that only 5 jobs are retained due to failedJobsHistoryLimit
	jobList, err := kc.BatchV1().Jobs(testNamespace).List(context.Background(), listOptions)
	assert.NoError(t, err, "failed to list jobs")

	var retainedJobs int
	for _, job := range jobList.Items {
		if job.Name != failedscaledJobName {
			retainedJobs++
		}
	}

	assert.Equal(t, 5, retainedJobs, "number of retained jobs should be 5")
}

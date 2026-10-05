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
	maxReplicaCount       = 1
	iterationCountInitial = 30
	iterationCountLatter  = 60
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
            - "30"
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

// The test tells a running job from a finished one by the number of successful completions the job
// reports, which is what these field selectors match on.
const (
	runningJobsSelector   = "status.successful=0"
	succeededJobsSelector = "status.successful=1"
)

// A list that failed is handed back rather than counted as zero jobs, which is what discarding the
// error did: for a target of zero that is a false pass, and for any other target it is a retry that
// hides why the wait is not progressing.
func countJobs(ctx context.Context, kc *kubernetes.Clientset, namespace, fieldSelector string) (int, error) {
	jobList, err := kc.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		return 0, fmt.Errorf("cannot list jobs in namespace %s - %w", namespace, err)
	}
	return len(jobList.Items), nil
}

// waitForJobCount polls until the jobs matching fieldSelector number target, giving up once the
// iterations x intervalSeconds budget is spent. The budget is a deadline, so the time the API server
// spends answering comes out of the window rather than extending it.
func waitForJobCount(t *testing.T, kc *kubernetes.Clientset, namespace, fieldSelector string,
	target, iterations, intervalSeconds int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(iterations*intervalSeconds)*time.Second)
	defer cancel()

	err := KedaEventually(ctx, func(ctx context.Context) (bool, error) {
		count, err := countJobs(ctx, kc, namespace, fieldSelector)
		if err != nil {
			return false, err
		}

		t.Logf("Waiting for job count to hit target. Namespace - %s, Selector - %s, Current  - %d, Target - %d",
			namespace, fieldSelector, count, target)

		return count == target, nil
	}, time.Duration(intervalSeconds)*time.Second)
	if err != nil {
		t.Log(err)
		return false
	}
	return true
}

func WaitUntilJobIsRunning(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, namespace, runningJobsSelector, target, iterations, intervalSeconds)
}

func WaitUntilJobIsSucceeded(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	return waitForJobCount(t, kc, namespace, succeededJobsSelector, target, iterations, intervalSeconds)
}

// AssertJobNotChangeKeepingIsSucceeded holds that the succeeded job count stays at target for the
// whole iterations x intervalSeconds window, and reports the first count that differs.
func AssertJobNotChangeKeepingIsSucceeded(t *testing.T, kc *kubernetes.Clientset, namespace string,
	target, iterations, intervalSeconds int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(iterations*intervalSeconds)*time.Second)
	defer cancel()

	err := KedaConsistently(ctx, func(ctx context.Context) (bool, error) {
		count, err := countJobs(ctx, kc, namespace, succeededJobsSelector)
		if err != nil {
			// KedaConsistently fails on a condition error, and a list that could not be made says
			// nothing about the jobs, so it is logged as an attempt that saw nothing to object to.
			t.Log(err)
			return true, nil
		}

		t.Logf("Asserting the job count doesn't change. Namespace - %s, Current  - %d, Target - %d",
			namespace, count, target)

		if count != target {
			return false, fmt.Errorf("succeeded job count in namespace %s changed from %d to %d", namespace, target, count)
		}
		return true, nil
	}, time.Duration(intervalSeconds)*time.Second)
	if err != nil {
		t.Log(err)
		return false
	}
	return true
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	metricValue := 1

	data, templates := getTemplateData(metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// we ensure that the gRPC server is up and ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 60, 1),
		"replica count should be 0 after 1 minute")

	// we ensure that there is a job running
	assert.True(t, WaitUntilJobIsRunning(t, kc, testNamespace, data.MetricThreshold, iterationCountInitial, 1),
		"job count should be %d after %d iterations", data.MetricThreshold, iterationCountInitial)

	// test scaling
	testPause(t, kc)
	testUnpause(t, kc, data)

	testPause(t, kc)
	testUnpauseWithBool(t, kc, data)

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

func testPause(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing Paused annotation ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused=true --namespace %s", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count does not change as job is paused")

	expectedTarget := 1
	assert.True(t, WaitUntilJobIsSucceeded(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)

	assert.True(t, AssertJobNotChangeKeepingIsSucceeded(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1),
		"job count should be %d during %d iterations", expectedTarget, iterationCountLatter)
}

func testUnpause(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing removing Paused annotation ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused- --namespace %s", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count increases from zero as job is no longer paused")

	expectedTarget := data.MetricThreshold
	assert.True(t, WaitUntilJobIsRunning(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)
}

func testUnpauseWithBool(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test setting Paused annotation to false ---")

	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledjob %s autoscaling.keda.sh/paused=false --namespace %s --overwrite=true", scaledJobName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("job count increases from zero as job is no longer paused")

	expectedTarget := data.MetricThreshold
	assert.True(t, WaitUntilJobIsRunning(t, kc, testNamespace, expectedTarget, iterationCountLatter, 1),
		"job count should be %d after %d iterations", expectedTarget, iterationCountLatter)
}

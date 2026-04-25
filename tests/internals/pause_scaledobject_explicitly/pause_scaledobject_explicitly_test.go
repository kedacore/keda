//go:build e2e
// +build e2e

package pause_scaledobject_explicitly_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file

const (
	testName = "pause-scaledobject-explicitly-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	hpaName                 = fmt.Sprintf("keda-hpa-%s", scaledObjectName)
	testScaleOutWaitMin     = 1
	testPauseAtNWaitMin     = 1
	testScaleInWaitMin      = 1
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	MonitoredDeploymentName string
}

const (
	monitoredDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: {{.MonitoredDeploymentName}}
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
        - name: {{.DeploymentName}}
          image: ghcr.io/nginx/nginx-unprivileged:1.26
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
  minReplicaCount: 0
  maxReplicaCount: 10
  cooldownPeriod:  5
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	unpausedMethods := [](func(assert.TestingT)){removeScaledObjectPausedAnnotation, setScaledObjectPausedAnnotationFalse}

	for _, unpauseMethod := range unpausedMethods {
		CreateKubernetesResources(t, kc, testNamespace, data, templates)

		// scaling to paused replica count
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
			"replica count should be 0 after 1 minute")
		// test scaling
		testPauseWhenScaleOut(t, kc)
		testScaleOut(t, kc, unpauseMethod)
		testPauseWhenScaleIn(t, kc)
		testScaleIn(t, kc, unpauseMethod)
		testBothPauseAnnotationActive(t, kc)
		testHPANotExistWhilePaused(t, kc)
		testHPANotExistWhilePausedReplicas(t, kc)
		testChangePausedReplicasValue(t, kc)
		testSwitchFromPausedReplicasToPaused(t, kc)

		// cleanup
		DeleteKubernetesResources(t, testNamespace, data, templates)
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			MonitoredDeploymentName: monitoredDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectAnnotatedTemplate", Config: scaledObjectTemplate},
		}
}

func upsertScaledObjectPausedAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused=true --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func removeScaledObjectPausedAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused- --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func setScaledObjectPausedAnnotationFalse(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused=false --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func upsertScaledObjectPausedReplicasAnnotation(t assert.TestingT, value int) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused-replicas=%d --overwrite", scaledObjectName, testNamespace, value))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func removeScaledObjectPausedReplicasAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused-replicas- --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testPauseWhenScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing pausing at 0 ---")

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testScaleOutWaitMin),
		"monitoredDeploymentName replica count should be 2 after %d minute(s)", testScaleOutWaitMin)

	upsertScaledObjectPausedAnnotation(t)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 2, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 2, 60, testScaleOutWaitMin),
		"monitoredDeploymentName replica count should be 2 after %d minute(s)", testScaleOutWaitMin)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, unpauseMethod func(assert.TestingT)) {
	t.Log("--- testing scale out ---")

	unpauseMethod(t)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 5, 60, testScaleOutWaitMin),
		"monitoredDeploymentName replica count should be 5 after %d minute(s)", testScaleOutWaitMin)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, testScaleOutWaitMin),
		"replica count should be 5 after %d minute(s)", testScaleOutWaitMin)
}

func testPauseWhenScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing pausing at N ---")

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	upsertScaledObjectPausedAnnotation(t)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"monitoredDeploymentName replica count should be 0 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 10, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, unpauseMethod func(assert.TestingT)) {
	t.Log("--- testing scale in ---")

	unpauseMethod(t)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testScaleInWaitMin),
		"replica count should be 0 after %d minutes", testScaleInWaitMin)
}

func testBothPauseAnnotationActive(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing paused and paused-replicas annotations at the same time---")

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	t.Log("--- testing adding paused first---")
	upsertScaledObjectPausedAnnotation(t)
	time.Sleep(3 * time.Second)
	upsertScaledObjectPausedReplicasAnnotation(t, 5)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 10, 60, testPauseAtNWaitMin),
		"monitoredDeploymentName replica count should be 10 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	t.Log("--- testing recover scale out---")
	removeScaledObjectPausedAnnotation(t)
	removeScaledObjectPausedReplicasAnnotation(t)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 10, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	t.Log("--- testing adding paused-replica first---")
	upsertScaledObjectPausedReplicasAnnotation(t, 5)
	time.Sleep(3 * time.Second)
	upsertScaledObjectPausedAnnotation(t)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 10, 60, testPauseAtNWaitMin),
		"monitoredDeploymentName replica count should be 0 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)
}

func testHPANotExistWhilePaused(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing HPA does not exist while paused ---")

	upsertScaledObjectPausedAnnotation(t)
	time.Sleep(5 * time.Second)

	_, err := kc.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(context.Background(), hpaName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err), "HPA should not exist while paused with paused=true")

	removeScaledObjectPausedAnnotation(t)
	time.Sleep(5 * time.Second)
}

func testHPANotExistWhilePausedReplicas(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing HPA does not exist while paused-replicas is set ---")

	upsertScaledObjectPausedReplicasAnnotation(t, 3)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 1),
		"replica count should be 3 after 1 minute")

	_, err := kc.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(context.Background(), hpaName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err), "HPA should not exist while paused with paused-replicas")

	removeScaledObjectPausedReplicasAnnotation(t)
	time.Sleep(5 * time.Second)
}

func testChangePausedReplicasValue(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing changing paused-replicas value while paused ---")

	upsertScaledObjectPausedReplicasAnnotation(t, 3)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 1),
		"replica count should be 3 after 1 minute")

	upsertScaledObjectPausedReplicasAnnotation(t, 7)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 7, 60, 1),
		"replica count should be 7 after 1 minute")

	removeScaledObjectPausedReplicasAnnotation(t)
	time.Sleep(5 * time.Second)
}

func testSwitchFromPausedReplicasToPaused(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing switching from paused-replicas to paused ---")

	// Ensure a stable starting state: use paused-replicas to set replicas to 5
	upsertScaledObjectPausedReplicasAnnotation(t, 5)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// Switch: remove paused-replicas, add paused=true
	removeScaledObjectPausedReplicasAnnotation(t)
	upsertScaledObjectPausedAnnotation(t)
	time.Sleep(5 * time.Second)

	// HPA should not exist after switch
	_, err := kc.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(context.Background(), hpaName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err), "HPA should not exist after switching to paused=true")

	// Replicas should stay frozen at 5
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 5, 30)

	// Cleanup
	removeScaledObjectPausedAnnotation(t)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

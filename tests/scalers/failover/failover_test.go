//go:build e2e
// +build e2e

package failover_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kedaclientv1alpha1 "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned/typed/keda/v1alpha1"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "failover-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
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
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 5
  pollingInterval: 5
  cooldownPeriod: 5
  triggers:
  - type: mock
    metadata:
      mockMetricValue: "20"
      mockIsActive: "true"
      mockTargetValue: "10"
    fallback:
      failover: true
      thresholds:
        failAfter: 3
        recoverAfter: 5
  - type: mock
    metadata:
      mockMetricValue: "15"
      mockIsActive: "true"
      mockTargetValue: "10"
`
)

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func TestMultiTriggerFailover(t *testing.T) {
	kc := GetKubernetesClient(t)
	kedaClient := GetKedaKubernetesClient(t)
	data, templates := getTemplateData()

	CreateNamespace(t, kc, testNamespace)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Test scenarios
	testPrimaryTriggerScaling(t, kc, kedaClient)
	testFailoverToSecondary(t, kc, kedaClient)
	testRecoveryToPrimary(t, kc, kedaClient)
}

func testPrimaryTriggerScaling(t *testing.T, kc *kubernetes.Clientset, kedaClient *kedaclientv1alpha1.KedaV1alpha1Client) {
	t.Log("--- testing primary trigger scaling ---")

	// Wait for deployment to scale up (primary metric value 20 > target 10 → replicas 2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 3),
		"replica count should be 2 after scaling with primary trigger")

	// Verify ActiveTriggerIndex is 0 (primary)
	so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, so.Status.ActiveTriggerIndex, "ActiveTriggerIndex should be set")
	assert.Equal(t, int32(0), *so.Status.ActiveTriggerIndex, "Should be using primary trigger (index 0)")
}

func testFailoverToSecondary(t *testing.T, kc *kubernetes.Clientset, kedaClient *kedaclientv1alpha1.KedaV1alpha1Client) {
	t.Log("--- testing failover to secondary trigger ---")

	// Start watching for KEDAScalerFailedOver event before triggering failure
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	eventChan := StartEventWatch(
		ctx, t, kc, testNamespace, scaledObjectName, "ScaledObject",
		"KEDAScalerFailedOver", corev1.EventTypeNormal,
		[]string{"Failover from trigger 0 to trigger 1"},
		"",
	)

	// Update ScaledObject to make primary trigger fail
	so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	require.NoError(t, err)

	// Change primary trigger metadata to simulate failure
	so.Spec.Triggers[0].Metadata["mockShouldFail"] = "true"
	so.Spec.Triggers[0].Metadata["mockFailureType"] = "connection"

	_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
	require.NoError(t, err)

	t.Log("Primary trigger set to fail, waiting for failover event...")

	// Wait for failover event (with 5 min timeout)
	select {
	case result := <-eventChan:
		assert.NoError(t, result.Err, "Should receive KEDAScalerFailedOver event")
		t.Logf("Received failover event: %s", result.Event.Message)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for KEDAScalerFailedOver event")
	}

	// Verify ActiveTriggerIndex changed to 1 (secondary)
	assert.Eventually(t, func() bool {
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return so.Status.ActiveTriggerIndex != nil && *so.Status.ActiveTriggerIndex == 1
	}, 3*time.Minute, 5*time.Second, "ActiveTriggerIndex should switch to 1 (secondary)")

	// Verify scaling continues with secondary trigger (metric 15 > target 10 → replicas still >= 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 3),
		"replica count should remain >= 1 with secondary trigger")
}

func testRecoveryToPrimary(t *testing.T, kc *kubernetes.Clientset, kedaClient *kedaclientv1alpha1.KedaV1alpha1Client) {
	t.Log("--- testing recovery to primary trigger ---")

	// Start watching for KEDAScalerRecovered event
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	eventChan := StartEventWatch(
		ctx, t, kc, testNamespace, scaledObjectName, "ScaledObject",
		"KEDAScalerRecovered", corev1.EventTypeNormal,
		[]string{"Recovered from trigger 1 to trigger 0"},
		"",
	)

	// Restore primary trigger health
	so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	require.NoError(t, err)

	so.Spec.Triggers[0].Metadata["mockShouldFail"] = "false"
	_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
	require.NoError(t, err)

	t.Log("Primary trigger restored, waiting for recovery event...")

	// Wait for recovery event
	select {
	case result := <-eventChan:
		assert.NoError(t, result.Err, "Should receive KEDAScalerRecovered event")
		t.Logf("Received recovery event: %s", result.Event.Message)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for KEDAScalerRecovered event")
	}

	// Verify ActiveTriggerIndex returned to 0 (primary)
	assert.Eventually(t, func() bool {
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return so.Status.ActiveTriggerIndex != nil && *so.Status.ActiveTriggerIndex == 0
	}, 3*time.Minute, 5*time.Second, "ActiveTriggerIndex should return to 0 (primary)")
}

func TestDebouncing_FailAfterThreshold(t *testing.T) {
	t.Log("--- testing FailAfter threshold debouncing ---")

	kc := GetKubernetesClient(t)
	kedaClient := GetKedaKubernetesClient(t)
	data, templates := getTemplateData()

	CreateNamespace(t, kc, testNamespace)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for initial scaling with primary trigger
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 3),
		"should scale up with primary trigger")

	// Verify starting on primary (index 0)
	so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), *so.Status.ActiveTriggerIndex, "should start on primary")

	// Simulate 2 failures (below FailAfter=3 threshold)
	t.Log("Simulating 2 failures (below threshold)...")
	for i := 0; i < 2; i++ {
		// Make primary fail temporarily
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		require.NoError(t, err)
		so.Spec.Triggers[0].Metadata["mockShouldFail"] = "true"
		_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
		require.NoError(t, err)

		// Wait for one polling cycle (pollingInterval=5s)
		time.Sleep(10 * time.Second)

		// Restore primary (create mixed success/failure pattern)
		so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		require.NoError(t, err)
		so.Spec.Triggers[0].Metadata["mockShouldFail"] = "false"
		_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
		require.NoError(t, err)

		time.Sleep(10 * time.Second)
	}

	// Verify still on primary (no failover yet)
	so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), *so.Status.ActiveTriggerIndex, "should still be on primary after 2 failures")

	// Now cause 3 consecutive failures (meets FailAfter=3 threshold)
	t.Log("Simulating 3 consecutive failures (meets threshold)...")
	so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	require.NoError(t, err)
	so.Spec.Triggers[0].Metadata["mockShouldFail"] = "true"
	_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Wait for failover (FailAfter threshold met)
	assert.Eventually(t, func() bool {
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return so.Status.ActiveTriggerIndex != nil && *so.Status.ActiveTriggerIndex == 1
	}, 2*time.Minute, 10*time.Second, "should failover to secondary after 3 consecutive failures")

	t.Log("FailAfter threshold verified: failover occurred only after meeting threshold")
}

func TestDebouncing_RecoverAfterThreshold(t *testing.T) {
	t.Log("--- testing RecoverAfter threshold debouncing ---")

	kc := GetKubernetesClient(t)
	kedaClient := GetKedaKubernetesClient(t)
	data, templates := getTemplateData()

	CreateNamespace(t, kc, testNamespace)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for initial scaling
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 3),
		"should scale up initially")

	// Cause failover to secondary
	t.Log("Causing failover to secondary...")
	so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	require.NoError(t, err)
	so.Spec.Triggers[0].Metadata["mockShouldFail"] = "true"
	_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Wait for failover to secondary
	assert.Eventually(t, func() bool {
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return so.Status.ActiveTriggerIndex != nil && *so.Status.ActiveTriggerIndex == 1
	}, 2*time.Minute, 10*time.Second, "should failover to secondary")

	t.Log("Now on secondary trigger, testing RecoverAfter threshold...")

	// Restore primary health and simulate intermittent successes (below RecoverAfter=5)
	for i := 0; i < 3; i++ {
		// Make primary healthy
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		require.NoError(t, err)
		so.Spec.Triggers[0].Metadata["mockShouldFail"] = "false"
		_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
		require.NoError(t, err)

		time.Sleep(10 * time.Second)

		// Make it fail again (interrupt recovery)
		so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		require.NoError(t, err)
		so.Spec.Triggers[0].Metadata["mockShouldFail"] = "true"
		_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
		require.NoError(t, err)

		time.Sleep(10 * time.Second)
	}

	// Verify still on secondary (no premature recovery)
	so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), *so.Status.ActiveTriggerIndex, "should still be on secondary after intermittent successes")

	// Now restore primary permanently for 5+ consecutive successes
	t.Log("Restoring primary for 5+ consecutive successes...")
	so, err = kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	require.NoError(t, err)
	so.Spec.Triggers[0].Metadata["mockShouldFail"] = "false"
	_, err = kedaClient.ScaledObjects(testNamespace).Update(context.Background(), so, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Wait for recovery (RecoverAfter=5 threshold met)
	assert.Eventually(t, func() bool {
		so, err := kedaClient.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return so.Status.ActiveTriggerIndex != nil && *so.Status.ActiveTriggerIndex == 0
	}, 3*time.Minute, 10*time.Second, "should recover to primary after 5 consecutive successes")

	t.Log("RecoverAfter threshold verified: recovery occurred only after meeting threshold")
}

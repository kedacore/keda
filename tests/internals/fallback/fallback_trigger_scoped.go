//go:build e2e
// +build e2e

package fallback

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

const (
	triggerScopedTestName = "fallback-trigger-scoped-test"
)

var (
	triggerScopedNamespace        = fmt.Sprintf("%s-ns", triggerScopedTestName)
	triggerScopedDeploymentName   = fmt.Sprintf("%s-deployment", triggerScopedTestName)
	triggerScopedScaledObjectName = fmt.Sprintf("%s-so", triggerScopedTestName)
	primaryConfigMapName          = fmt.Sprintf("%s-primary-cm", triggerScopedTestName)
	secondaryConfigMapName        = fmt.Sprintf("%s-secondary-cm", triggerScopedTestName)
	triggerScopedMinReplicas      = 0
	triggerScopedMaxReplicas      = 10
)

type triggerScopedTemplateData struct {
	Namespace              string
	DeploymentName         string
	ScaledObjectName       string
	PrimaryConfigMapName   string
	SecondaryConfigMapName string
	MinReplicaCount        int
	MaxReplicaCount        int
	PrimaryValue           string
	SecondaryValue         string
}

const (
	triggerScopedDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.Namespace}}
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
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	primaryConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.PrimaryConfigMapName}}
  namespace: {{.Namespace}}
data:
  value: "{{.PrimaryValue}}"`

	secondaryConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.SecondaryConfigMapName}}
  namespace: {{.Namespace}}
data:
  value: "{{.SecondaryValue}}"`

	triggerScopedScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.Namespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  fallback:
    behavior: triggerScoped
    failureThreshold: 3
    replicas: 5
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
    scalingModifiers:
      target: "1"
      formula: "primary ?? secondary ?? 5"
  triggers:
  - type: kubernetes-resource
    name: primary
    metadata:
      resourceKind: ConfigMap
      resourceName: {{.PrimaryConfigMapName}}
      key: value
      targetValue: "2"
  - type: kubernetes-resource
    name: secondary
    metadata:
      resourceKind: ConfigMap
      resourceName: {{.SecondaryConfigMapName}}
      key: value
      targetValue: "2"`
)

func TestTriggerScopedFallback(t *testing.T) {
	kc := helper.GetKubernetesClient(t)
	data := triggerScopedTemplateData{
		Namespace:              triggerScopedNamespace,
		DeploymentName:         triggerScopedDeploymentName,
		ScaledObjectName:       triggerScopedScaledObjectName,
		PrimaryConfigMapName:   primaryConfigMapName,
		SecondaryConfigMapName: secondaryConfigMapName,
		MinReplicaCount:        triggerScopedMinReplicas,
		MaxReplicaCount:        triggerScopedMaxReplicas,
		PrimaryValue:           "10",
		SecondaryValue:         "8",
	}

	templates := []helper.Template{
		{Name: "triggerScopedDeploymentTemplate", Config: triggerScopedDeploymentTemplate},
		{Name: "primaryConfigMapTemplate", Config: primaryConfigMapTemplate},
		{Name: "secondaryConfigMapTemplate", Config: secondaryConfigMapTemplate},
		{Name: "triggerScopedScaledObjectTemplate", Config: triggerScopedScaledObjectTemplate},
	}

	// Create resources
	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	// Test 1: Both triggers healthy - should use primary
	testBothTriggersHealthy(t, kc, data)

	// Test 2: Primary fails - should failover to secondary
	testPrimaryTriggerFails(t, kc, data)

	// Test 3: Both triggers fail - should use static fallback (5 replicas)
	testBothTriggersFail(t, kc, data)

	// Test 4: Recovery - primary comes back
	testTriggerRecovery(t, kc, data)

	// Cleanup
	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func testBothTriggersHealthy(t *testing.T, kc *kubernetes.Clientset, data triggerScopedTemplateData) {
	t.Log("--- testing both triggers healthy (should use primary) ---")

	// Primary value is 10, target is 2, so: 10/2 = 5 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 60, 3),
		"replica count should be 5 (from primary trigger) after 3 minutes")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 30)
}

func testPrimaryTriggerFails(t *testing.T, kc *kubernetes.Clientset, data triggerScopedTemplateData) {
	t.Log("--- testing primary trigger failure (should failover to secondary) ---")

	// Delete primary ConfigMap to simulate trigger failure
	helper.KubectlDeleteWithTemplate(t, data, "primaryConfigMapTemplate", primaryConfigMapTemplate)

	// After primary fails 3 times (failureThreshold), should use secondary
	// Secondary value is 8, target is 2, so: 8/2 = 4 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, triggerScopedDeploymentName, data.Namespace, 4, 90, 3),
		"replica count should be 4 (from secondary trigger) after failover")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, triggerScopedDeploymentName, data.Namespace, 4, 30)
}

func testBothTriggersFail(t *testing.T, kc *kubernetes.Clientset, data triggerScopedTemplateData) {
	t.Log("--- testing both triggers fail (should use static fallback) ---")

	// Delete secondary ConfigMap to simulate both triggers failing
	helper.KubectlDeleteWithTemplate(t, data, "secondaryConfigMapTemplate", secondaryConfigMapTemplate)

	// After both triggers fail, should use static fallback: 5 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 90, 3),
		"replica count should be 5 (static fallback) when both triggers fail")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 30)
}

func testTriggerRecovery(t *testing.T, kc *kubernetes.Clientset, data triggerScopedTemplateData) {
	t.Log("--- testing trigger recovery (should switch back to primary) ---")

	// Recreate both ConfigMaps
	helper.KubectlApplyWithTemplate(t, data, "primaryConfigMapTemplate", primaryConfigMapTemplate)
	helper.KubectlApplyWithTemplate(t, data, "secondaryConfigMapTemplate", secondaryConfigMapTemplate)

	// After recovery, should use primary again: 10/2 = 5 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 90, 3),
		"replica count should be 5 (from primary trigger) after recovery")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, triggerScopedDeploymentName, data.Namespace, 5, 30)
}

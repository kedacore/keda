//go:build e2e
// +build e2e

package fallback

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	scalingModifiersTestName = "fallback-scaling-modifiers-test"
)

var (
	scalingModifiersNamespace        = fmt.Sprintf("%s-ns", scalingModifiersTestName)
	scalingModifiersDeploymentName   = fmt.Sprintf("%s-deployment", scalingModifiersTestName)
	scalingModifiersScaledObjectName = fmt.Sprintf("%s-so", scalingModifiersTestName)
	primaryConfigMapName             = fmt.Sprintf("%s-primary-cm", scalingModifiersTestName)
	secondaryConfigMapName           = fmt.Sprintf("%s-secondary-cm", scalingModifiersTestName)
	scalingModifiersMinReplicas      = 1
	scalingModifiersMaxReplicas      = 20
)

type scalingModifiersTemplateData struct {
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
	scalingModifiersDeploymentTemplate = `apiVersion: apps/v1
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

	scalingModifiersScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
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
    behavior: scalingModifiers
    failureThreshold: 3
    replicas: 4
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
    scalingModifiers:
      target: "2"
      formula: "primary ?? secondary ?? 8"
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

func TestScalingModifiersFallback(t *testing.T) {
	kc := helper.GetKubernetesClient(t)
	data := scalingModifiersTemplateData{
		Namespace:              scalingModifiersNamespace,
		DeploymentName:         scalingModifiersDeploymentName,
		ScaledObjectName:       scalingModifiersScaledObjectName,
		PrimaryConfigMapName:   primaryConfigMapName,
		SecondaryConfigMapName: secondaryConfigMapName,
		MinReplicaCount:        scalingModifiersMinReplicas,
		MaxReplicaCount:        scalingModifiersMaxReplicas,
		PrimaryValue:           "16",
		SecondaryValue:         "12",
	}

	templates := []helper.Template{
		{Name: "scalingModifiersDeploymentTemplate", Config: scalingModifiersDeploymentTemplate},
		{Name: "primaryConfigMapTemplate", Config: primaryConfigMapTemplate},
		{Name: "secondaryConfigMapTemplate", Config: secondaryConfigMapTemplate},
		{Name: "scalingModifiersScaledObjectTemplate", Config: scalingModifiersScaledObjectTemplate},
	}

	// Create resources
	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	// Test 1: Both triggers healthy - formula picks the first non-nil (primary)
	testBothTriggersHealthy(t, kc, data)

	// Test 2: Primary fails - formula's ?? chain falls through to secondary
	testPrimaryTriggerFails(t, kc, data)

	// Test 3: Both triggers fail - formula's trailing constant "?? 8" is used
	testBothTriggersFail(t, kc, data)

	// Test 4: Recovery - primary comes back
	testTriggerRecovery(t, kc, data)

	// Cleanup
	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func testBothTriggersHealthy(t *testing.T, kc *kubernetes.Clientset, data scalingModifiersTemplateData) {
	t.Log("--- testing both triggers healthy (should use primary) ---")

	// Primary value is 16, target is 2, so: 16/2 = 8 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, scalingModifiersDeploymentName, data.Namespace, 8, 60, 3),
		"replica count should be 8 (from primary trigger) after 3 minutes")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scalingModifiersDeploymentName, data.Namespace, 8, 30)
}

func testPrimaryTriggerFails(t *testing.T, kc *kubernetes.Clientset, data scalingModifiersTemplateData) {
	t.Log("--- testing primary trigger failure (should failover to secondary) ---")

	// Delete primary ConfigMap to simulate trigger failure
	helper.KubectlDeleteWithTemplate(t, data, "primaryConfigMapTemplate", primaryConfigMapTemplate)

	// After primary fails 3 times (failureThreshold), should use secondary
	// Secondary value is 12, target is 2, so: 12/2 = 6 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, scalingModifiersDeploymentName, data.Namespace, 6, 90, 3),
		"replica count should be 6 (from secondary trigger) after failover")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scalingModifiersDeploymentName, data.Namespace, 6, 30)
}

func testBothTriggersFail(t *testing.T, kc *kubernetes.Clientset, data scalingModifiersTemplateData) {
	t.Log("--- testing both triggers fail (formula evaluates to trailing constant 8) ---")

	// Delete secondary ConfigMap so both triggers fail. fallback.replicas (4)
	// is NOT consulted here - the formula's trailing "?? 8" still produces a
	// real value, so HPA gets 8/target(2) = 4 desired replicas.
	helper.KubectlDeleteWithTemplate(t, data, "secondaryConfigMapTemplate", secondaryConfigMapTemplate)

	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, scalingModifiersDeploymentName, data.Namespace, 4, 90, 3),
		"replica count should be 4 (formula constant 8 / target 2) when both triggers fail")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scalingModifiersDeploymentName, data.Namespace, 4, 30)
}

func testTriggerRecovery(t *testing.T, kc *kubernetes.Clientset, data scalingModifiersTemplateData) {
	t.Log("--- testing trigger recovery (should switch back to primary) ---")

	// Recreate both ConfigMaps
	helper.KubectlApplyWithTemplate(t, data, "primaryConfigMapTemplate", primaryConfigMapTemplate)
	helper.KubectlApplyWithTemplate(t, data, "secondaryConfigMapTemplate", secondaryConfigMapTemplate)

	// After recovery, should use primary again: 16/2 = 8 replicas
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, scalingModifiersDeploymentName, data.Namespace, 8, 90, 3),
		"replica count should be 8 (from primary trigger) after recovery")

	// Ensure it stays stable
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scalingModifiersDeploymentName, data.Namespace, 8, 30)
}

//go:build e2e
// +build e2e

package kubernetes_resource_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "kubernetes-resource-test"
)

var (
	testNamespace     = fmt.Sprintf("%s-ns", testName)
	deploymentName    = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName  = fmt.Sprintf("%s-so", testName)
	configMapName     = fmt.Sprintf("%s-cm", testName)
	secretName        = fmt.Sprintf("%s-secret", testName)
	configMapJSONName = fmt.Sprintf("%s-cm-json", testName)
	secretJSONName    = fmt.Sprintf("%s-secret-json", testName)
	minReplicaCount   = 0
	maxReplicaCount   = 5
)

type templateData struct {
	TestNamespace     string
	DeploymentName    string
	ScaledObjectName  string
	ConfigMapName     string
	SecretName        string
	ConfigMapJSONName string
	SecretJSONName    string
	MinReplicaCount   int
	MaxReplicaCount   int
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
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	configMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapName}}
  namespace: {{.TestNamespace}}
data:
  threshold: "0"`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  limit: "0"`

	configMapJSONTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapJSONName}}
  namespace: {{.TestNamespace}}
data:
  metrics: |
    {
      "scaling": {
        "threshold": 0
      }
    }`

	secretJSONTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretJSONName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  config: |
    {
      "limits": {
        "requests": 0
      }
    }`

	scaledObjectConfigMapTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-resource
    metadata:
      resourceKind: ConfigMap
      resourceName: {{.ConfigMapName}}
      key: threshold
      targetValue: "3"
      activationTargetValue: "1"`

	scaledObjectSecretTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-resource
    metadata:
      resourceKind: Secret
      resourceName: {{.SecretName}}
      key: limit
      targetValue: "5"
      activationTargetValue: "2"`

	scaledObjectConfigMapJSONTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-resource
    metadata:
      resourceKind: ConfigMap
      resourceName: {{.ConfigMapJSONName}}
      key: metrics
      format: json
      valueLocation: scaling.threshold
      targetValue: "4"
      activationTargetValue: "1"`

	scaledObjectSecretJSONTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-resource
    metadata:
      resourceKind: Secret
      resourceName: {{.SecretJSONName}}
      key: config
      format: json
      valueLocation: limits.requests
      targetValue: "3"
      activationTargetValue: "1"`
)

func TestScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Create kubernetes resources (namespace, deployment, configmap, secret)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// Test ConfigMap scaling
	t.Run("ConfigMap simple value", func(t *testing.T) {
		testConfigMapSimpleValue(t, kc, data)
	})

	// Test Secret scaling
	t.Run("Secret simple value", func(t *testing.T) {
		testSecretSimpleValue(t, kc, data)
	})

	// Test ConfigMap with JSON
	t.Run("ConfigMap JSON value", func(t *testing.T) {
		testConfigMapJSONValue(t, kc, data)
	})

	// Test Secret with JSON
	t.Run("Secret JSON value", func(t *testing.T) {
		testSecretJSONValue(t, kc, data)
	})
}

func testConfigMapSimpleValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test configmap simple value ---")

	// Create ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectConfigMapTemplate", scaledObjectConfigMapTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectConfigMapTemplate", scaledObjectConfigMapTemplate)

	// Should remain at 0 (value=0, activationValue=1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)

	// Update ConfigMap to activation threshold (value=1)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"1\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should still be at 0 (value=1 equals activationValue, not greater)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)

	// Update ConfigMap above activation threshold (value=2)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"2\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to 1 (value=2, targetValue=3 -> desiredReplicas = ceil[2/3] = 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Update ConfigMap to target value (value=3)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"3\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to 1 (value=3, targetValue=3 -> desiredReplicas = ceil[3/3] = 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Update ConfigMap above target value (value=9)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"9\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to 3 (value=9, targetValue=3 -> desiredReplicas = ceil[9/3] = 3)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 2),
		"replica count should be 3")

	// Update ConfigMap to very high value (value=15)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"15\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to maxReplicaCount (value=15, targetValue=3 -> desiredReplicas = ceil[15/3] = 5, capped at maxReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 2),
		"replica count should be at max")

	// Scale back down - update ConfigMap to 0
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch configmap %s -n %s --type merge -p '{\"data\":{\"threshold\":\"0\"}}'", configMapName, testNamespace))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale back to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be 0")
}

func testSecretSimpleValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test secret simple value ---")

	// Create ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectSecretTemplate", scaledObjectSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectSecretTemplate", scaledObjectSecretTemplate)

	// Should remain at 0 (value=0, activationValue=2)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)

	// Update Secret above activation threshold (value=3)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl patch secret %s -n %s --type merge -p '{\"stringData\":{\"limit\":\"3\"}}'", secretName, testNamespace))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale to 1 (value=3, targetValue=5 -> desiredReplicas = ceil[3/5] = 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Update Secret to high value (value=10)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch secret %s -n %s --type merge -p '{\"stringData\":{\"limit\":\"10\"}}'", secretName, testNamespace))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale to 2 (value=10, targetValue=5 -> desiredReplicas = ceil[10/5] = 2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2")

	// Scale back down - update Secret to 0
	_, err = ExecuteCommand(fmt.Sprintf("kubectl patch secret %s -n %s --type merge -p '{\"stringData\":{\"limit\":\"0\"}}'", secretName, testNamespace))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale back to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be 0")
}

func testConfigMapJSONValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test configmap json value ---")

	// Create ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectConfigMapJSONTemplate", scaledObjectConfigMapJSONTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectConfigMapJSONTemplate", scaledObjectConfigMapJSONTemplate)

	// Should remain at 0 (value=0, activationValue=1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)

	// Update ConfigMap JSON value (value=2)
	jsonData := `{\"scaling\":{\"threshold\":2}}`
	_, err := ExecuteCommand(fmt.Sprintf(`kubectl patch configmap %s -n %s --type merge -p '{"data":{"metrics":"%s"}}'`, configMapJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to 1 (value=2, targetValue=4 -> desiredReplicas = ceil[2/4] = 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Update ConfigMap JSON value (value=8)
	jsonData = `{\"scaling\":{\"threshold\":8}}`
	_, err = ExecuteCommand(fmt.Sprintf(`kubectl patch configmap %s -n %s --type merge -p '{"data":{"metrics":"%s"}}'`, configMapJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale to 2 (value=8, targetValue=4 -> desiredReplicas = ceil[8/4] = 2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2")

	// Scale back down
	jsonData = `{\"scaling\":{\"threshold\":0}}`
	_, err = ExecuteCommand(fmt.Sprintf(`kubectl patch configmap %s -n %s --type merge -p '{"data":{"metrics":"%s"}}'`, configMapJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update configmap - %s", err)

	// Should scale back to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be 0")
}

func testSecretJSONValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test secret json value ---")

	// Create ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectSecretJSONTemplate", scaledObjectSecretJSONTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectSecretJSONTemplate", scaledObjectSecretJSONTemplate)

	// Should remain at 0 (value=0, activationValue=1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)

	// Update Secret JSON value (value=3)
	jsonData := `{\"limits\":{\"requests\":3}}`
	_, err := ExecuteCommand(fmt.Sprintf(`kubectl patch secret %s -n %s --type merge -p '{"stringData":{"config":"%s"}}'`, secretJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale to 1 (value=3, targetValue=3 -> desiredReplicas = ceil[3/3] = 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Update Secret JSON value (value=9)
	jsonData = `{\"limits\":{\"requests\":9}}`
	_, err = ExecuteCommand(fmt.Sprintf(`kubectl patch secret %s -n %s --type merge -p '{"stringData":{"config":"%s"}}'`, secretJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale to 3 (value=9, targetValue=3 -> desiredReplicas = ceil[9/3] = 3)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 2),
		"replica count should be 3")

	// Scale back down
	jsonData = `{\"limits\":{\"requests\":0}}`
	_, err = ExecuteCommand(fmt.Sprintf(`kubectl patch secret %s -n %s --type merge -p '{"stringData":{"config":"%s"}}'`, secretJSONName, testNamespace, jsonData))
	assert.NoErrorf(t, err, "cannot update secret - %s", err)

	// Should scale back to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be 0")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:     testNamespace,
			DeploymentName:    deploymentName,
			ScaledObjectName:  scaledObjectName,
			ConfigMapName:     configMapName,
			SecretName:        secretName,
			ConfigMapJSONName: configMapJSONName,
			SecretJSONName:    secretJSONName,
			MinReplicaCount:   minReplicaCount,
			MaxReplicaCount:   maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "configMapTemplate", Config: configMapTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "configMapJSONTemplate", Config: configMapJSONTemplate},
			{Name: "secretJSONTemplate", Config: secretJSONTemplate},
		}
}

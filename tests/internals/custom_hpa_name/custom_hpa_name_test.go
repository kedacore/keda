//go:build e2e
// +build e2e

package custom_hpa_name_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	CustomHpaName    string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      run: {{.DeploymentName}}
  replicas: 1
  template:
    metadata:
      labels:
        run: {{.DeploymentName}}
    spec:
      containers:
      - name: {{.DeploymentName}}
        image: registry.k8s.io/hpa-example
        ports:
        - containerPort: 80
        imagePullPolicy: IfNotPresent
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
  minReplicaCount: 1
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
`

	scaledObjectTemplateWithCustomName = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 1
  maxReplicaCount: 1
  cooldownPeriod: 10
  advanced:
    horizontalPodAutoscalerConfig:
      name: {{.CustomHpaName}}
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
`
)

func TestCustomToDefault(t *testing.T) {
	// setup
	testName := "custom-to-default-hpa-name"
	scaledObjectName := fmt.Sprintf("%s-so", testName)
	defaultHpaName := fmt.Sprintf("keda-hpa-%s", scaledObjectName)
	customHpaName := fmt.Sprintf("%s-custom", testName)
	test(t, testName, customHpaName, scaledObjectTemplateWithCustomName, "custom",
		defaultHpaName, scaledObjectTemplate, "default")
}

func TestDefaultToCustom(t *testing.T) {
	// setup
	testName := "default-to-custom-hpa-name"
	scaledObjectName := fmt.Sprintf("%s-so", testName)
	defaultHpaName := fmt.Sprintf("keda-hpa-%s", scaledObjectName)
	customHpaName := fmt.Sprintf("%s-custom", testName)
	test(t, testName, defaultHpaName, scaledObjectTemplate, "default",
		customHpaName, scaledObjectTemplateWithCustomName, "custom")
}

func test(t *testing.T, testName string, firstHpaName string, firstSOTemplate string, firstHpaDescription string,
	secondHpaName string, secondSOTemplate string, secondHpaDescription string) {
	testNamespace := fmt.Sprintf("%s-ns", testName)
	deploymentName := fmt.Sprintf("%s-deployment", testName)
	scaledObjectName := fmt.Sprintf("%s-so", testName)
	customHpaName := fmt.Sprintf("%s-custom", testName)
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data := getTemplateData(testNamespace, deploymentName, scaledObjectName, customHpaName)
	templates := []Template{
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "firstSOTemplate", Config: firstSOTemplate},
	}

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	t.Logf("--- validate hpa is with %s name ---", firstHpaDescription)
	hpa, _ := WaitForHpaCreation(t, kc, firstHpaName, testNamespace, 60, 1)
	assert.Equal(t, firstHpaName, hpa.Name)

	t.Log("--- change hpa name ---")
	templatesCustomName := []Template{{Name: "secondSOTemplate", Config: secondSOTemplate}}
	KubectlApplyMultipleWithTemplate(t, data, templatesCustomName)

	t.Logf("--- validate new hpa is with %s name ---", secondHpaDescription)
	hpa, _ = WaitForHpaCreation(t, kc, secondHpaName, testNamespace, 60, 1)
	assert.Equal(t, secondHpaName, hpa.Name)

	t.Logf("--- validate hpa with %s name does not exists ---", firstHpaDescription)
	_, err := WaitForHpaCreation(t, kc, firstHpaName, testNamespace, 15, 1)
	assert.True(t, errors.IsNotFound(err))

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData(testNamespace string, deploymentName string, scaledObjectName string, customHpaName string) templateData {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		CustomHpaName:    customHpaName,
	}
}

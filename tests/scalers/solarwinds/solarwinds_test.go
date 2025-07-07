//go:build e2e
// +build e2e

package solarwinds_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "solarwinds-scaler-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	metricName       = "composite.kedascaler.test"
	apiToken         = os.Getenv("SOLARWINDS_API_TOKEN")
	host             = os.Getenv("SOLARWINDS_API_URL")
	maxReplicaCount  = 2
	minReplicaCount  = 0
)

type templateData struct {
	TestNamespace    string
	ScaledObjectName string
	DeploymentName   string
	MetricName       string
	APIToken         string
	Host             string
	TargetValue      string
	ActivationValue  string
	Aggregation      string
	IntervalS        string
	Filter           string
	MinReplicaCount  string
	MaxReplicaCount  string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
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
        image: nginxinc/nginx-unprivileged
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiToken: {{.APIToken}}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: apiToken
      name: {{.SecretName}}
      key: apiToken
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
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: solarwinds
    metadata:
      host: {{.Host}}
      metricName: {{.MetricName}}
      targetValue: "{{.TargetValue}}"
      activationValue: "{{.ActivationValue}}"
      aggregation: "AVG"
      intervalS: "60"
	  filter: ""
	authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			ScaledObjectName: scaledObjectName,
			DeploymentName:   deploymentName,
			MetricName:       metricName,
			APIToken:         apiToken,
			Host:             host,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// the metric returns 10, let's change the scaled object resource to force scaling out
	data := getScaledObjectTemplateData("1", "9")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleOut", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	data := getScaledObjectTemplateData("10", "15")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleIn", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getScaledObjectTemplateData(targetValue, activationValue string) templateData {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		MetricName:       metricName,
		APIToken:         apiToken,
		Host:             host,
		MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
		MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
		TargetValue:      targetValue,
		ActivationValue:  activationValue,
	}
}

//go:build e2e
// +build e2e

package solarwinds_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

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
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
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
	SecretName       string
	TriggerAuthName  string
	MetricName       string
	APITokenBase64   string
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
  apiToken: {{.APITokenBase64}}
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
	// skip test if environment variables are not set
	if apiToken == "" || host == "" {
		t.Skip("Skipping SolarWinds scaler test - SOLARWINDS_API_TOKEN or SOLARWINDS_API_URL environment variables not set")
	}

	// setup
	t.Logf("--- setting up ---")
	t.Logf("Using SolarWinds host: %s", host)
	t.Logf("Using metric name: %s", metricName)
	t.Logf("Test namespace: %s", testNamespace)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait a bit for resources to be ready
	t.Log("--- waiting for resources to be ready ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 120, 2),
		"initial replica count should be 0")

	// test scaling
	testScaleOut(t, kc)

	// Wait between scale operations to ensure clean state
	t.Log("--- waiting between scale operations ---")
	time.Sleep(5 * time.Second)

	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			ScaledObjectName: scaledObjectName,
			DeploymentName:   deploymentName,
			SecretName:       secretName,
			TriggerAuthName:  triggerAuthName,
			MetricName:       metricName,
			APITokenBase64:   base64.StdEncoding.EncodeToString([]byte(apiToken)),
			Host:             host,
			TargetValue:      "5",
			ActivationValue:  "1",
			Aggregation:      "AVG",
			IntervalS:        "60",
			Filter:           "",
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

	// Assuming the SolarWinds API returns a value like 10
	// Set targetValue=8 so that 10 > 8 will trigger scaling
	// Set activationValue=2 so that 10 > 2 will activate the scaler
	data := getScaledObjectTemplateData("8", "2")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleOut", scaledObjectTemplate)

	t.Log("--- waiting for scale out (target: 8, activation: 2) ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 120, 2),
		"replica count should be 1 after 2 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Set targetValue and activationValue higher than the expected metric value
	// If metric is ~10, setting activation=15 means 10 < 15, so not active -> scale to 0
	data := getScaledObjectTemplateData("20", "15")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleIn", scaledObjectTemplate)

	t.Log("--- waiting for scale in (target: 20, activation: 15) ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 120, 2),
		"replica count should be 0 after 2 minutes")
}

func getScaledObjectTemplateData(targetValue, activationValue string) templateData {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		SecretName:       secretName,
		TriggerAuthName:  triggerAuthName,
		MetricName:       metricName,
		APITokenBase64:   base64.StdEncoding.EncodeToString([]byte(apiToken)),
		Host:             host,
		TargetValue:      targetValue,
		ActivationValue:  activationValue,
		Aggregation:      "AVG",
		IntervalS:        "60",
		Filter:           "",
		MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
		MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
	}
}

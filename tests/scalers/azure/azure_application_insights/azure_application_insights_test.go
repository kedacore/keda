//go:build e2e
// +build e2e

package azure_application_insights_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-app-insights-test"
)

var (
	appInsightsAppID              = os.Getenv("TF_AZURE_APP_INSIGHTS_APP_ID")
	appInsightsInstrumentationKey = os.Getenv("TF_AZURE_APP_INSIGHTS_INSTRUMENTATION_KEY")
	appInsightsMetricName         = fmt.Sprintf("metric-%d", GetRandomNumber())
	appInsightsRole               = fmt.Sprintf("%s-role", testName)
	azureADClientID               = os.Getenv("TF_AZURE_SP_APP_ID")
	azureADSecret                 = os.Getenv("AZURE_SP_KEY")
	azureADTenantID               = os.Getenv("TF_AZURE_SP_TENANT")
	testNamespace                 = fmt.Sprintf("%s-ns", testName)
	secretName                    = fmt.Sprintf("%s-secret", testName)
	deploymentName                = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName               = fmt.Sprintf("%s-ta", testName)
	scaledObjectName              = fmt.Sprintf("%s-so", testName)
	minReplicaCount               = 0
	maxReplicaCount               = 2
)

type templateData struct {
	TestNamespace                 string
	SecretName                    string
	DeploymentName                string
	TriggerAuthName               string
	ScaledObjectName              string
	AzureADClientID               string
	AzureADSecret                 string
	AzureADTenantID               string
	ApplicationInsightsID         string
	ApplicationInsightsMetricName string
	ApplicationInsightsRole       string
	MinReplicaCount               string
	MaxReplicaCount               string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  applicationInsightsId: {{.ApplicationInsightsID}}
  clientId: {{.AzureADClientID}}
  clientSecret: {{.AzureADSecret}}
  tenantId: {{.AzureADTenantID}}
`

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
      - name: app-insights-scaler-test
        image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: applicationInsightsId
      name: {{.SecretName}}
      key: applicationInsightsId
    - parameter: activeDirectoryClientId
      name: {{.SecretName}}
      key: clientId
    - parameter: activeDirectoryClientPassword
      name: {{.SecretName}}
      key: clientSecret
    - parameter: tenantId
      name: {{.SecretName}}
      key: tenantId
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
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
    - type: azure-app-insights
      metadata:
        metricId: "customMetrics/{{.ApplicationInsightsMetricName}}"
        metricAggregationTimespan: "0:3"
        metricAggregationType: max
        metricFilter: cloud/roleName eq '{{.ApplicationInsightsRole}}'
        targetValue: "10"
        activationTargetValue: "20"
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, appInsightsAppID, "TF_AZURE_APP_INSIGHTS_APP_ID env variable is required for application insights tests")
	require.NotEmpty(t, appInsightsInstrumentationKey, "TF_AZURE_APP_INSIGHTS_INSTRUMENTATION_KEY env variable is required for application insights tests")
	require.NotEmpty(t, azureADClientID, "TF_AZURE_SP_APP_ID env variable is required for application insights tests")
	require.NotEmpty(t, azureADSecret, "AZURE_SP_KEY env variable is required for application insights tests")
	require.NotEmpty(t, azureADTenantID, "TF_AZURE_SP_TENANT env variable is required for application insights tests")
	client := appinsights.NewTelemetryClient(appInsightsInstrumentationKey)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc, client)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, client appinsights.TelemetryClient) {
	t.Log("--- testing activation ---")
	stopCh := make(chan struct{})
	go setMetricValue(client, 10, stopCh)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 300)
	close(stopCh)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, client appinsights.TelemetryClient) {
	t.Log("--- testing scale out ---")
	stopCh := make(chan struct{})
	go setMetricValue(client, 100, stopCh)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 5),
		"replica count should be 2 after 5 minutes")
	close(stopCh)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, client appinsights.TelemetryClient) {
	t.Log("--- testing scale in ---")
	stopCh := make(chan struct{})
	go setMetricValue(client, 0, stopCh)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be 0 after 5 minutes")
	close(stopCh)
}

func setMetricValue(client appinsights.TelemetryClient, value float64, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
			client.Context().Tags.Cloud().SetRole(appInsightsRole)
			client.TrackMetric(appInsightsMetricName, value)
			client.Channel().Flush()
			time.Sleep(time.Second * 15)
		}
	}
}

func getTemplateData() (templateData, []Template) {
	base64ClientSecret := base64.StdEncoding.EncodeToString([]byte(azureADSecret))
	base64ClientID := base64.StdEncoding.EncodeToString([]byte(azureADClientID))
	base64TenantID := base64.StdEncoding.EncodeToString([]byte(azureADTenantID))
	base64ApplicationInsightsID := base64.StdEncoding.EncodeToString([]byte(appInsightsAppID))

	return templateData{
			TestNamespace:                 testNamespace,
			SecretName:                    secretName,
			DeploymentName:                deploymentName,
			TriggerAuthName:               triggerAuthName,
			ScaledObjectName:              scaledObjectName,
			AzureADClientID:               base64ClientID,
			AzureADSecret:                 base64ClientSecret,
			AzureADTenantID:               base64TenantID,
			ApplicationInsightsID:         base64ApplicationInsightsID,
			ApplicationInsightsMetricName: appInsightsMetricName,
			ApplicationInsightsRole:       appInsightsRole,
			MinReplicaCount:               fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:               fmt.Sprintf("%v", maxReplicaCount),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

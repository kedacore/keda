//go:build e2e
// +build e2e

package openstack_metrics_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/scalers/openstack_metrics"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "openstack-metrics-test"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)

	userID    = os.Getenv("OPENSTACK_USER_ID")
	password  = os.Getenv("OPENSTACK_PASSWORD")
	projectID = os.Getenv("OPENSTACK_PROJECT_ID")
	authURL   = os.Getenv("OPENSTACK_AUTH_URL")

	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	SecretName                string
	TriggerAuthenticationName string
	ScaledObjectName          string
	UserID                    string
	Password                  string
	ProjectID                 string
	AuthURL                   string
	MetricsURL                string
	MetricID                  string
	MinReplicaCount           int
	MaxReplicaCount           int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: {{.DeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.DeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  userID: {{.UserID}}
  password: {{.Password}}
  projectID: {{.ProjectID}}
  authURL: {{.AuthURL}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: userID
    name: {{.SecretName}}
    key: userID
  - parameter: password
    name: {{.SecretName}}
    key: password
  - parameter: projectID
    name: {{.SecretName}}
    key: projectID
  - parameter: authURL
    name: {{.SecretName}}
    key: authURL
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: openstack-metrics
    metadata:
      metricsURL: {{.MetricsURL}}
      metricID: {{.MetricID}}
      aggregationMethod: mean
      granularity: 300
      threshold: 1
      activationThreshold: 5
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
)

func TestScaler(t *testing.T) {
	require.NotEmpty(t, userID, "OPENSTACK_USER_ID env variable is required")
	require.NotEmpty(t, password, "OPENSTACK_PASSWORD env variable is required")
	require.NotEmpty(t, projectID, "OPENSTACK_PROJECT_ID env variable is required")
	require.NotEmpty(t, authURL, "OPENSTACK_AUTH_URL env variable is required")

	client := openstack_metrics.CreateClient(t, authURL, userID, password, projectID)
	metricsClient := openstack_metrics.CreateMetricsClient(t, authURL, userID, password, projectID)
	metricsURL, metricID := openstack_metrics.CreateMetric(t, metricsClient, testName)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(metricsURL, metricID)

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		openstack_metrics.DeleteMetric(t, metricsClient, metricsURL, metricID)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testActivation(t, kc, client, metricsURL, metricID)
	testScaleOut(t, kc, client, metricsURL, metricID)
	testScaleIn(t, kc, client, metricsURL, metricID)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, client openstack.Client, metricsURL, metricID string) {
	t.Log("--- testing activation ---")

	openstack_metrics.PostMeasure(t, client, metricsURL, metricID, 2)

	AssertReplicaCountNotChangeDuringTimePeriod(
		t, kc, deploymentName, testNamespace, minReplicaCount, 60,
	)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, client openstack.Client, metricsURL, metricID string) {
	t.Log("--- testing scale out ---")

	openstack_metrics.PostMeasure(t, client, metricsURL, metricID, 10)

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(
			t, kc, deploymentName, testNamespace,
			maxReplicaCount, 60, 3,
		),
		"replica count should be %d after 3 minutes",
		maxReplicaCount,
	)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, client openstack.Client, metricsURL, metricID string) {
	t.Log("--- testing scale in ---")

	openstack_metrics.PostMeasure(t, client, metricsURL, metricID, 0)

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(
			t, kc, deploymentName, testNamespace,
			minReplicaCount, 60, 5,
		),
		"replica count should be %d after 5 minutes",
		minReplicaCount,
	)
}

func getTemplateData(metricsURL, metricID string) (templateData, []Template) {
	return templateData{
			TestNamespace:             testNamespace,
			DeploymentName:            deploymentName,
			SecretName:                secretName,
			TriggerAuthenticationName: triggerAuthenticationName,
			ScaledObjectName:          scaledObjectName,
			UserID:                    base64.StdEncoding.EncodeToString([]byte(userID)),
			Password:                  base64.StdEncoding.EncodeToString([]byte(password)),
			ProjectID:                 base64.StdEncoding.EncodeToString([]byte(projectID)),
			AuthURL:                   base64.StdEncoding.EncodeToString([]byte(authURL)),
			MetricsURL:                metricsURL,
			MetricID:                  metricID,
			MinReplicaCount:           minReplicaCount,
			MaxReplicaCount:           maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

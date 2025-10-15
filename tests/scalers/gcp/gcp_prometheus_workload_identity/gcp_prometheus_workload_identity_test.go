//go:build e2e
// +build e2e

package gcp_prometheus_workload_identity_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "gcp-prometheus-workload-identity-test"
)

var (
	gcpKey           = os.Getenv("TF_GCP_SA_CREDENTIALS")
	creds            = make(map[string]interface{})
	errGcpKey        = json.Unmarshal([]byte(gcpKey), &creds)
	projectID        = creds["project_id"]
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	ProjectID        string
	TestNamespace    string
	ScaledObjectName string
	DeploymentName   string
}

const (
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-gcp-credentials
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: gcp`

	// NOTE: We employ a PromQL query that consistently yields a value of 100.
	// This is due to the absence of metrics being exported to Google Managed
	// Prometheus. In this particular test case, the crucial aspect lies in
	// validating the workload identity authentication with the Google service.
	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{ .ScaledObjectName }}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: prometheus
    authenticationRef:
      name: keda-trigger-auth-gcp-credentials
    metadata:
      serverAddress: "https://monitoring.googleapis.com/v1/projects/{{.ProjectID}}/location/global/prometheus"
      query: "vector(100)"
      threshold: "50.0"
---`

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
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
---`
)

func TestScaler(t *testing.T) {
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP storage test")
	require.NoErrorf(t, errGcpKey, "Failed to load credentials from gcpKey - %s", errGcpKey)

	t.Log("--- setting up ---")

	kc := GetKubernetesClient(t)

	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	t.Log("--- assert ---")
	expectedReplicaCountNumber := 2 // as mentioned above, as the GMP returns 100 and the threshold set to 50, the expected replica count is 100 / 50 = 2
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, expectedReplicaCountNumber, 60, 1),
		"replica count should be %d after 1 minute", expectedReplicaCountNumber)

	t.Log("--- cleaning up ---")
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			ProjectID:        projectID.(string),
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

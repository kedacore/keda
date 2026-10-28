//go:build e2e
// +build e2e

package honeycomb_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "honeycomb-test"
	apiHost  = "https://api.honeycomb.io"
	dataset  = "keda-e2e"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	// apiKey is a Configuration API key with the Run Queries and Manage Queries and Columns permissions.
	apiKey = os.Getenv("HONEYCOMB_API_KEY")
	// ingestKey is an Ingest API key for the same environment.
	ingestKey = os.Getenv("HONEYCOMB_INGEST_KEY")
	// runID tags the events of this run so that concurrent or earlier runs do not affect the query.
	runID           = fmt.Sprintf("%s-%d", testName, GetRandomNumber())
	minReplicaCount = 0
	maxReplicaCount = 2
	eventsPerSecond = 2
)

type templateData struct {
	TestNamespace       string
	DeploymentName      string
	ScaledObjectName    string
	SecretName          string
	TriggerAuthName     string
	APIKey              string
	Dataset             string
	RunID               string
	MinReplicaCount     int
	MaxReplicaCount     int
	ActivationThreshold string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiKey: {{.APIKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: apiKey
    name: {{.SecretName}}
    key: apiKey
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
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

	// With eventsPerSecond events per second over a 2 minute window, COUNT reaches about 240,
	// well above threshold * maxReplicaCount.
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
  pollingInterval: 10
  cooldownPeriod: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: honeycomb
    metadata:
      dataset: {{.Dataset}}
      query: |
        {
          "calculations": [{"op": "COUNT"}],
          "filters": [{"column": "keda_run_id", "op": "=", "value": "{{.RunID}}"}],
          "time_range": 120
        }
      resultField: COUNT
      threshold: "10"
      activationThreshold: "{{.ActivationThreshold}}"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestHoneycombScaler(t *testing.T) {
	require.NotEmpty(t, apiKey, "HONEYCOMB_API_KEY env variable is required for the Honeycomb test")
	require.NotEmpty(t, ingestKey, "HONEYCOMB_INGEST_KEY env variable is required for the Honeycomb test")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after a minute", minReplicaCount)

	ctx, stopEvents := context.WithCancel(context.Background())
	defer stopEvents()
	go sendEvents(ctx, t)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	stopEvents()
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.ActivationThreshold = "1000000"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 90)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.ActivationThreshold = "0"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	// Events stop now; the 2 minute query window empties, the result becomes 0 and the workload scales to zero.
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 100, 3),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

// sendEvents sends eventsPerSecond events per second to the Honeycomb Events API until ctx is done.
func sendEvents(ctx context.Context, t *testing.T) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	url := fmt.Sprintf("%s/1/batch/%s", apiHost, dataset)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		batch := make([]map[string]any, eventsPerSecond)
		for i := range batch {
			batch[i] = map[string]any{
				"time": time.Now().UTC().Format(time.RFC3339Nano),
				"data": map[string]any{"keda_run_id": runID, "service.name": testName},
			}
		}
		body, _ := json.Marshal(batch)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			t.Logf("creating Honeycomb events request: %v", err)
			continue
		}
		req.Header.Set("X-Honeycomb-Team", ingestKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				t.Logf("sending events to Honeycomb: %v", err)
			}
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Logf("sending events to Honeycomb: unexpected status %s", resp.Status)
		}
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:       testNamespace,
		DeploymentName:      deploymentName,
		ScaledObjectName:    scaledObjectName,
		SecretName:          secretName,
		TriggerAuthName:     triggerAuthName,
		APIKey:              base64.StdEncoding.EncodeToString([]byte(apiKey)),
		Dataset:             dataset,
		RunID:               runID,
		MinReplicaCount:     minReplicaCount,
		MaxReplicaCount:     maxReplicaCount,
		ActivationThreshold: "1000000",
	}, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

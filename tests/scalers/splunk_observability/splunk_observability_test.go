//go:build e2e
// +build e2e

package splunk_observability_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "splunk-observability-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	authName         = fmt.Sprintf("%s-auth", testName)
	accessToken      = os.Getenv("SPLUNK_OBSERVABILITY_ACCESS_TOKEN")
	ingestToken      = os.Getenv("SPLUNK_OBSERVABILITY_INGEST_TOKEN")
	realm            = os.Getenv("SPLUNK_OBSERVABILITY_REALM")
	signalflowQuery  = "data('keda-test-metric').publish()"
	duration         = "10"
	maxReplicaCount  = 10
	minReplicaCount  = 1
	// highValue/lowValue drive the max/latest aggregators; highInterval/lowInterval (send
	// frequency) additionally drive the sum/count aggregators, which are insensitive to the
	// datapoint value alone.
	highValue         = 1000.0
	lowValue          = 100.0
	highInterval      = 1 * time.Second
	lowInterval       = 5 * time.Second
	highPhaseDuration = 4 * time.Minute
)

// aggregatorTestCase pairs a queryAggregator with target values tuned to the signal produced by
// sendTestMetrics, so that each aggregator both activates/scales-out during the high phase and
// scales back in during the low phase.
type aggregatorTestCase struct {
	aggregator            string
	targetValue           string
	activationTargetValue string
}

var aggregatorTestCases = []aggregatorTestCase{
	// max/latest: high phase datapoints are all 1000, low phase datapoints are all 100.
	{aggregator: "max", targetValue: "400", activationTargetValue: "1.1"},
	{aggregator: "latest", targetValue: "400", activationTargetValue: "1.1"},
	// sum: ~10 points * 1000 during high phase vs ~2 points * 100 during low phase.
	{aggregator: "sum", targetValue: "2000", activationTargetValue: "1.1"},
	// count: ~10 points during high phase (1s interval) vs ~2 points during low phase (5s interval).
	{aggregator: "count", targetValue: "5", activationTargetValue: "1.1"},
}

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	AuthName              string
	AccessToken           string
	Realm                 string
	SignalflowQuery       string
	Duration              string
	MinReplicaCount       string
	MaxReplicaCount       string
	TargetValue           string
	ActivationTargetValue string
	QueryAggregator       string
}

const (
	authTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: splunk-secrets
  namespace: {{.TestNamespace}}
data:
  accessToken: {{.AccessToken}}
  realm: {{.Realm}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-splunk-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: accessToken
    name: splunk-secrets
    key: accessToken
  - parameter: realm
    name: splunk-secrets
    key: realm
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: keda
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{.DeploymentName}}
  pollingInterval: 3
  cooldownPeriod: 1
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: splunk-observability
    metricType: Value
    metadata:
      query: data('keda-test-metric').publish()
      duration: "10"
      targetValue: "{{.TargetValue}}"
      activationTargetValue: "{{.ActivationTargetValue}}"
      queryAggregator: "{{.QueryAggregator}}" # 'min', 'max', 'avg', 'sum', 'count', 'latest'
    authenticationRef:
      name: keda-trigger-auth-splunk-secret
`
)

func sendTestMetrics(ctx context.Context, token string, realm string) {
	tStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping metrics sender")
			return
		default:
			tNow := time.Now()
			var value float64
			var interval time.Duration
			if tNow.Sub(tStart) < highPhaseDuration {
				value = highValue
				interval = highInterval
			} else {
				value = lowValue
				interval = lowInterval
			}

			body := map[string]interface{}{
				"gauge": []map[string]interface{}{
					{
						"metric": "keda-test-metric",
						"value":  value,
						"dimensions": map[string]string{
							"service": "keda-splunk-observability-scaler-test",
						},
					},
				},
			}

			jsonBody, err := json.Marshal(body)
			if err != nil {
				log.Printf("Error marshalling JSON: %v\n", err)
				continue
			}

			url := fmt.Sprintf("https://ingest.%s.signalfx.com/v2/datapoint", realm)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
			if err != nil {
				log.Printf("Error creating request: %v\n", err)
				continue
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-SF-Token", token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Error sending request: %v\n", err)
				continue
			}

			log.Printf("Sent value %.5f to SignalFx. Status: %d. Response: %s\n", value, resp.StatusCode, resp.Status)
			resp.Body.Close()

			time.Sleep(interval)
		}
	}
}

func TestSplunkObservabilityScaler(t *testing.T) {
	kc := GetKubernetesClient(t)

	for _, tc := range aggregatorTestCases {
		t.Run(tc.aggregator, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			data, templates := getTemplateData(tc)

			t.Cleanup(func() {
				DeleteKubernetesResources(t, testNamespace, data, templates)
			})

			// Start sending metrics concurrently
			go sendTestMetrics(ctx, ingestToken, realm)

			// Wait 30 seconds to ensure initial metrics are in Splunk
			t.Log("Waiting 30 seconds for initial metrics to populate in Splunk...")
			time.Sleep(30 * time.Second)

			// Create kubernetes resources
			CreateKubernetesResources(t, kc, testNamespace, data, templates)

			// Ensure nginx deployment is ready
			assert.True(t, WaitForAllPodRunningInNamespace(t, kc, testNamespace, 18, 10),
				"pods should be running after 3 minutes")

			// test scaling
			testScaleOut(t, kc)
			testScaleIn(t, kc)
		})
	}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	t.Log("waiting for 4 minutes for scale out to complete")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 10, 4, 60),
		"replica count should be 10 after 4 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	t.Log("waiting for 10 minutes for scale in to complete")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 10, 60),
		"replica count should be 4 after 10 minutes")
}

func getTemplateData(tc aggregatorTestCase) (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			ScaledObjectName:      scaledObjectName,
			AuthName:              authName,
			AccessToken:           base64.StdEncoding.EncodeToString([]byte(accessToken)),
			Realm:                 base64.StdEncoding.EncodeToString([]byte(realm)),
			SignalflowQuery:       signalflowQuery,
			Duration:              duration,
			MinReplicaCount:       fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:       fmt.Sprintf("%v", maxReplicaCount),
			TargetValue:           tc.targetValue,
			ActivationTargetValue: tc.activationTargetValue,
			QueryAggregator:       tc.aggregator,
		}, []Template{
			{Name: "authTemplate", Config: authTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

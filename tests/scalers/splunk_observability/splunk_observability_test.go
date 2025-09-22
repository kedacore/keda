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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "splunk-observability-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	authName               = fmt.Sprintf("%s-auth", testName)
	accessToken            = os.Getenv("SPLUNK_OBSERVABILITY_ACCESS_TOKEN")
	ingestToken            = os.Getenv("SPLUNK_OBSERVABILITY_INGEST_TOKEN")
	realm                  = os.Getenv("SPLUNK_OBSERVABILITY_REALM")
	signalflowQuery        = "data('keda-test-metric').publish()"
	duration               = "10"
	maxReplicaCount        = 10
	minReplicaCount        = 1
	scaleInTargetValue     = "400"
	scaleInActivationValue = "1.1"
)

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
      targetValue: "250"
      activationTargetValue: "1.1"
      queryAggregator: "max" # 'min', 'max', or 'avg'
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
			if tNow.Sub(tStart) < 3*time.Minute {
				value = 1000.0
			} else {
				value = 100.0
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

			time.Sleep(3 * time.Second)
		}
	}
}

func TestSplunkObservabilityScaler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Start sending metrics concurrently
	go sendTestMetrics(ctx, ingestToken, realm)

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Ensure nginx deployment is ready
	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, testNamespace, minReplicaCount, 120),
		"replica count should be greater than %d after 2 minutes", minReplicaCount)

	// test scaling
	testScaleOut(ctx, t, kc, testNamespace)
	testScaleIn(ctx, t, kc)
}

func getPodCount(ctx context.Context, kc *kubernetes.Clientset, namespace string) int {
	pods, err := kc.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return len(pods.Items)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, namespace string) {
	t.Log("--- testing scale out ---")
	t.Log("waiting for 3 minutes")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 10, 3, 60), "replica count should be 10 after 3 minutes")
}

func testScaleIn(ctx context.Context, t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	t.Log("waiting for 10 minutes")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 10, 60), "replica count should be 4 after 10 minutes")
}

func getTemplateData() (templateData, []Template) {
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
			TargetValue:           scaleInTargetValue,
			ActivationTargetValue: scaleInActivationValue,
		}, []Template{
			{Name: "authTemplate", Config: authTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

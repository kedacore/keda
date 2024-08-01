//go:build e2e
// +build e2e

package dynatrace_test

import (
	"bytes"
	"encoding/base64"
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
	testName = "dynatrace-test"
)

var (
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	triggerAuthName     = fmt.Sprintf("%s-ta", testName)
	dynatraceHost       = os.Getenv("DYNATRACE_HOST")
	dynatraceToken      = os.Getenv("DYNATRACE_METRICS_TOKEN")
	dynatraceInjestHost = fmt.Sprintf("%s/api/v2/metrics/ingest", dynatraceHost)
	dynatraceMetricName = fmt.Sprintf("metric-%d", GetRandomNumber())
	minReplicaCount     = 0
	maxReplicaCount     = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	TriggerAuthName  string
	SecretName       string
	DynatraceToken   string
	DynatraceHost    string
	MinReplicaCount  string
	MaxReplicaCount  string
	MetricName       string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiToken: {{.DynatraceToken}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: token
    name: {{.SecretName}}
    key: apiToken
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
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 1
  cooldownPeriod:  1
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  triggers:
    - type: dynatrace
      metadata:
        host: {{.DynatraceHost}}
        threshold: "2"
        activationThreshold: "3"
        metricSelector: "{{.MetricName}}:max"
        from: now-2m
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestDynatraceScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, dynatraceToken, "DYNATRACE_METRICS_TOKEN env variable is required for dynatrace tests")
	require.NotEmpty(t, dynatraceHost, "DYNATRACE_HOST env variable is required for dynatrace tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %s after a minute", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	stopCh := make(chan struct{})
	go setMetricValue(t, 1, stopCh)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 120)
	close(stopCh)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	stopCh := make(chan struct{})
	go setMetricValue(t, 10, stopCh)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 2 minutes", maxReplicaCount)
	close(stopCh)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	stopCh := make(chan struct{})
	go setMetricValue(t, 0, stopCh)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 2 minutes", minReplicaCount)
	close(stopCh)
}

func setMetricValue(t *testing.T, value float64, stopCh <-chan struct{}) {
	metric := fmt.Sprintf("%s %f", dynatraceMetricName, value)
	for {
		select {
		case <-stopCh:
			return
		default:
			time.Sleep(time.Second)
			req, err := http.NewRequest("POST", dynatraceInjestHost, bytes.NewBufferString(metric))
			req.Header.Add("'Content-Type", "text/plain")
			if err != nil {
				t.Log("Invalid injection request")
				continue
			}
			req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dynatraceToken))
			r, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Log("Error executing request")
				continue
			}
			defer r.Body.Close()
			if r.StatusCode != http.StatusAccepted {
				msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
				t.Log(msg)
			}
		}
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			ScaledObjectName: scaledObjectName,
			SecretName:       secretName,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
			DynatraceToken:   base64.StdEncoding.EncodeToString([]byte(dynatraceToken)),
			DynatraceHost:    dynatraceHost,
			MetricName:       dynatraceMetricName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

//go:build e2e
// +build e2e

package sumologic_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "sumologic-keda-test"
	host     = "https://api.sumologic.com"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	accessID                = os.Getenv("SUMO_LOGIC_ACCESS_ID")
	accessKey               = os.Getenv("SUMO_LOGIC_ACCESS_KEY")
	collectorURL            = os.Getenv("SUMO_LOGIC_COLLECTOR_URL")
	logPushInterval         = 5 * time.Second
	query                   = fmt.Sprintf("_sourceCategory=%s | count", testName)
	queryType               = "logs"
	resultField             = "_count"
	timeRange               = "2m"
	timeZone                = "Asia/Kolkata"
	queryAggregator         = "Max"
	minReplicaCount         = 0
	maxReplicaCount         = 2
	scaleInTargetValue      = "100000"
	scaleInActivationValue  = "100000"
	scaleOutTargetValue     = "6"
	scaleOutActivationValue = "1"
)

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	SecretName            string
	SumologicAccessID     string
	SumologicAccessIDB64  string
	SumologicAccessKey    string
	SumologicAccessKeyB64 string
	Host                  string
	Query                 string
	QueryType             string
	ResultField           string
	TimeRange             string
	TimeZone              string
	QueryAggregator       string
	MinReplicaCount       string
	MaxReplicaCount       string
	TargetValue           string
	ActivationThreshold   string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  accessID: {{.SumologicAccessIDB64}}
  accessKey: {{.SumologicAccessKeyB64}}
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
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
  pollingInterval: 5
  cooldownPeriod: 1
  fallback:
    failureThreshold: 3
    replicas: 0
  triggers:
    - type: sumologic
      metricType: AverageValue
      metadata:
        host: "{{.Host}}"
        queryType: "{{.QueryType}}"
        query: "{{.Query}}"
        resultField: "{{.ResultField}}"
        timerange: "{{.TimeRange}}"
        timezone: "{{.TimeZone}}"
        queryAggregator: "{{.QueryAggregator}}"
        threshold: "{{.TargetValue}}"
        activationThreshold: "{{.ActivationThreshold}}"
        maxRetries: "3"
      authenticationRef:
        name: keda-trigger-auth-sumologic
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-sumologic
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: accessID
      name: {{.SecretName}}
      key: accessID
    - parameter: accessKey
      name: {{.SecretName}}
      key: accessKey
`

	deploymentTemplate = `
apiVersion: apps/v1
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
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
`
)

func TestSumologicScaler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create Kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Ensure deployment is at min replica count
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 9 minutes", minReplicaCount)

	// Test scaling
	testActivation(t, kc)

	go pushDataToSumoLogic(t, ctx)
	testScaleOut(t, kc)
	cancel()

	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// Update scaled object to force scaling out
	data := getScaledObjectTemplateData(scaleOutTargetValue, scaleOutActivationValue)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleOut", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Update scaled object to force scaling in
	data := getScaledObjectTemplateData(scaleInTargetValue, scaleInActivationValue)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleIn", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func pushDataToSumoLogic(t *testing.T, ctx context.Context) {
	if collectorURL == "" {
		t.Logf("No collector URL set, failed data push to Sumo Logic")
		return
	}

	client := &http.Client{}
	ticker := time.NewTicker(logPushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := []byte(fmt.Sprintf("log generated at %s", time.Now().Format(time.RFC3339)))

			req, err := http.NewRequest("POST", collectorURL, bytes.NewBuffer(payload))
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				continue
			}

			req.Header.Set("X-Sumo-Category", testName)
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("Failed to push data: %v", err)
				continue
			}

			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				t.Logf("Non-OK response from Sumo Logic: %s", resp.Status)
			}
		}
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			ScaledObjectName:      scaledObjectName,
			SecretName:            secretName,
			SumologicAccessID:     accessID,
			SumologicAccessIDB64:  base64.StdEncoding.EncodeToString([]byte(accessID)),
			SumologicAccessKey:    accessKey,
			SumologicAccessKeyB64: base64.StdEncoding.EncodeToString([]byte(accessKey)),
			Host:                  host,
			Query:                 query,
			QueryType:             queryType,
			ResultField:           resultField,
			TimeRange:             timeRange,
			TimeZone:              timeZone,
			QueryAggregator:       queryAggregator,
			MinReplicaCount:       fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:       fmt.Sprintf("%v", maxReplicaCount),
			TargetValue:           scaleInTargetValue,
			ActivationThreshold:   scaleInActivationValue,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func getScaledObjectTemplateData(targetValue, activationThreshold string) templateData {
	return templateData{
		TestNamespace:       testNamespace,
		DeploymentName:      deploymentName,
		ScaledObjectName:    scaledObjectName,
		Host:                host,
		Query:               query,
		QueryType:           queryType,
		ResultField:         resultField,
		TimeRange:           timeRange,
		TimeZone:            timeZone,
		QueryAggregator:     queryAggregator,
		MinReplicaCount:     fmt.Sprintf("%v", minReplicaCount),
		MaxReplicaCount:     fmt.Sprintf("%v", maxReplicaCount),
		TargetValue:         targetValue,
		ActivationThreshold: activationThreshold,
	}
}

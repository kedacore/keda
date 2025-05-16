//go:build e2e
// +build e2e

package sumologic_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "sumologic-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	secretName             = fmt.Sprintf("%s-secret", testName)
	accessID               = "access-id"
	accessKey              = "access-key"
	query                  = "_sourceCategory=prod | count by _sourceHost"
	queryType              = "logs"
	resultField            = "_count"
	timeRange              = "15"
	timeZone               = "Asia/Kolkata"
	queryAggregator        = "Max"
	maxReplicaCount        = 2
	minReplicaCount        = 0
	scaleInTargetValue     = "10"
	scaleInActivationValue = "15"
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
	Query                 string
	QueryType             string
	ResultField           string
	TimeRange             string
	TimeZone              string
	QueryAggregator       string
	MinReplicaCount       string
	MaxReplicaCount       string
	TargetValue           string
	ActivationValue       string
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
  pollingInterval: 3
  cooldownPeriod: 1
  triggers:
    - type: sumologic
      metadata:
        host: "https://api.sumologic.com"
        queryType: "{{.QueryType}}"
        query: "{{.Query}}"
        resultField: "{{.ResultField}}"
        timerange: "{{.TimeRange}}"
        timezone: "{{.TimeZone}}"
        queryAggregator: "{{.QueryAggregator}}"
        threshold: "{{.TargetValue}}"
        activationValue: "{{.ActivationValue}}"
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
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create Kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Ensure deployment is at min replica count
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// Update scaled object to force scaling out
	data := getScaledObjectTemplateData("1", "9")
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
			Query:                 query,
			QueryType:             queryType,
			ResultField:           resultField,
			TimeRange:             timeRange,
			TimeZone:              timeZone,
			QueryAggregator:       queryAggregator,
			MinReplicaCount:       fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:       fmt.Sprintf("%v", maxReplicaCount),
			TargetValue:           scaleInTargetValue,
			ActivationValue:       scaleInActivationValue,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func getScaledObjectTemplateData(targetValue, activationValue string) templateData {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		Query:            query,
		QueryType:        queryType,
		ResultField:      resultField,
		TimeRange:        timeRange,
		TimeZone:         timeZone,
		QueryAggregator:  queryAggregator,
		MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
		MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
		TargetValue:      targetValue,
		ActivationValue:  activationValue,
	}
}

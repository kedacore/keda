//go:build e2e
// +build e2e

package opencost_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "opencost-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	mockOpenCostDeployment = fmt.Sprintf("%s-mock-opencost", testName)
	serviceName            = fmt.Sprintf("%s-service", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	mockOpenCostEndpoint   = fmt.Sprintf("http://%s.%s.svc.cluster.local:9003", serviceName, testNamespace)
	minReplicaCount        = 0
	maxReplicaCount        = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	MockOpenCostDeployment  string
	ServiceName             string
	ScaledObjectName        string
	MockOpenCostEndpoint    string
	MinReplicaCount         int
	MaxReplicaCount         int
	CostThreshold           string
	ActivationCostThreshold string
	MockResponseBase64      string
}

// OpenCost API mock response with cost = 150.50
var lowCostResponse = `{"code":200,"status":"success","data":[{"default":{"name":"default","totalCost":150.50,"cpuCost":50.25,"gpuCost":0,"ramCost":75.15,"pvCost":10.10,"networkCost":15.00}}]}`

const (
	// Mock OpenCost server using nginx with a static JSON response
	mockOpenCostConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.MockOpenCostDeployment}}-config
  namespace: {{.TestNamespace}}
data:
  nginx.conf: |
    events {}
    http {
      server {
        listen 9003;
        location /allocation {
          default_type application/json;
          return 200 '{"code":200,"status":"success","data":[{"default":{"name":"default","totalCost":150.50,"cpuCost":50.25,"gpuCost":0,"ramCost":75.15,"pvCost":10.10,"networkCost":15.00}}]}';
        }
      }
    }
`

	mockOpenCostDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MockOpenCostDeployment}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MockOpenCostDeployment}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MockOpenCostDeployment}}
  template:
    metadata:
      labels:
        app: {{.MockOpenCostDeployment}}
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 9003
        volumeMounts:
        - name: nginx-config
          mountPath: /etc/nginx/nginx.conf
          subPath: nginx.conf
      volumes:
      - name: nginx-config
        configMap:
          name: {{.MockOpenCostDeployment}}-config
`

	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MockOpenCostDeployment}}
  ports:
  - port: 9003
    targetPort: 9003
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 8080
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
  cooldownPeriod: 1
  triggers:
  - type: opencost
    metadata:
      serverAddress: "{{.MockOpenCostEndpoint}}"
      costThreshold: "{{.CostThreshold}}"
      activationCostThreshold: "{{.ActivationCostThreshold}}"
      costType: "totalCost"
      aggregate: "namespace"
      window: "1h"
`
)

func TestOpenCostScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for mock opencost server to be ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, mockOpenCostDeployment, testNamespace, 1, 60, 3),
		"mock opencost deployment should be ready")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test activation (cost=150.50, activationThreshold=200, should NOT activate)
	testActivation(t, kc, data)

	// test scale out (update activationThreshold to 100, cost=150.50 > 100, should scale out)
	testScaleOut(t, kc, data)

	// test scale in (update activationThreshold to 300, cost=150.50 < 300, should scale in)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, _ templateData) {
	t.Log("--- testing activation ---")
	// Cost is 150.50, activation threshold is 200, so should not activate
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	// Update activation threshold to 100 (cost 150.50 > 100, should scale)
	data.ActivationCostThreshold = "100"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	// Update activation threshold to 300 (cost 150.50 < 300, should scale in)
	data.ActivationCostThreshold = "300"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			MockOpenCostDeployment:  mockOpenCostDeployment,
			ServiceName:             serviceName,
			ScaledObjectName:        scaledObjectName,
			MockOpenCostEndpoint:    mockOpenCostEndpoint,
			MinReplicaCount:         minReplicaCount,
			MaxReplicaCount:         maxReplicaCount,
			CostThreshold:           "100",
			ActivationCostThreshold: "200",
			MockResponseBase64:      base64.StdEncoding.EncodeToString([]byte(lowCostResponse)),
		}, []Template{
			{Name: "mockOpenCostConfigMapTemplate", Config: mockOpenCostConfigMapTemplate},
			{Name: "mockOpenCostDeploymentTemplate", Config: mockOpenCostDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

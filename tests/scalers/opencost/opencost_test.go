//go:build e2e
// +build e2e

package opencost_test

import (
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
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	mockServerName       = fmt.Sprintf("%s-mock-server", testName)
	mockServerConfigMap  = fmt.Sprintf("%s-mock-config", testName)
	mockServerEndpoint   = fmt.Sprintf("http://%s.%s.svc.cluster.local:9003", mockServerName, testNamespace)
	minReplicaCount      = 0
	maxReplicaCount      = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	MockServerName          string
	MockServerConfigMap     string
	OpenCostEndpoint        string
	MinReplicaCount         int
	MaxReplicaCount         int
	CostThreshold           string
	ActivationCostThreshold string
	MockCostValue           string
}

const (
	// Mock server ConfigMap with nginx config to return OpenCost-like responses
	mockServerConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.MockServerConfigMap}}
  namespace: {{.TestNamespace}}
data:
  nginx.conf: |
    events {
      worker_connections 1024;
    }
    http {
      server {
        listen 9003;
        location /allocation {
          default_type application/json;
          return 200 '{"code":200,"status":"success","data":[{"test-namespace":{"name":"test-namespace","properties":{"cluster":"test","namespace":"test-namespace"},"window":{"start":"2024-01-01T00:00:00Z","end":"2024-01-02T00:00:00Z"},"start":"2024-01-01T00:00:00Z","end":"2024-01-02T00:00:00Z","cpuCost":10.5,"gpuCost":0,"ramCost":5.25,"pvCost":2.0,"networkCost":1.25,"totalCost":19.0}}]}';
        }
        location /health {
          return 200 'OK';
        }
      }
    }
`

	mockServerDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MockServerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MockServerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MockServerName}}
  template:
    metadata:
      labels:
        app: {{.MockServerName}}
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 9003
        volumeMounts:
        - name: config
          mountPath: /etc/nginx/nginx.conf
          subPath: nginx.conf
      volumes:
      - name: config
        configMap:
          name: {{.MockServerConfigMap}}
`

	mockServerServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.MockServerName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MockServerName}}
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
  pollingInterval: 5
  cooldownPeriod: 10
  triggers:
  - type: opencost
    metadata:
      serverAddress: "{{.OpenCostEndpoint}}"
      costThreshold: "{{.CostThreshold}}"
      activationCostThreshold: "{{.ActivationCostThreshold}}"
      costType: "totalCost"
      aggregate: "namespace"
      window: "1d"
`
)

func TestOpenCostScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateNamespace(t, kc, testNamespace)

	// Create test resources (including mock server)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for mock server to be ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, mockServerName, testNamespace, 1, 60, 3),
		"mock server should be ready")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 1 minute", minReplicaCount)

	// test scaling based on mock OpenCost metrics
	testScaling(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	DeleteNamespace(t, testNamespace)
}

func testScaling(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaling ---")

	// The mock server returns totalCost of 19.0
	// Test 1: With very high activation threshold (999999), should NOT activate
	// since 19.0 < 999999
	t.Log("--- testing no activation with high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)

	// Test 2: With low activation threshold (10), should activate and scale
	// since 19.0 > 10
	t.Log("--- testing scale out with low threshold ---")
	data.ActivationCostThreshold = "10"
	data.CostThreshold = "10"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 120, 3),
		"replica count should be %d after scaling", maxReplicaCount)

	// Test 3: With high threshold again, should scale back down
	t.Log("--- testing scale in with high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 120, 3),
		"replica count should be %d after scaling in", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			MockServerName:          mockServerName,
			MockServerConfigMap:     mockServerConfigMap,
			OpenCostEndpoint:        mockServerEndpoint,
			MinReplicaCount:         minReplicaCount,
			MaxReplicaCount:         maxReplicaCount,
			CostThreshold:           "999999",
			ActivationCostThreshold: "999999",
		}, []Template{
			{Name: "mockServerConfigMapTemplate", Config: mockServerConfigMapTemplate},
			{Name: "mockServerDeploymentTemplate", Config: mockServerDeploymentTemplate},
			{Name: "mockServerServiceTemplate", Config: mockServerServiceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

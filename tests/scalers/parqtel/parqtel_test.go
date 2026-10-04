//go:build e2e
// +build e2e

package parqtel_test

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
	testName = "parqtel-test"

	parqtelImage = "ghcr.io/parqtel/parqtel-oss:latest"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	parqtelName      = fmt.Sprintf("%s-server", testName)
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	ParqtelName      string
	ParqtelImage     string
	MinReplicaCount  int
	MaxReplicaCount  int
}

const (
	parqtelDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: parqtel
  name: {{.ParqtelName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: parqtel
  template:
    metadata:
      labels:
        app: parqtel
        type: keda-testing
    spec:
      containers:
      - name: parqtel
        image: {{ .ParqtelImage }}
        imagePullPolicy: IfNotPresent
        args: ["serve"]
        ports:
        - containerPort: 9090
        env:
        - name: PARQTEL_BIND
          value: "0.0.0.0:9090"
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
---
`

	parqtelServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.ParqtelName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: parqtel
  ports:
  - port: 9090
    targetPort: 9090
---
`

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
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
---
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
  cooldownPeriod: 1
  triggers:
  - type: parqtel
    metadata:
      serverAddress: http://{{.ParqtelName}}.{{.TestNamespace}}.svc.cluster.local:9090
      query: rate(parqtel_keda_requests_total[1m])
      threshold: '10'
      activationThreshold: '5'
`

	// generateLoadJobTemplate POSTs cumulative counter samples into Parqtel so that
	// rate(parqtel_keda_requests_total[1m]) rises above the activation threshold.
	generateLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-load-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "v=0; for i in $(seq 1 60); do v=$((v+100)); TS=$(date +%s)000000000; curl -s -X POST http://{{.ParqtelName}}.{{.TestNamespace}}.svc.cluster.local:9090/v1/metrics/json -H 'Content-Type: application/json' -d \"{\\\"resourceMetrics\\\":[{\\\"scopeMetrics\\\":[{\\\"metrics\\\":[{\\\"name\\\":\\\"parqtel_keda_requests_total\\\",\\\"sum\\\":{\\\"dataPoints\\\":[{\\\"asDouble\\\":$v,\\\"timeUnixNano\\\":\\\"$TS\\\",\\\"isMonotonic\\\":true}]}}]}]}]}\"; sleep 1; done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
`
)

// TestParqtelScaler deploys a Parqtel instance and a target workload, then verifies
// the Parqtel scaler scales the workload out on rising query values and back to zero
// once the load stops.
func TestParqtelScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources for testing (parqtel + target deployment + ScaledObject)
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, parqtelName, testNamespace, 1, 60, 3),
		"parqtel replica count should be 1 after 3 minutes")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			ParqtelName:      parqtelName,
			ParqtelImage:     parqtelImage,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, []Template{
			{Name: "parqtelDeploymentTemplate", Config: parqtelDeploymentTemplate},
			{Name: "parqtelServiceTemplate", Config: parqtelServiceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

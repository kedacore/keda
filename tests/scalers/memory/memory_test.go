//go:build e2e
// +build e2e

package memory_test

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
	testName = "memory-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	workloadDeploymentName = fmt.Sprintf("%s-workload-deployment", testName)
	minReplicas            = 0
	maxReplicas            = 5
	utilizationValue       = 45 // downScale value
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	UtilizationValue int32

	MinReplicas            string
	MaxReplicas            string
	WorkloadDeploymentName string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    deploy: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      run: {{.DeploymentName}}
  replicas: 1
  template:
    metadata:
      labels:
        run: {{.DeploymentName}}
    spec:
      containers:
      - name: {{.DeploymentName}}
        image: registry.k8s.io/hpa-example
        ports:
        - containerPort: 80
        resources:
          limits:
            memory: 100Mi
          requests:
            memory: 50Mi
        imagePullPolicy: IfNotPresent
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    run: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          policies:
          - type: Pods
            value: 1
            periodSeconds: 10
          stabilizationWindowSeconds: 0
  maxReplicaCount: 2
  minReplicaCount: 1
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
  - type: memory
    metadata:
      type: Utilization
      value: "{{.UtilizationValue}}"
`

	scaledObjectTwoTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    run: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  pollingInterval: 1
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: memory
    metadata:
      type: Utilization
      value: "{{.UtilizationValue}}"
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod={{.WorkloadDeploymentName}}'
      value: '1'
`

	workloadDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.WorkloadDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.WorkloadDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: {{.WorkloadDeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.WorkloadDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	scaleUpValue   = 1
	scaleDownValue = 45
)

func TestMemoryScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	scaleOut(t, kc, data)
	scaleToZero(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func scaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"Replica count should start out as 1")

	t.Log("--- testing scale out ---")
	t.Log("--- applying scaled object with scaled up utilization ---")

	data.UtilizationValue = int32(scaleUpValue)
	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 180, 1),
		"Replica count should scale out in next 3 minutes")

	t.Log("--- testing scale in ---")
	t.Log("--- applying scaled object with scaled down utilization ---")

	data.UtilizationValue = int32(scaleDownValue)
	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 180, 1),
		"Replica count should be 1 in next 3 minutes")
}

func scaleToZero(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale to zero ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"Replica count should be 1")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 1, 60)

	// replica count is 1 without scaleToZero metadata field

	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)

	// expect replica count to drop to 0 after updating SO with scaleToZero
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"Replica count should be 0")

	// scale external trigger out (expect replicas scale out)
	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(maxReplicas), testNamespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicas, 60, 1),
		"Replica count should be %v", maxReplicas)

	// scale external trigger in (expect replicas back to 0 -- external trigger not active)
	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(minReplicas), testNamespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicas, 60, 1),
		"Replica count should be %v", minReplicas)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:          testNamespace,
			DeploymentName:         deploymentName,
			ScaledObjectName:       scaledObjectName,
			UtilizationValue:       int32(utilizationValue),
			MinReplicas:            fmt.Sprintf("%v", minReplicas),
			MaxReplicas:            fmt.Sprintf("%v", maxReplicas),
			WorkloadDeploymentName: workloadDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
		}
}

//go:build e2e
// +build e2e

package memory_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "memory-test"
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	UtilizationValue int32
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
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
        image: k8s.gcr.io/hpa-example
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

	scaleUpValue   = 1
	scaleDownValue = 45
)

func TestMemoryScaler(t *testing.T) {
	testNamespace := fmt.Sprintf("%s-ns", testName)
	deploymentName := fmt.Sprintf("%s-deployment", testName)
	scaledObjectName := fmt.Sprintf("%s-so", testName)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(testNamespace, deploymentName, scaledObjectName, scaleUpValue)

	CreateKubernetesResources(t, kc, testNamespace, data, []Template{{Name: "deploymentTemplate", Config: deploymentTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"Replica count should start out as 1")

	t.Log("--- testing scale out ---")
	t.Log("--- applying scaled object with scaled up utilization ---")

	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 180, 1),
		"Replica count should scale out in next 3 minutes")

	t.Log("--- testing scale in ---")
	t.Log("--- applying scaled object with scaled down utilization ---")

	data, _ = getTemplateData(testNamespace, deploymentName, scaledObjectName, scaleDownValue)
	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 180, 1),
		"Replica count should be 1 in next 3 minutes")

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func getTemplateData(testNamespace string, deploymentName string, scaledObjectName string, utilizationValue int32) (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			UtilizationValue: utilizationValue,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

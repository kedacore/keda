//go:build e2e
// +build e2e

package cpu_test

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
	testName = "cpu-test"
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
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
            cpu: 500m
          requests:
            cpu: 200m
        imagePullPolicy: IfNotPresent
`

	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    run: {{.DeploymentName}}
spec:
  ports:
  - port: 80
  selector:
    run: {{.DeploymentName}}
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
          stabilizationWindowSeconds: 0
  maxReplicaCount: 2
  minReplicaCount: 1
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
  - type: cpu
    metadata:
      type: Utilization
      value: "50"
`

	triggerJob = `apiVersion: batch/v1
kind: Job
metadata:
  name: trigger-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: busybox
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 400);do wget -q -O- http://{{.DeploymentName}}.{{.TestNamespace}}.svc/;sleep 0.1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 400
  backoffLimit: 3`
)

func TestCpuScaler(t *testing.T) {
	testNamespace := fmt.Sprintf("%s-ns", testName)
	deploymentName := fmt.Sprintf("%s-deployment", testName)
	scaledObjectName := fmt.Sprintf("%s-so", testName)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(testNamespace, deploymentName, scaledObjectName)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"Replica count should start out as 1")

	t.Log("--- testing scale out ---")
	t.Log("--- applying job ---")

	templateTriggerJob := []Template{{Name: "triggerJobTemplate", Config: triggerJob}}
	KubectlApplyMultipleWithTemplate(t, data, templateTriggerJob)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 180, 1),
		"Replica count should scale out in next 3 minutes")

	t.Log("--- testing scale in ---")
	t.Log("--- deleting job ---")

	KubectlDeleteMultipleWithTemplate(t, data, templateTriggerJob)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 180, 1),
		"Replica count should be 1 in next 3 minutes")

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func getTemplateData(testNamespace string, deploymentName string, scaledObjectName string) (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

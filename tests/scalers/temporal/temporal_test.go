//go:build e2e
// +build e2e

package temporal_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

var _ = godotenv.Load("../../.env")

const (
	testName = "temporal-test"
)

var (
	testNamespace = fmt.Sprintf("%s-ns", testName)

	temporalDeploymentName = fmt.Sprintf("temporal-%s", testName)

	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s", testName)
)

type templateData struct {
	WorkFlowCommand        string
	WorkFlowIterations     int
	DeploymentName         string
	TestNamespace          string
	TemporalDeploymentName string
	ScaledObjectName       string
}

const (
	temporalServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.TemporalDeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  type: ClusterIP
  ports:
    - port: 7233
      protocol: TCP
      targetPort: 7233
  selector:
    app: temporal
`

	temporalDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.TemporalDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: temporal
spec:
  replicas: 1
  selector:
    matchLabels:
      app: temporal
  template:
    metadata:
      labels:
        app: temporal
    spec:
      containers:
        - name: temporal
          image: temporalio/admin-tools:latest
          command: ["temporal", "server", "start-dev", "--ip", "0.0.0.0"]
          ports:
            - containerPort: 7233
          livenessProbe:
            tcpSocket:
              port: 7233
            failureThreshold: 5
            initialDelaySeconds: 10
            periodSeconds: 30
            successThreshold: 1
            timeoutSeconds: 2
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: 0
  maxReplicaCount: 1
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  triggers:
  - type: temporal
    metadata:
      namespace: default
      taskQueue: "workflow_with_single_noop_activity:test"
      targetQueueSize: "2"
      activationTargetQueueSize: "3"
      endpoint: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
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
      - name: worker
        image: "temporaliotest/omes:go-ci-latest"
        imagePullPolicy: Always
        command: ["/app/temporal-omes"]
        args:
        - "run-worker"
        - "--language=go"
        - "--server-address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--run-id=test"
        - "--scenario=workflow_with_single_noop_activity"
        - "--dir-name=prepared"
`

	jobWorkFlowTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: workflow
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: workflow
        image: "temporaliotest/omes:go-ci-latest"
        imagePullPolicy: Always
        command: ["/app/temporal-omes"]
        args:
        - "{{.WorkFlowCommand}}"
        {{- if eq .WorkFlowCommand "run-scenario"}}
        - "--iterations={{.WorkFlowIterations}}"
        {{- end}}
        - "--scenario=workflow_with_single_noop_activity"
        - "--run-id=test"
        - "--server-address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
      restartPolicy: OnFailure
  backoffLimit: 10
`
)

func getTemplateData() (templateData, []Template) {
	return templateData{
			WorkFlowCommand:        "run-scenario",
			WorkFlowIterations:     2,
			TestNamespace:          testNamespace,
			TemporalDeploymentName: TemporalDeploymentName,
			ScaledObjectName:       scaledObjectName,
			DeploymentName:         deploymentName,
		}, []Template{
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

func TestTemporalScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)

	KubectlApplyWithTemplate(t, data, "temporalServiceTemplate", temporalServiceTemplate)
	KubectlApplyWithTemplate(t, data, "temporalDeploymentTemplate", temporalDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, TemporalDeploymentName, testNamespace, 1, 30, 4), "temporal is not in a ready state")

	KubectlApplyMultipleWithTemplate(t, data, templates)
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")

	KubectlApplyWithTemplate(t, data, "jobWorkFlowActivation", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlowActivation", jobWorkFlowTemplate)
	data.WorkFlowCommand = "cleanup-scenario"
	KubectlApplyWithTemplate(t, data, "jobWorkflowCleanup", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	KubectlDeleteWithTemplate(t, data, "jobWorkflowCleanup", jobWorkFlowTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	data.WorkFlowIterations = 3
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 5),
		"replica count should be %d after 5 minutes", 0)
}

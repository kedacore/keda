//go:build e2e
// +build e2e

package temporal_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

var _ = godotenv.Load("../../.env")

const (
	testName = "temporal-test"
)

var (
	testNamespace = fmt.Sprintf("%s-ns", testName)

	temporalDeploymentName = fmt.Sprintf("temporal-%s", testName)

	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = testName
)

type templateData struct {
	WorkFlowCommand        string
	WorkFlowIterations     int
	BuildID                string
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
          command: ["temporal", "server", "start-dev", "--ip", "0.0.0.0",  "--dynamic-config-value", "frontend.workerVersioningWorkflowAPIs=true", "--dynamic-config-value", "frontend.workerVersioningRuleAPIs=true"]
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
      taskQueue: "omes-test"
      targetQueueSize: "2"
      activationTargetQueueSize: "3"
      endpoint: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
      {{- if ne .BuildID "" }}
      buildId: {{.BuildID}}
    {{- end}}
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
        image: "temporaliotest/omes:go-latest"
        imagePullPolicy: Always
        command: ["/app/temporal-omes"]
        args:
        - "run-worker"
        - "--language=go"
        - "--server-address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--run-id=test"
        - "--scenario=workflow_with_single_noop_activity"
        - "--dir-name=prepared"
        {{- if ne .BuildID "" }}
        - "--build-id={{.BuildID}}"
	{{- end}}
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
        image: "temporaliotest/omes:cli-latest"
        imagePullPolicy: Always
        command: ["/app/temporal-omes"]
        args:
        - "{{.WorkFlowCommand}}"
        {{- if eq .WorkFlowCommand "run-scenario"}}
        {{- if ne .WorkFlowIterations 0 }}
        - "--iterations={{.WorkFlowIterations}}"
        {{ else }}
        - "--duration=2m"
        {{- end}}
        {{- end}}
        - "--scenario=workflow_with_single_noop_activity"
        - "--run-id=test"
        - "--server-address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
      restartPolicy: OnFailure
  backoffLimit: 10
`

	jobUpdateBuildIDTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
 name: update-worker-version
 namespace: {{.TestNamespace}}
spec:
 template:
   spec:
     containers:
     - name: workflow
       image: "temporalio/admin-tools:latest"
       imagePullPolicy: Always
       command: ["temporal"]
       args:
       - "task-queue"
       - "versioning"
       - "commit-build-id"
       - "--task-queue=omes-test"
       - "--build-id={{.BuildID}}"
       - "--yes"
       - "--force"
       - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
     restartPolicy: OnFailure
 backoffLimit: 10
`
)

func getTemplateData() (templateData, []Template) {
	return templateData{
			WorkFlowCommand:        "run-scenario",
			WorkFlowIterations:     2,
			TestNamespace:          testNamespace,
			TemporalDeploymentName: temporalDeploymentName,
			ScaledObjectName:       scaledObjectName,
			DeploymentName:         deploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func TestTemporalScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)

	KubectlApplyWithTemplate(t, data, "temporalServiceTemplate", temporalServiceTemplate)
	KubectlApplyWithTemplate(t, data, "temporalDeploymentTemplate", temporalDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, temporalDeploymentName, testNamespace, 1, 30, 4), "temporal is not in a ready state")

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 2), "deployment should exist with 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
	testWorkerVersioning(t, kc, data, templates)
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
	KubectlDeleteWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
}

func testWorkerVersioning(t *testing.T, kc *kubernetes.Clientset, data templateData, templates []Template) {
	t.Log("--- testing worker versioning ---")

	data.BuildID = "1.1.1"
	updateWorkerVersion(t, kc, data, 1)
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 2), "deployment should exist with 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	data.WorkFlowIterations = 0
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 3),
		"replica count for build id %s should be %d after 3 minutes", data.BuildID, 1)

	data.BuildID = "1.1.2"
	updateWorkerVersion(t, kc, data, 2)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 5),
		"replica count for build id %s should be %d after 5 minutes", data.BuildID, 0)

	data.DeploymentName = "temporal-worker-latest"
	data.ScaledObjectName = "temporal-worker-latest"
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "temporal-worker-latest", testNamespace, 0, 30, 2), "deployment temporal-worker-latest should exist with 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "temporal-worker-latest", testNamespace, 1, 60, 3),
		"replica count for build id %s should be %d after 3 minutes", data.BuildID, 1)
}

func updateWorkerVersion(t *testing.T, kc *kubernetes.Clientset, data templateData, numJobs int) {
	t.Log("--- updating worker version ---")

	KubectlApplyWithTemplate(t, data, "jobUpdateBuildID", jobUpdateBuildIDTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, numJobs, 60, 3), "job update-build-id count in namespace should be 1")
	assert.True(t, WaitForJobSuccess(t, kc, "update-worker-version", testNamespace, 3, 30), "job update-build-id should be successful")
	KubectlDeleteWithTemplate(t, data, "jobUpdateBuildID", jobUpdateBuildIDTemplate)
}

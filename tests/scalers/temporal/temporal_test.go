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
	WorkFlowCommand              string
	WorkFlowIterations           int
	BuildID                      string
	DeploymentName               string
	TemporalWorkerDeploymentName string
	TestNamespace                string
	TemporalDeploymentName       string
	ScaledObjectName             string
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
          command: ["temporal", "server", "start-dev", "--ip", "0.0.0.0", "--dynamic-config-value", "frontend.workerVersioningWorkflowAPIs=true", "--dynamic-config-value", "frontend.workerVersioningRuleAPIs=true", "--dynamic-config-value", "frontend.enableDeployments=true", "--dynamic-config-value", "system.enableDeploymentVersions=true"]
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

	deploymentVersionWorkerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}-deploy-ver
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}-deploy-ver
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}-deploy-ver
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}-deploy-ver
    spec:
      containers:
      - name: worker
        image: "temporal-deployment-worker:latest"
        imagePullPolicy: Never
        args:
        - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--task-queue=omes-test"
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
        - "--build-id={{.BuildID}}"
`

	deploymentVersionWorkerScaleUpTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}-deploy-ver
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}-deploy-ver
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.DeploymentName}}-deploy-ver
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}-deploy-ver
    spec:
      containers:
      - name: worker
        image: "temporal-deployment-worker:latest"
        imagePullPolicy: Never
        args:
        - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--task-queue=omes-test"
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
        - "--build-id={{.BuildID}}"
`

	scaledObjectDeploymentVersionTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}-deploy-ver
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}-deploy-ver
  pollingInterval: 5
  cooldownPeriod: 10
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
      endpoint: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
      workerDeploymentName: {{.TemporalWorkerDeploymentName}}
      workerDeploymentBuildId: {{.BuildID}}
`

	jobSetCurrentVersionTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: set-current-version
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: set-version
        image: "temporalio/admin-tools:latest"
        imagePullPolicy: Always
        command: ["temporal"]
        args:
        - "worker"
        - "deployment"
        - "set-current-version"
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
        - "--build-id={{.BuildID}}"
        - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--yes"
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
	testWorkerVersioning(t, kc, data)
	testDeploymentVersion(t, kc, data)
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

func testWorkerVersioning(t *testing.T, kc *kubernetes.Clientset, data templateData) {
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

// testDeploymentVersion tests scaling with Worker Deployment Version versioning.
// Flow: start worker (1 replica) to register the deployment version with the
// server, set it as current, scale to 0, submit workflows to create backlog,
// then let KEDA scale back up.
func testDeploymentVersion(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing deployment version scaling ---")

	data.DeploymentName = deploymentName
	data.ScaledObjectName = scaledObjectName
	data.BuildID = "v2.0.0"
	data.TemporalWorkerDeploymentName = "omes-deployment"

	// Step 1: Start worker with 1 replica to register the deployment version
	KubectlApplyWithTemplate(t, data, "deploymentVersionWorkerScaleUpTemplate", deploymentVersionWorkerScaleUpTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName+"-deploy-ver", testNamespace, 1, 60, 3),
		"deployment version worker should start with 1 replica")

	// Step 2: Set this version as current
	KubectlApplyWithTemplate(t, data, "jobSetCurrentVersionTemplate", jobSetCurrentVersionTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "set-current-version", testNamespace, 3, 30),
		"set-current-version job should succeed")
	KubectlDeleteWithTemplate(t, data, "jobSetCurrentVersionTemplate", jobSetCurrentVersionTemplate)

	// Step 3: Scale worker back to 0 and apply the ScaledObject
	KubectlApplyWithTemplate(t, data, "deploymentVersionWorkerTemplate", deploymentVersionWorkerTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName+"-deploy-ver", testNamespace, 0, 60, 2),
		"deployment should scale to 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectDeploymentVersionTemplate", scaledObjectDeploymentVersionTemplate)

	// Step 4: Submit workflows to create backlog → scale out
	data.WorkFlowCommand = "run-scenario"
	data.WorkFlowIterations = 3
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName+"-deploy-ver", testNamespace, 1, 60, 3),
		"replica count for deployment %s build %s should be 1 after 3 minutes", data.TemporalWorkerDeploymentName, data.BuildID)

	// Step 5: Scale in after workflows drain
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName+"-deploy-ver", testNamespace, 0, 60, 5),
		"replica count for deployment %s build %s should be 0 after 5 minutes", data.TemporalWorkerDeploymentName, data.BuildID)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)

	KubectlDeleteWithTemplate(t, data, "scaledObjectDeploymentVersionTemplate", scaledObjectDeploymentVersionTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentVersionWorkerTemplate", deploymentVersionWorkerTemplate)
}

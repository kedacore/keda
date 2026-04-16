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
          command: ["temporal", "server", "start-dev", "--ip", "0.0.0.0", "--dynamic-config-value", "frontend.workerVersioningWorkflowAPIs=true", "--dynamic-config-value", "frontend.workerVersioningRuleAPIs=true", "--dynamic-config-value", "frontend.workerDeploymentAPIs=true", "--dynamic-config-value", "frontend.enableDeploymentVersions=true"]
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

	workerDeploymentTemplate = `
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
        {{- if ne .TemporalWorkerDeploymentName "" }}
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
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

	scaledObjectUnversionedTemplate = `
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
`

	scaledObjectBuildIDTemplate = `
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
      endpoint: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
      workerVersioningType: build-id
      buildId: {{.BuildID}}
`

	scaledObjectDeploymentVersionTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
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
      workerVersioningType: deployment
      deploymentName: {{.TemporalWorkerDeploymentName}}
      buildId: {{.BuildID}}
`

	deploymentVersionWorkerTemplate = `
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
        image: "temporal-deployment-worker:latest"
        imagePullPolicy: Never
        args:
        - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--task-queue=omes-test"
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
        - "--build-id={{.BuildID}}"
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

func TestTemporalScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Deploy shared Temporal server
	KubectlApplyWithTemplate(t, data, "temporalServiceTemplate", temporalServiceTemplate)
	KubectlApplyWithTemplate(t, data, "temporalDeploymentTemplate", temporalDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, temporalDeploymentName, testNamespace, 1, 30, 4), "temporal is not in a ready state")

	testUnversioned(t, kc, data)
	testBuildID(t, kc, data)
	// TODO: Enable once Temporal dev server populates task queue stats in
	// DescribeWorkerDeploymentVersion responses. The API is reachable and the
	// test infrastructure (custom worker image, set-current-version) works,
	// but the dev server returns 0 backlog for deployment versions.
	// testDeploymentVersion(t, kc, data)
}

// testUnversioned tests scaling with no worker versioning.
// Covers: activation threshold, scale out (0→1), scale in (1→0).
func testUnversioned(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing unversioned scaling ---")

	data.DeploymentName = deploymentName + "-unversioned"
	data.ScaledObjectName = scaledObjectName + "-unversioned"
	data.BuildID = ""
	data.TemporalWorkerDeploymentName = ""

	KubectlApplyWithTemplate(t, data, "workerDeploymentTemplate", workerDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 30, 2), "deployment should exist with 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectUnversionedTemplate", scaledObjectUnversionedTemplate)
	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "scaledObjectUnversionedTemplate", scaledObjectUnversionedTemplate)
		KubectlDeleteWithTemplate(t, data, "workerDeploymentTemplate", workerDeploymentTemplate)
	})

	// Activation: 2 workflows is below activationTargetQueueSize=3, should not scale
	data.WorkFlowCommand = "run-scenario"
	data.WorkFlowIterations = 2
	KubectlApplyWithTemplate(t, data, "jobWorkFlowActivation", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, data.DeploymentName, testNamespace, 0, 60)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlowActivation", jobWorkFlowTemplate)

	// Clean up activation workflows
	data.WorkFlowCommand = "cleanup-scenario"
	KubectlApplyWithTemplate(t, data, "jobWorkflowCleanup", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	KubectlDeleteWithTemplate(t, data, "jobWorkflowCleanup", jobWorkFlowTemplate)

	// Scale out: 3 workflows exceeds threshold, should scale 0→1
	data.WorkFlowCommand = "run-scenario"
	data.WorkFlowIterations = 3
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 1, 60, 3),
		"replica count should be 1 after 3 minutes")

	// Scale in: after workflows drain, should scale 1→0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 60, 5),
		"replica count should be 0 after 5 minutes")
	KubectlDeleteWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
}

// testBuildID tests scaling with build-id worker versioning.
// Covers: scale out (0→1), scale in (1→0) for a specific build ID.
func testBuildID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing build-id scaling ---")

	data.DeploymentName = deploymentName + "-buildid"
	data.ScaledObjectName = scaledObjectName + "-buildid"
	data.BuildID = "1.0.0"
	data.TemporalWorkerDeploymentName = ""

	// Register the build ID on the task queue before scaling
	commitBuildID(t, kc, data)

	KubectlApplyWithTemplate(t, data, "workerDeploymentTemplate", workerDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 30, 2), "deployment should exist with 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectBuildIDTemplate", scaledObjectBuildIDTemplate)
	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "scaledObjectBuildIDTemplate", scaledObjectBuildIDTemplate)
		KubectlDeleteWithTemplate(t, data, "workerDeploymentTemplate", workerDeploymentTemplate)
	})

	// Scale out
	data.WorkFlowCommand = "run-scenario"
	data.WorkFlowIterations = 3
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 1, 60, 3),
		"replica count for build-id %s should be 1 after 3 minutes", data.BuildID)

	// Scale in
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 60, 5),
		"replica count for build-id %s should be 0 after 5 minutes", data.BuildID)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
}

// testDeploymentVersion tests scaling with deployment version worker versioning.
// Covers: scale out (0→1), scale in (1→0) for a specific deployment version.
//
// Flow: start worker (1 replica) to register the deployment version with the
// server, set it as current, scale to 0, submit workflows to create backlog,
// then let KEDA scale back up.
func testDeploymentVersion(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing deployment version scaling ---")

	data.DeploymentName = deploymentName + "-deploy-ver"
	data.ScaledObjectName = scaledObjectName + "-deploy-ver"
	data.BuildID = "v2.0.0"
	data.TemporalWorkerDeploymentName = "omes-deployment"

	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "scaledObjectDeploymentVersionTemplate", scaledObjectDeploymentVersionTemplate)
		KubectlDeleteWithTemplate(t, data, "deploymentVersionWorkerTemplate", deploymentVersionWorkerTemplate)
	})

	// Step 1: Start worker with 1 replica to register the deployment version
	KubectlApplyWithTemplate(t, data, "deploymentVersionWorkerTemplate", deploymentVersionWorkerTemplate)
	// Temporarily scale to 1 so the worker registers with the server
	KubectlApplyWithTemplate(t, data, "deploymentVersionWorkerTemplate-scale", `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 1
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
        image: "temporal-deployment-worker:latest"
        imagePullPolicy: Never
        args:
        - "--address={{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233"
        - "--task-queue=omes-test"
        - "--deployment-name={{.TemporalWorkerDeploymentName}}"
        - "--build-id={{.BuildID}}"
`)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 1, 60, 3),
		"deployment version worker should start with 1 replica")

	// Step 2: Set this version as current
	KubectlApplyWithTemplate(t, data, "jobSetCurrentVersionTemplate", jobSetCurrentVersionTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "set-current-version", testNamespace, 3, 30),
		"set-current-version job should succeed")
	KubectlDeleteWithTemplate(t, data, "jobSetCurrentVersionTemplate", jobSetCurrentVersionTemplate)

	// Step 3: Scale worker back to 0 and apply the ScaledObject
	KubectlApplyWithTemplate(t, data, "deploymentVersionWorkerTemplate", deploymentVersionWorkerTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 60, 2),
		"deployment should scale to 0 replicas")
	KubectlApplyWithTemplate(t, data, "scaledObjectDeploymentVersionTemplate", scaledObjectDeploymentVersionTemplate)

	// Step 4: Submit workflows to create backlog → scale out
	data.WorkFlowCommand = "run-scenario"
	data.WorkFlowIterations = 3
	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 1, 60, 3),
		"replica count for deployment %s build %s should be 1 after 3 minutes", data.TemporalWorkerDeploymentName, data.BuildID)

	// Step 5: Scale in after workflows drain
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, testNamespace, 0, 60, 5),
		"replica count for deployment %s build %s should be 0 after 5 minutes", data.TemporalWorkerDeploymentName, data.BuildID)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			WorkFlowCommand:        "run-scenario",
			WorkFlowIterations:     2,
			TestNamespace:          testNamespace,
			TemporalDeploymentName: temporalDeploymentName,
			ScaledObjectName:       scaledObjectName,
			DeploymentName:         deploymentName,
		}, []Template{
			{Name: "temporalServiceTemplate", Config: temporalServiceTemplate},
			{Name: "temporalDeploymentTemplate", Config: temporalDeploymentTemplate},
		}
}

const jobCommitBuildIDTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: commit-build-id
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: commit
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

func commitBuildID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- committing build ID ---")
	KubectlApplyWithTemplate(t, data, "jobCommitBuildIDTemplate", jobCommitBuildIDTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "commit-build-id", testNamespace, 3, 30), "job commit-build-id should be successful")
	KubectlDeleteWithTemplate(t, data, "jobCommitBuildIDTemplate", jobCommitBuildIDTemplate)
}

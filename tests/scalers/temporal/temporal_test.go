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

	TemporalDeploymentName = fmt.Sprintf("temporal-%s", testName)

	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s", testName)
)

type templateData struct {
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
          command: ["bash", "-c"]
          args:
            - |
              temporal server start-dev --namespace v2 --ip 0.0.0.0
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
      queueName: hello-task-queue
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
      - name: nginx
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
`

	jobWorkerTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: worker
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: worker
        image: "prajithp/temporal-sample:1.0.0"
        imagePullPolicy: Always
        env:
        - name: TEMPORAL_ADDR
          value: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
        - name: TEMPORAL_NAMESPACE
          value: default
        - name: MODE
          value: WORKER
      restartPolicy: OnFailure
  backoffLimit: 4
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
        image: "prajithp/temporal-sample:1.0.0"
        imagePullPolicy: Always
        env:
        - name: TEMPORAL_ADDR
          value: {{.TemporalDeploymentName}}.{{.TestNamespace}}.svc.cluster.local:7233
      restartPolicy: OnFailure
  backoffLimit: 4
`
)

func getTemplateData() (templateData, []Template) {
	return templateData{

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

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 180)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlowActivation", jobWorkFlowTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	KubectlApplyWithTemplate(t, data, "jobWorker", jobWorkerTemplate)
	// workflow is already waiting for response from worker
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 2, 60, 3), "job count in namespace should be 2")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 5),
		"replica count should be %d after 5 minutes", 0)

}

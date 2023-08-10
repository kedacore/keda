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
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	PgDeploymentName = fmt.Sprintf("postgresql-%s", testName)

	TemporalDeploymentName = fmt.Sprintf("temporal-%s", testName)

	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s", testName)
	pgUsername       = "test-user"
	pgRootPassword   = "some-test-password"
)

type templateData struct {
	DeploymentName         string
	TestNamespace          string
	PgDeploymentName       string
	TemporalDeploymentName string
	ScaledObjectName       string
	PgRootUserName         string
	PgRootPassword         string
	ItemsToWrite           int
}

const (
	pgServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.PgDeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: postgresql
  ports:
    - name: psql
      protocol: TCP
      port: 5432
      targetPort: 5432
`
	pgDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.PgDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: postgresql
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
        - name: psql
          image: bitnami/postgresql:13
          env:
            - name: POSTGRESQL_USERNAME
              value: {{.PgRootUserName}}
            - name: POSTGRESQL_PASSWORD
              value: {{.PgRootPassword}}
          ports:
            - containerPort: 5432
          livenessProbe:
            tcpSocket:
              port: 5432
            initialDelaySeconds: 5
            timeoutSeconds: 5
          readinessProbe:
            tcpSocket:
              port: 5432
            initialDelaySeconds: 5
            timeoutSeconds: 5
`
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
      - env:
        - name: AUTO_SETUP
          value: "true"
        - name: DB
          value: postgresql
        - name: DB_PORT
          value: "5432"
        - name: POSTGRES_USER
          value: {{.PgRootUserName}}
        - name: POSTGRES_PWD
          value: {{.PgRootPassword}}
        - name: POSTGRES_SEEDS
          value: {{.PgDeploymentName}}.{{.TestNamespace}}.svc.cluster.local
        image: temporalio/auto-setup:1.20.1
        imagePullPolicy: IfNotPresent
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -ec
            - test $(ps -ef | grep -v grep | grep temporal-server | wc -l) -eq 1
          failureThreshold: 3
          initialDelaySeconds: 5
          periodSeconds: 30
          successThreshold: 1
          timeoutSeconds: 1
        name: temporal
        ports:
        - containerPort: 7233
          protocol: TCP
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -ec
            - test $(ps -ef | grep -v grep | grep temporal-server | wc -l) -eq 1
          failureThreshold: 3
          initialDelaySeconds: 5
          periodSeconds: 30
          successThreshold: 1
          timeoutSeconds: 1
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
      workflowName: SayHello
      activityName: say_hello
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

	jobWorkeFlowTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: workerflow
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: workerflow
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
			PgDeploymentName:       PgDeploymentName,
			ScaledObjectName:       scaledObjectName,
			PgRootUserName:         pgUsername,
			PgRootPassword:         pgRootPassword,
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

	KubectlApplyWithTemplate(t, data, "pgDeploymentTemplate", pgDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, PgDeploymentName, testNamespace, 1, 30, 2), "postgresql is not in a ready state")
	KubectlApplyWithTemplate(t, data, "pgServiceTemplate", pgServiceTemplate)

	KubectlApplyWithTemplate(t, data, "temporalServiceTemplate", temporalServiceTemplate)
	KubectlApplyWithTemplate(t, data, "temporalDeploymentTemplate", temporalDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, TemporalDeploymentName, testNamespace, 1, 30, 4), "temporal is not in a ready state")
	KubectlApplyWithTemplate(t, data, "temporalServiceTemplate", temporalServiceTemplate)

	KubectlApplyMultipleWithTemplate(t, data, templates)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")

	KubectlApplyWithTemplate(t, data, "jobWorkFlowActivation", jobWorkeFlowTemplate)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 3), "job count in namespace should be 1")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 180)
	KubectlDeleteWithTemplate(t, data, "jobWorkFlowActivation", jobWorkeFlowTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	KubectlApplyWithTemplate(t, data, "jobWorkFlow", jobWorkeFlowTemplate)
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

//go:build e2e
// +build e2e

package cpu_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "cpu-test"
)

var (
	minReplicas            = 0
	maxReplicas            = 5
	workloadDeploymentName = fmt.Sprintf("%s-workload-deployment", testName)
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	hpaName                = fmt.Sprintf("keda-hpa-%s-so", testName)
)

type templateData struct {
	TestNamespace          string
	DeploymentName         string
	ScaledObjectName       string
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
  - type: cpu
    metadata:
      type: Utilization
      value: "50"
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

	// The default metrics-server window is 30s, and that's what keda is used to, but some platforms use things like
	// prometheus-adapter, and have the window tuned to a larger window of say 5m. In that case it takes 5 minutes before
	// the HPA can even start scaling, and as a result we'll fail this test unless we wait for the metrics before we start.
	// We'd read the window straight from the metrics-server config, but we'd have to know too much about unusual configurations,
	// so we just wait up to 10 minutes for the metrics (wherever they're coming from) before we proceed with the test.
	require.True(t, WaitForHPAMetricsToPopulate(t, kc, hpaName, testNamespace, 120, 5),
		"HPA should populate metrics within 10 minutes")

	t.Log("--- testing scale out ---")
	t.Log("--- applying job ---")

	KubectlReplaceWithTemplate(t, data, "triggerJobTemplate", triggerJob)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 180, 1),
		"Replica count should scale out in next 3 minutes")

	t.Log("--- testing scale in ---")
	t.Log("--- deleting job ---")

	KubectlDeleteWithTemplate(t, data, "triggerJobTemplate", triggerJob)

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
			MinReplicas:            fmt.Sprintf("%v", minReplicas),
			MaxReplicas:            fmt.Sprintf("%v", maxReplicas),
			WorkloadDeploymentName: workloadDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
		}
}

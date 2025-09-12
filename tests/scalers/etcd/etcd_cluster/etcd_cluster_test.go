//go:build e2e
// +build e2e

package etcd_cluster_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "etcd-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	jobName          = fmt.Sprintf("%s-job", testName)
	etcdClientName   = fmt.Sprintf("%s-client", testName)
	etcdEndpoints    = fmt.Sprintf("etcd-0.etcd-headless.%s:2379,etcd-1.%s:2379,etcd-2.etcd-headless.%s:2379", testNamespace, testNamespace, testNamespace)
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	JobName          string
	ScaledObjectName string
	MinReplicaCount  int
	MaxReplicaCount  int
	EtcdName         string
	EtcdClientName   string
	EtcdEndpoints    string
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: my-app
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
`
	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 15
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      name: keda-hpa-etcd-scaledobject
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
    - type: etcd
      metadata:
        endpoints: {{.EtcdEndpoints}}
        watchKey: var
        value: '1.5'
        activationValue: '5'
        watchProgressNotifyInterval: '10'
`
	etcdClientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.EtcdClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.EtcdClientName}}
    image: gcr.io/etcd-development/etcd:v3.4.10
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "etcdClientTemplate", etcdClientTemplate)
		RemoveCluster(t, kc)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})
	CreateNamespace(t, kc, testNamespace)

	// Create Etcd Cluster
	KubectlApplyWithTemplate(t, data, "etcdClientTemplate", etcdClientTemplate)
	InstallCluster(t, kc)
	setVarValue(t, 0)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)

	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	setVarValue(t, 4)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	setVarValue(t, 9)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	setVarValue(t, 0)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			JobName:          jobName,
			EtcdName:         testName,
			EtcdClientName:   etcdClientName,
			EtcdEndpoints:    etcdEndpoints,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func setVarValue(t *testing.T, value int) {
	_, _, err := ExecCommandOnSpecificPod(t, etcdClientName, testNamespace, fmt.Sprintf(`etcdctl put var %d --endpoints=%s`, value, etcdEndpoints))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func InstallCluster(t *testing.T, kc *kubernetes.Clientset) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm upgrade --install --set persistence.enabled=false --set resourcesPreset=none --set auth.rbac.create=false --set replicaCount=3 --namespace %s --wait etcd oci://registry-1.docker.io/bitnamicharts/etcd`,
		testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func RemoveCluster(t *testing.T, kc *kubernetes.Clientset) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm delete --namespace %s --wait etcd`,
		testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

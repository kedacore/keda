//go:build e2e
// +build e2e

package etcd_cluster_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	etcd "github.com/kedacore/keda/v2/tests/scalers/etcd/helper"
)

const (
	testName = "etcd-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	jobName          = fmt.Sprintf("%s-job", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	JobName          string
	ScaledObjectName string
	EtcdName         string
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
        image: nginxinc/nginx-unprivileged
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
  pollingInterval: 30
  cooldownPeriod: 5
  advanced:
    horizontalPodAutoscalerConfig:
      name: keda-hpa-etcd-scaledobject
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
    - type: etcd
      metadata:
        endpoints: {{.EtcdName}}-0.etcd-headless.{{.TestNamespace}}:2379,{{.EtcdName}}-1.etcd-headless.{{.TestNamespace}}:2379,{{.EtcdName}}-2.etcd-headless.{{.TestNamespace}}:2379
        watchKey: var
        value: '1.5'
        watchProgressNotifyInterval: '10'
`
	insertJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: etcd
        image: gcr.io/etcd-development/etcd:v3.4.20
        imagePullPolicy: IfNotPresent
        command:
        - sh
        - -c
        - "/usr/local/bin/etcdctl put var 9 --endpoints=http://{{.EtcdName}}-0.etcd-headless.{{.TestNamespace}}:2380,http://{{.EtcdName}}-1.etcd-headless.{{.TestNamespace}}:2380,http://{{.EtcdName}}-2.etcd-headless.{{.TestNamespace}}:2380"
      restartPolicy: Never
  backoffLimit: 4
`
	deleteJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: etcd
        image: gcr.io/etcd-development/etcd:v3.4.20
        imagePullPolicy: IfNotPresent
        command:
        - sh
        - -c
        - "/usr/local/bin/etcdctl put var 0 --endpoints=http://{{.EtcdName}}-0.etcd-headless.{{.TestNamespace}}:2380,http://{{.EtcdName}}-1.etcd-headless.{{.TestNamespace}}:2380,http://{{.EtcdName}}-2.etcd-headless.{{.TestNamespace}}:2380"
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)

	// Create kubernetes resources for testing
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Create Etcd Cluster
	etcd.InstallCluster(t, kc, testName, testNamespace)

	testActivation(t, kc, data)
	testScaleIn(t, kc, data)
	testScaleOut(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 10)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	KubectlApplyWithTemplate(t, data, "insertJobTemplate", insertJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 6, 60, 3),
		"replica count should be %d after 3 minutes", 6)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "deleteJobTemplate", deleteJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 3),
		"replica count should be %d after 3 minutes", 0)
}

var data = templateData{
	TestNamespace:    testNamespace,
	DeploymentName:   deploymentName,
	ScaledObjectName: scaledObjectName,
	JobName:          jobName,
	EtcdName:         testName,
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

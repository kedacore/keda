//go:build e2e
// +build e2e

package disruption_test

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "disruption-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName = "monitored-deployment"
	sutDeploymentName       = "sut-deployment-%d"
	scaledObjectName        = "so-%d"
	kedaNamespace           = "keda"
	operatorLabelSelector   = "app=keda-operator"
	msLabelSelector         = "app=keda-metrics-apiserver"
	operatorLogName         = fmt.Sprintf("%s-operator", testName)
	msLogName               = fmt.Sprintf("%s-metrics-server", testName)
	scaledObjectCount       = 5
	minReplicaCount         = 0
	maxReplicaCount         = 4
)

type templateData struct {
	TestNamespace           string
	MonitoredDeploymentName string
	SutDeploymentName       string
	ScaledObjectName        string
	MinReplicaCount         int
	MaxReplicaCount         int
}

const (
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	sutDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.SutDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.SutDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: {{.SutDeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.SutDeploymentName}}
    spec:
      containers:
      - name: nginx
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.SutDeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{ .MinReplicaCount }}
  maxReplicaCount: {{ .MaxReplicaCount }}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod=workload-test'
      value: '1'`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, _ := getTemplateData()
	CreateNamespace(t, kc, testNamespace)
	monitoredDeployment := []Template{{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate}}
	KubectlApplyMultipleWithTemplate(t, data, monitoredDeployment)
	for i := 0; i < scaledObjectCount; i++ {
		data.ScaledObjectName = fmt.Sprintf(scaledObjectName, i)
		data.SutDeploymentName = fmt.Sprintf(sutDeploymentName, i)
		sutDeployment := []Template{{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate}}
		scaledObject := []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}}
		KubectlApplyMultipleWithTemplate(t, data, sutDeployment)
		KubectlApplyMultipleWithTemplate(t, data, scaledObject)
	}

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, monitoredDeployment)
	for i := 0; i < scaledObjectCount; i++ {
		data.ScaledObjectName = fmt.Sprintf(scaledObjectName, i)
		data.SutDeploymentName = fmt.Sprintf(sutDeploymentName, i)
		sutDeployment := []Template{{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate}}
		scaledObject := []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}}
		KubectlDeleteMultipleWithTemplate(t, data, sutDeployment)
		KubectlDeleteMultipleWithTemplate(t, data, scaledObject)
	}
	DeleteNamespace(t, testNamespace)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to maxReplicaCount - 2 replicas
	replicas := maxReplicaCount - 2
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	saveLogs(t, kc, operatorLogName, operatorLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	var wg sync.WaitGroup
	wg.Add(scaledObjectCount)
	for i := 0; i < scaledObjectCount; i++ {
		go func(index int) {
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, fmt.Sprintf(sutDeploymentName, index), testNamespace, replicas, 60, 3),
				fmt.Sprintf("replica count should be %d after 3 minutes", replicas))
			wg.Done()
		}(i)
	}
	wg.Wait()

	// scale monitored deployment to maxReplicaCount - 1 replicas
	replicas = maxReplicaCount - 1
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	saveLogs(t, kc, operatorLogName, operatorLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	wg.Add(scaledObjectCount)
	for i := 0; i < scaledObjectCount; i++ {
		go func(index int) {
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, fmt.Sprintf(sutDeploymentName, index), testNamespace, replicas, 60, 3),
				fmt.Sprintf("replica count should be %d after 3 minutes", replicas))
			wg.Done()
		}(i)
	}
	wg.Wait()

	// scale monitored deployment to maxReplicaCount replicas
	replicas = maxReplicaCount
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	saveLogs(t, kc, msLogName, msLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, msLabelSelector, kedaNamespace)
	wg.Add(scaledObjectCount)
	for i := 0; i < scaledObjectCount; i++ {
		go func(index int) {
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, fmt.Sprintf(sutDeploymentName, index), testNamespace, replicas, 60, 3),
				fmt.Sprintf("replica count should be %d after 3 minutes", replicas))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to minReplicaCount + 1 replicas
	replicas := minReplicaCount + 1
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	saveLogs(t, kc, operatorLogName, operatorLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	saveLogs(t, kc, msLogName, msLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, msLabelSelector, kedaNamespace)
	var wg sync.WaitGroup
	wg.Add(scaledObjectCount)
	for i := 0; i < scaledObjectCount; i++ {
		go func(index int) {
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, fmt.Sprintf(sutDeploymentName, index), testNamespace, replicas, 60, 3),
				fmt.Sprintf("replica count should be %d after 3 minutes", replicas))
			wg.Done()
		}(i)
	}
	wg.Wait()

	// scale monitored deployment to minReplicaCount replicas
	replicas = minReplicaCount
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	saveLogs(t, kc, operatorLogName, operatorLabelSelector, kedaNamespace)
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	wg.Add(scaledObjectCount)
	for i := 0; i < scaledObjectCount; i++ {
		go func(index int) {
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, fmt.Sprintf(sutDeploymentName, index), testNamespace, replicas, 60, 3),
				fmt.Sprintf("replica count should be %d after 3 minutes", replicas))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func saveLogs(t *testing.T, kc *kubernetes.Clientset, logName, selector, namespace string) {
	logs, err := FindPodLogs(kc, namespace, selector, false)
	assert.NoErrorf(t, err, "cannotget logs - %s", err)
	f, err := os.Create(fmt.Sprintf("%s-%s.log", logName, time.Now().Format("20060102150405")))
	assert.NoErrorf(t, err, "cannot create log file - %s", err)
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range logs {
		fmt.Fprintln(w, line)
	}
	err = w.Flush()
	assert.NoErrorf(t, err, "cannot save log file - %s", err)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			MonitoredDeploymentName: monitoredDeploymentName,
			SutDeploymentName:       sutDeploymentName,
			ScaledObjectName:        scaledObjectName,
			MinReplicaCount:         minReplicaCount,
			MaxReplicaCount:         maxReplicaCount,
		}, []Template{
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

//go:build e2e
// +build e2e

package events_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "events-test"
)

var (
	testNamespace                       = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName             = fmt.Sprintf("%s-monitor-deployment", testName)
	deploymentName                      = fmt.Sprintf("%s-deployment", testName)
	daemonsetName                       = fmt.Sprintf("%s-daemonset", testName)
	scaledObjectName                    = fmt.Sprintf("%s-so", testName)
	scaledObjectTargetNotFoundName      = fmt.Sprintf("%s-so-target-error", testName)
	scaledObjectTargetNoSubresourceName = fmt.Sprintf("%s-so-target-no-subresource", testName)
)

type templateData struct {
	TestNamespace                       string
	ScaledObjectName                    string
	ScaledObjectTargetNotFoundName      string
	ScaledObjectTargetNoSubresourceName string
	DeploymentName                      string
	MonitoredDeploymentName             string
	DaemonsetName                       string
}

const (
	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'nginxinc/nginx-unprivileged'
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
        - name: {{.DeploymentName}}
          image: nginxinc/nginx-unprivileged:alpine-slim
`

	scaledObjectTargetErrTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectTargetNotFoundName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: no-exist
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.DeploymentName}}'
        value: '1'
`

	daemonSetTemplate = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: {{.DaemonsetName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DaemonsetName}}
spec:
  selector:
    matchLabels:
      app: {{.DaemonsetName}}
  template:
    metadata:
      labels:
        app: {{.DaemonsetName}}
    spec:
      containers:
        - name: {{.DaemonsetName}}
          image: nginxinc/nginx-unprivileged:alpine-slim
`

	scaledObjectTargetNotSupportTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectTargetNoSubresourceName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DaemonsetName}}
    kind: DaemonSet
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.DeploymentName}}'
        value: '1'
`
)

func TestEvents(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test scaling
	testNormalEvent(t, kc, data)
	testTargetNotFoundErr(t, kc, data)
	testTargetNotSupportEventErr(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:                       testNamespace,
		DeploymentName:                      deploymentName,
		MonitoredDeploymentName:             monitoredDeploymentName,
		DaemonsetName:                       daemonsetName,
		ScaledObjectName:                    scaledObjectName,
		ScaledObjectTargetNotFoundName:      scaledObjectTargetNotFoundName,
		ScaledObjectTargetNoSubresourceName: scaledObjectTargetNoSubresourceName,
	}, []Template{}
}

func checkingEvent(t *testing.T, scaledObject string, index int, eventreason string, message string) {
	result, err := ExecuteCommand(fmt.Sprintf("kubectl get events -n %s --field-selector involvedObject.name=%s --sort-by=.metadata.creationTimestamp -o jsonpath=\"{.items[%d].reason}:{.items[%d].message}\"", testNamespace, scaledObject, index, index))

	assert.NoError(t, err)
	lastEventMessage := strings.Trim(string(result), "\"")
	assert.Equal(t, lastEventMessage, eventreason+":"+message)
}

func testNormalEvent(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing normal event ---")

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentName", monitoredDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// time.Sleep(2 * time.Second)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 2, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
	checkingEvent(t, scaledObjectName, 0, eventreason.KEDAScalersStarted, fmt.Sprintf(message.ScalerIsBuiltMsg, "kubernetes-workload"))
	checkingEvent(t, scaledObjectName, 1, eventreason.KEDAScalersStarted, message.ScalerStartMsg)
	checkingEvent(t, scaledObjectName, 2, eventreason.ScaledObjectReady, message.ScalerReadyMsg)
}

func testTargetNotFoundErr(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing target not found error event ---")

	KubectlApplyWithTemplate(t, data, "scaledObjectTargetErrTemplate", scaledObjectTargetErrTemplate)
	checkingEvent(t, scaledObjectTargetNotFoundName, -2, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNotFoundMsg)
	checkingEvent(t, scaledObjectTargetNotFoundName, -1, eventreason.ScaledObjectCheckFailed, message.ScaleTargetErrMsg)
}

func testTargetNotSupportEventErr(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing target not support error event ---")

	KubectlApplyWithTemplate(t, data, "daemonSetTemplate", daemonSetTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTargetNotSupportTemplate", scaledObjectTargetNotSupportTemplate)
	checkingEvent(t, scaledObjectTargetNoSubresourceName, -2, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNoSubresourceMsg)
	checkingEvent(t, scaledObjectTargetNoSubresourceName, -1, eventreason.ScaledObjectCheckFailed, message.ScaleTargetErrMsg)
}

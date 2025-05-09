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
	secretName                          = fmt.Sprintf("%s-secret", testName)
	secretName2                         = fmt.Sprintf("%s-secret-2", testName)
	triggerAuthName                     = fmt.Sprintf("%s-ta", testName)
	clusterTriggerAuthName              = fmt.Sprintf("%s-cta", testName)

	scaledJobName    = fmt.Sprintf("%s-sj", testName)
	scaledJobErrName = fmt.Sprintf("%s-sj-target-error", testName)
)

type templateData struct {
	TestNamespace                       string
	ScaledObjectName                    string
	ScaledObjectTargetNotFoundName      string
	ScaledObjectTargetNoSubresourceName string
	DeploymentName                      string
	MonitoredDeploymentName             string
	DaemonsetName                       string
	ScaledJobName                       string
	ScaledJobErrName                    string
	SecretName                          string
	SecretName2                         string
	SecretTargetName                    string
	TriggerAuthName                     string
	ClusterTriggerAuthName              string
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
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'
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
          image: ghcr.io/nginx/nginx-unprivileged:1.26
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
          image: ghcr.io/nginx/nginx-unprivileged:1.26
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

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 8
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`

	scaledJobErrTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobErrName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 8
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
    - type: cpu
      name: x
      metadata:
        typex: Utilization
        value: "50"
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`
	secret2Template = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName2}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretTargetName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretTargetName}}
      key: AUTH_PASSWORD
`

	clusterTriggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthName}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretTargetName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretTargetName}}
      key: AUTH_PASSWORD
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

	testScaledJobNormalEvent(t, kc, data)
	testScaledJobTargetNotSupportEventErr(t, kc, data)

	testTriggerAuthenticationEvent(t, kc, data)

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
		ScaledJobName:                       scaledJobName,
		ScaledJobErrName:                    scaledJobErrName,
		SecretName:                          secretName,
		SecretName2:                         secretName2,
		TriggerAuthName:                     triggerAuthName,
		ClusterTriggerAuthName:              clusterTriggerAuthName,
	}, []Template{}
}

func checkingEvent(t *testing.T, namespace string, scaledObject string, index int, eventReason string, message string) {
	result, err := ExecuteCommand(fmt.Sprintf("kubectl get events -n %s --field-selector involvedObject.name=%s --sort-by=.metadata.creationTimestamp -o jsonpath=\"{.items[%d].reason}:{.items[%d].message}\"", namespace, scaledObject, index, index))

	assert.NoError(t, err)
	lastEventMessage := strings.Trim(string(result), "\"")
	assert.Equal(t, eventReason+":"+message, lastEventMessage)
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
	checkingEvent(t, testNamespace, scaledObjectName, 0, eventreason.KEDAScalersStarted, fmt.Sprintf(message.ScalerIsBuiltMsg, "kubernetes-workload"))
	checkingEvent(t, testNamespace, scaledObjectName, 1, eventreason.KEDAScalersStarted, message.ScalerStartMsg)
	checkingEvent(t, testNamespace, scaledObjectName, 2, eventreason.ScaledObjectReady, message.ScalerReadyMsg)

	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "monitoredDeploymentName", monitoredDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testTargetNotFoundErr(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing target not found error event ---")

	KubectlApplyWithTemplate(t, data, "scaledObjectTargetErrTemplate", scaledObjectTargetErrTemplate)
	checkingEvent(t, testNamespace, scaledObjectTargetNotFoundName, -2, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNotFoundMsg)
	checkingEvent(t, testNamespace, scaledObjectTargetNotFoundName, -1, eventreason.ScaledObjectCheckFailed, message.ScaleTargetErrMsg)
}

func testTargetNotSupportEventErr(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing target not support error event ---")

	KubectlApplyWithTemplate(t, data, "daemonSetTemplate", daemonSetTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTargetNotSupportTemplate", scaledObjectTargetNotSupportTemplate)
	checkingEvent(t, testNamespace, scaledObjectTargetNoSubresourceName, -2, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNoSubresourceMsg)
	checkingEvent(t, testNamespace, scaledObjectTargetNoSubresourceName, -1, eventreason.ScaledObjectCheckFailed, message.ScaleTargetErrMsg)
}

func testScaledJobNormalEvent(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing ScaledJob normal event ---")

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentName", monitoredDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 2, testNamespace)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
	checkingEvent(t, testNamespace, scaledJobName, 0, eventreason.KEDAScalersStarted, fmt.Sprintf(message.ScalerIsBuiltMsg, "kubernetes-workload"))
	checkingEvent(t, testNamespace, scaledJobName, 1, eventreason.KEDAScalersStarted, message.ScalerStartMsg)
	checkingEvent(t, testNamespace, scaledJobName, 2, eventreason.ScaledJobReady, message.ScaledJobReadyMsg)

	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "monitoredDeploymentName", monitoredDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
}

func testScaledJobTargetNotSupportEventErr(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing target not support error event ---")

	KubectlApplyWithTemplate(t, data, "scaledJobErrTemplate", scaledJobErrTemplate)
	checkingEvent(t, testNamespace, scaledJobErrName, -1, eventreason.ScaledJobCheckFailed, "Failed to ensure ScaledJob is correctly created")
}

func testTriggerAuthenticationEvent(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- testing ScaledJob normal event ---")

	KubectlApplyWithTemplate(t, data, "secretTemplate", secretTemplate)
	KubectlApplyWithTemplate(t, data, "secret2Template", secret2Template)

	data.SecretTargetName = secretName
	KubectlApplyWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)

	checkingEvent(t, testNamespace, triggerAuthName, 0, eventreason.TriggerAuthenticationAdded, message.TriggerAuthenticationCreatedMsg)

	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthenticationTemplate", clusterTriggerAuthenticationTemplate)

	checkingEvent(t, "default", clusterTriggerAuthName, 0, eventreason.ClusterTriggerAuthenticationAdded, message.ClusterTriggerAuthenticationCreatedMsg)

	data.SecretTargetName = secretName2
	KubectlApplyWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)

	checkingEvent(t, testNamespace, triggerAuthName, -1, eventreason.TriggerAuthenticationUpdated, fmt.Sprintf(message.TriggerAuthenticationUpdatedMsg, triggerAuthName))
	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthenticationTemplate", clusterTriggerAuthenticationTemplate)

	checkingEvent(t, "default", clusterTriggerAuthName, -1, eventreason.ClusterTriggerAuthenticationUpdated, fmt.Sprintf(message.ClusterTriggerAuthenticationUpdatedMsg, clusterTriggerAuthName))
	KubectlDeleteWithTemplate(t, data, "secretTemplate", secretTemplate)
	KubectlDeleteWithTemplate(t, data, "secret2Template", secret2Template)
	KubectlDeleteWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)
	KubectlDeleteWithTemplate(t, data, "clusterTriggerAuthenticationTemplate", clusterTriggerAuthenticationTemplate)
}

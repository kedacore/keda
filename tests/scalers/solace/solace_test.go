//go:build e2e
// +build e2e

package solace_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "solace-test"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	helperName                = fmt.Sprintf("%s-helper", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	minReplicaCount           = 0
	maxReplicaCount           = 2
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	HelperName                string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	MinReplicaCount           int
	MaxReplicaCount           int
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
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: solace-jms-consumer
        image: ghcr.io/solacelabs/kedalab-consumer
        env:
        - name:  SOLACE_CLIENT_HOST
          value: tcp://kedalab-pubsubplus-dev:55555
        - name:  SOLACE_CLIENT_MSGVPN
          value: keda_vpn
        - name:  SOLACE_CLIENT_USERNAME
          value: consumer_user
        - name:  SOLACE_CLIENT_PASSWORD
          value: consumer_pwd
        - name:  SOLACE_CLIENT_QUEUENAME
          value: SCALED_CONSUMER_QUEUE1
        - name:  SOLACE_CLIENT_CONSUMER_DELAY
          value: '1000'
        imagePullPolicy: Always
      restartPolicy: Always`

	helperTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: {{.HelperName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: sdk-perf
    image: ghcr.io/solacelabs/kedalab-helper:latest
    # Just spin & wait forever
    command: [ "/bin/bash", "-c", "--" ]
    args: [ "while true; do sleep 10; done;" ]
`
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  SEMP_USER:         YWRtaW4=
  SEMP_PASSWORD:     S2VkYUxhYkFkbWluUHdkMQ==
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter:   username
    name:        {{.SecretName}}
    key:         SEMP_USER
  - parameter:   password
    name:        {{.SecretName}}
    key:         SEMP_PASSWORD
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Percent
            value:         100
            periodSeconds: 10
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Pods
            value:         10
            periodSeconds: 10
          selectPolicy:    Max
  triggers:
  - type: solace-event-queue
    metadata:
      solaceSempBaseURL: http://kedalab-pubsubplus-dev.{{.TestNamespace}}.svc.cluster.local:8080
      messageVpn: keda_vpn
      queueName: SCALED_CONSUMER_QUEUE1
      messageCountTarget: '20'
      messageSpoolUsageTarget: '1'
      activationMessageCountTarget: '20'
      activationMessageSpoolUsageTarget: '20'
      activationMessageReceiveRateTarget: '100'
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	scaledObjectTemplateRate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  5
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Percent
            value:         100
            periodSeconds: 10
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type:          Pods
            value:         10
            periodSeconds: 10
          selectPolicy:    Max
  triggers:
  - type: solace-event-queue
    metadata:
      solaceSempBaseURL: http://kedalab-pubsubplus-dev.{{.TestNamespace}}.svc.cluster.local:8080
      messageVpn: keda_vpn
      queueName: SCALED_CONSUMER_QUEUE1
      messageReceiveRateTarget: '5'
      # Will not activate on count or spool
      activationMessageCountTarget: '1000'
      activationMessageSpoolUsageTarget: '1000'
      activationMessageReceiveRateTarget: '3'
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
)

func TestStanScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	installSolace(t)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateRate", scaledObjectTemplateRate)

	testActivationRate(t, kc)
	testScaleOutRate(t, kc)
	testScaleInRate(t, kc)

	// cleanup
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplateRate", scaledObjectTemplateRate)
	uninstallSolace(t)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func installSolace(t *testing.T) {
	_, err := ExecuteCommand("helm repo add solacecharts https://solaceproducts.github.io/pubsubplus-kubernetes-helm-quickstart/helm-charts")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --set solace.usernameAdminPassword=KedaLabAdminPwd1 --set storage.persistent=false,solace.size=dev,nameOverride=pubsubplus-dev --namespace %s kedalab solacecharts/pubsubplus`,
		testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("sleep 60") // there is a bug in the solace helm chart where it is looking for the wrong number of replicas on --wait
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	// Create the pubsub broker
	_, _, err = ExecCommandOnSpecificPod(t, helperName, testNamespace, "./config/config_solace.sh")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func uninstallSolace(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --namespace %s --wait kedalab`,
		testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func publishMessages(t *testing.T, messageRate, messageNumber, messageSize int) {
	_, _, err := ExecCommandOnSpecificPod(t, helperName, testNamespace, fmt.Sprintf("./sdkperf/sdkperf_java.sh -cip=kedalab-pubsubplus-dev:55555 -cu consumer_user@keda_vpn -cp=consumer_pwd -mr %d -mn %d -msx %d -mt=persistent -pql=SCALED_CONSUMER_QUEUE1", messageRate, messageNumber, messageSize))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	publishMessages(t, 50, 10, 1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	publishMessages(t, 50, 40, 256)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func testActivationRate(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation activationMsgRxRateTarget ---")
	// Next line is a delay -- Wait to smooth out msg receive rate to avoid false+
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)
	publishMessages(t, 1, 30, 256)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)
}

func testScaleOutRate(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out msgRxRateTarget---")
	publishMessages(t, 30, 300, 256)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleInRate(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in msgRxRateTarget ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:             testNamespace,
			DeploymentName:            deploymentName,
			HelperName:                helperName,
			ScaledObjectName:          scaledObjectName,
			TriggerAuthenticationName: triggerAuthenticationName,
			SecretName:                secretName,
			MinReplicaCount:           minReplicaCount,
			MaxReplicaCount:           maxReplicaCount,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "helperTemplate", Config: helperTemplate},
		}
}

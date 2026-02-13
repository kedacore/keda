//go:build e2e
// +build e2e

// ^ This is necessary to ensure the tests don't get run in the GitHub workflow.
package solace_dm_test

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
	solaceDMTestName = "solace-dm"
)

var (
	solaceDMTestNamespace         = fmt.Sprintf("%s-ns", solaceDMTestName)
	consumerDeploymentName        = "direct-messaging-simple-consumer"
	consumerDeploymentSecretsName = fmt.Sprintf("%s-secrets", solaceDMTestName)
	producerPodName               = fmt.Sprintf("%s-producer", solaceDMTestName)
	// scaler
	scalerSecretsName               = fmt.Sprintf("%s-scaler-secrets", solaceDMTestName)
	scalerTriggerAuthenticationName = fmt.Sprintf("%s-scaler-trigger-auth", solaceDMTestName)
	scalerScaledObjectName          = fmt.Sprintf("%s-scaler-so", solaceDMTestName)
	// scaled object parameters
	minReplicaCount                         = 1
	maxReplicaCount                         = 5
	aggregatedClientTxMsgRateTarget         = 600
	aggregatedClientTxByteRateTarget        = 0
	aggregatedClientAverageTxByteRateTarget = 0
	aggregatedClientAverageTxMsgRateTarget  = 0
)

type solaceDMTemplateData struct {
	TestNamespace                   string
	ConsumerDeploymentName          string
	ConsumerDeploymentSecretsName   string
	ProducerPodName                 string
	ScalerSecretsName               string
	ScalerTriggerAuthenticationName string
	ScalerScaledObjectName          string

	MinReplicaCount                         int
	MaxReplicaCount                         int
	AggregatedClientTxMsgRateTarget         int
	AggregatedClientTxByteRateTarget        int
	AggregatedClientAverageTxByteRateTarget int
	AggregatedClientAverageTxMsgRateTarget  int
}

const consumerDeploymentSecretsTemplate = `
kind: Secret
apiVersion: v1
metadata:
  name: {{.ConsumerDeploymentSecretsName}}
  namespace: {{.TestNamespace}}
stringData:
    SOL_HOST: kedalab-pubsubplus-dev.{{.TestNamespace}}.svc.cluster.local:55555
    SOL_USER: default@default
    SOL_PWD:  ""
    SOL_TOPIC: "#share/consumer_group/topic/message/A"
    SOL_EXTRA_ARGS: -d=5
`
const consumerDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ConsumerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ConsumerDeploymentName}}
spec:
  selector:
    matchLabels:
      app: {{.ConsumerDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.ConsumerDeploymentName}}
    spec:
      containers:
      - name: {{.ConsumerDeploymentName}}
        image: "ghcr.io/solacelabs/direct-messaging-simple-consumer:latest"
        envFrom:
        - secretRef:
            name: {{.ConsumerDeploymentSecretsName}}
`

const producerPodTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ProducerPodName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
    - name: sdk-perf
      image: ghcr.io/solacelabs/kedalab-helper:latest
      # Just spin & wait forever
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "while true; do sleep 10; done;" ]
`
const scalerSecretsTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name:      {{.ScalerSecretsName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  SEMP_USER:         admin
  SEMP_PASSWORD:     admin
`
const scalerTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.ScalerTriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter:   username
      name:        {{.ScalerSecretsName}}
      key:         SEMP_USER
    - parameter:   password
      name:        {{.ScalerSecretsName}}
      key:         SEMP_PASSWORD
`

const scalerObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name:      {{.ScalerScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    apiVersion:    apps/v1
    kind:          Deployment
    name:          {{.ConsumerDeploymentName}}
  pollingInterval:  3
  cooldownPeriod:  60
  #Always > 0
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    restoreToOriginalReplicaCount: true
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
          #Policy (Pods) allows at most 5 replicas to be scaled down in 10 seconds.
          policies:
          - type:          Pods
            value:         5
            #Indicates the length of time in the past for which the policy must hold true
            periodSeconds: 5
        scaleUp:
          stabilizationWindowSeconds: 0
          #Policy (Pods) allows at most 3 replicas to be scaled up in 3 seconds.
          policies:
          - type:          Pods
            value:         5
            periodSeconds: 3
          selectPolicy:    Max
  triggers:
  - type: solace-direct-messaging
    metricType: Value
    metadata:
      solaceSempBaseURL:  "http://kedalab-pubsubplus-dev.{{.TestNamespace}}.svc.cluster.local:8080"
      messageVpn: "default"
      clientNamePattern: "direct-messaging-simple"
      #to be able to use self signed certs
      unsafeSSL: "true"
      #to increase weight on queued messages and scale faster
      #if there are messages queued means we are behind
      queuedMessagesFactor: '3'
      #Metrics
      aggregatedClientTxMsgRateTarget: '{{.AggregatedClientTxMsgRateTarget}}'
      aggregatedClientTxByteRateTarget: '{{.AggregatedClientTxByteRateTarget}}'
      aggregatedClientAverageTxByteRateTarget: '{{.AggregatedClientAverageTxByteRateTarget}}'
      aggregatedClientAverageTxMsgRateTarget: '{{.AggregatedClientAverageTxMsgRateTarget}}'
    authenticationRef:
      name: {{.ScalerTriggerAuthenticationName}}
`

func TestSolaceDMScalerRatePerSecond(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getSolaceDMTemplateData()

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, solaceDMTestNamespace, data, templates)
	installSolace(t)

	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "scaledObjectTemplateRate", scalerObjectTemplate)
		uninstallSolace(t)
		DeleteKubernetesResources(t, solaceDMTestNamespace, data, templates)
	})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, solaceDMTestNamespace, minReplicaCount, 60, 1),
		"replica count should be 1 after 1 minute - before start testing")

	testMsgRatePerSecond(t, kc, &data)
}

func TestSolaceDMScalerBytePerSecond(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getSolaceDMTemplateData()

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, solaceDMTestNamespace, data, templates)
	installSolace(t)

	t.Cleanup(func() {
		KubectlDeleteWithTemplate(t, data, "scaledObjectTemplateRate", scalerObjectTemplate)
		uninstallSolace(t)
		DeleteKubernetesResources(t, solaceDMTestNamespace, data, templates)
	})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, solaceDMTestNamespace, minReplicaCount, 60, 1),
		"replica count should be 1 after 1 minute - before start testing")

	testByteRatePerSecond(t, kc, &data)
}

/*************************************************************************/
/*** Solace Broker Install/Uninstall                                     */
/*************************************************************************/
func installSolace(t *testing.T) {
	_, err := ExecuteCommand("helm repo add solacecharts https://solaceproducts.github.io/pubsubplus-kubernetes-helm-quickstart/helm-charts")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --set solace.usernameAdminPassword=admin --set storage.persistent=false,solace.size=dev,nameOverride=pubsubplus-dev,service.type=ClusterIP --wait --namespace %s kedalab solacecharts/pubsubplus`,
		solaceDMTestNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func uninstallSolace(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --namespace %s --wait kedalab`, solaceDMTestNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func publishMessages(t *testing.T, messageRate, messageNumber, messageSize int) {
	_, _, err := ExecCommandOnSpecificPod(t, producerPodName, solaceDMTestNamespace, fmt.Sprintf("./sdkperf/sdkperf_java.sh -cip=kedalab-pubsubplus-dev:55555 -cu default@default -cp= -mr %d -mn %d -msx %d -mt=direct -ptl=topic/message/A", messageRate, messageNumber, messageSize))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, mode string, messageRate int, messageNumber int, iterations int, interval int) {
	t.Logf("--- testing scale out: '%s' ---", mode)
	publishMessages(t, messageRate, messageNumber, 1024)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, solaceDMTestNamespace, maxReplicaCount, iterations, interval),
		"replica count should be '%d' after '%d' seconds", maxReplicaCount, iterations*interval)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, mode string, iterations int, interval int) {
	t.Logf("--- testing scale in: '%s' ---", mode)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, solaceDMTestNamespace, minReplicaCount, iterations, interval),
		"replica count should be '%d' after '%d' seconds", minReplicaCount, iterations*interval)
}

func cleanParams(data *solaceDMTemplateData) {
	// clean
	data.AggregatedClientTxMsgRateTarget = 0
	data.AggregatedClientTxByteRateTarget = 0
	data.AggregatedClientAverageTxMsgRateTarget = 0
	data.AggregatedClientAverageTxByteRateTarget = 0
}

func testMsgRatePerSecond(t *testing.T, kc *kubernetes.Clientset, data *solaceDMTemplateData) {
	cleanParams(data)

	data.AggregatedClientTxMsgRateTarget = 600
	// Install ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scalerObjectTemplate)
	// will send 700 msgs per second during 30 secs
	// wait 30 secs
	// the number of instances should be the maximum configured
	testScaleOut(t, kc, "TxMsgRate", 700, 700*30, 30, 1)

	// wait 125 seconds to scaler to reduce the replica number to the minimumn
	testScaleIn(t, kc, "TxMsgRate", 180, 1)
}
func testByteRatePerSecond(t *testing.T, kc *kubernetes.Clientset, data *solaceDMTemplateData) {
	cleanParams(data)

	data.AggregatedClientTxByteRateTarget = 600 * 1024
	// Install ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scalerObjectTemplate)

	// will send 900 msgs per second during 30 secs
	// wait 30 secs
	// the number of instances should be the maximum configured
	testScaleOut(t, kc, "TxByteRate", 1000, 1000*30, 30, 1)

	// wait 130 seconds to scaler to reduce the replica number to the minimumn
	testScaleIn(t, kc, "TxByteRate", 180, 1)
}

func getSolaceDMTemplateData() (solaceDMTemplateData, []Template) {
	return solaceDMTemplateData{
			TestNamespace:                           solaceDMTestNamespace,
			ConsumerDeploymentName:                  consumerDeploymentName,
			ConsumerDeploymentSecretsName:           consumerDeploymentSecretsName,
			ProducerPodName:                         producerPodName,
			ScalerSecretsName:                       scalerSecretsName,
			ScalerTriggerAuthenticationName:         scalerTriggerAuthenticationName,
			ScalerScaledObjectName:                  scalerScaledObjectName,
			MinReplicaCount:                         minReplicaCount,
			MaxReplicaCount:                         maxReplicaCount,
			AggregatedClientTxMsgRateTarget:         aggregatedClientTxMsgRateTarget,
			AggregatedClientTxByteRateTarget:        aggregatedClientTxByteRateTarget,
			AggregatedClientAverageTxByteRateTarget: aggregatedClientAverageTxByteRateTarget,
			AggregatedClientAverageTxMsgRateTarget:  aggregatedClientAverageTxMsgRateTarget,
		}, []Template{
			{Name: "consumerDeploymentSecrets", Config: consumerDeploymentSecretsTemplate},
			{Name: "consumerDeployment", Config: consumerDeploymentTemplate},
			{Name: "producerPodTemplate", Config: producerPodTemplate},
			{Name: "scalerSecretsTemplate", Config: scalerSecretsTemplate},
			{Name: "scalerTriggerAuthenticationTemplate", Config: scalerTriggerAuthenticationTemplate},
		}
}

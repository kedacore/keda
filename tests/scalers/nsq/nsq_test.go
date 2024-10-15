//go:build e2e
// +build e2e

package nsq_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

var _ = godotenv.Load("../../.env")

const (
	testName = "nsq-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-consumer-deployment", testName)
	jobName          = fmt.Sprintf("%s-producer-job", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	nsqNamespace     = "nsq"
	nsqHelmRepoURL   = "https://nsqio.github.io/helm-chart"
	minReplicas      = 1
	maxReplicas      = 10
	topicName        = "test_topic"
	channelName      = "test_channel"
)

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
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
      - image: ghcr.io/kedacore/tests-nsq:latest
        name: {{.DeploymentName}}
        args:
        - "--mode=consumer"
        - "--topic={{.TopicName}}"
        - "--channel={{.ChannelName}}"
        - "--nsqlookupd-http-address=nsq-nsqlookupd.{{.NSQNamespace}}.svc.cluster.local:4161"
        imagePullPolicy: Always
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 10
  idleReplicaCount: 0
  maxReplicaCount: {{.MaxReplicas}}
  minReplicaCount: {{.MinReplicas}}
  scaleTargetRef:
    apiVersion: "apps/v1"
    kind: "Deployment"
    name: {{.DeploymentName}}
  triggers:
  - type: nsq
    metricType: "AverageValue"
    metadata:
      nsqLookupdHTTPAddresses: "nsq-nsqlookupd.{{.NSQNamespace}}.svc.cluster.local:4161"
      topic: "{{.TopicName}}"
      channel: "{{.ChannelName}}"
      depthThreshold: "10"
      activationDepthThreshold: "5"
`

	jobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-nsq:latest
        name: {{.JobName}}
        args:
        - "--mode=producer"
        - "--topic={{.TopicName}}"
        - "--nsqd-tcp-address=nsq-nsqd.{{.NSQNamespace}}.svc.cluster.local:4150"
        - "--message-count={{.MessageCount}}"
        imagePullPolicy: Always
      restartPolicy: Never
`
)

type templateData struct {
	TestNamespace    string
	NSQNamespace     string
	DeploymentName   string
	ScaledObjectName string
	JobName          string
	MinReplicas      int
	MaxReplicas      int
	TopicName        string
	ChannelName      string
	MessageCount     int
}

func TestNSQScaler(t *testing.T) {
	kc := GetKubernetesClient(t)

	t.Cleanup(func() {
		data, templates := getTemplateData()
		uninstallNSQ(t)
		KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	installNSQ(t, kc)

	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"Replica count should start out as 0")

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func installNSQ(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- installing NSQ ---")
	CreateNamespace(t, kc, nsqNamespace)

	_, err := ExecuteCommand("which helm")
	require.NoErrorf(t, err, "nsq test requires helm - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("helm repo add nsqio %s", nsqHelmRepoURL))
	require.NoErrorf(t, err, "error while adding nsqio helm repo - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("helm install nsq nsqio/nsq --namespace %s --set nsqadmin.enabled=false --wait", nsqNamespace))
	require.NoErrorf(t, err, "error while installing nsq - %s", err)
}

func uninstallNSQ(t *testing.T) {
	t.Log("--- uninstalling NSQ ---")
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall nsq --namespace %s", nsqNamespace))
	require.NoErrorf(t, err, "error while uninstalling nsq - %s", err)
	DeleteNamespace(t, nsqNamespace)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			NSQNamespace:     nsqNamespace,
			DeploymentName:   deploymentName,
			JobName:          jobName,
			ScaledObjectName: scaledObjectName,
			MinReplicas:      minReplicas,
			MaxReplicas:      maxReplicas,
			TopicName:        topicName,
			ChannelName:      channelName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")

	data.MessageCount = 5
	KubectlReplaceWithTemplate(t, data, "jobTemplate", jobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	data.MessageCount = 1 // 5 already published + 1 > activationDepthThreshold
	KubectlReplaceWithTemplate(t, data, "jobTemplate", jobTemplate)

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

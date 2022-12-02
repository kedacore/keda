//go:build e2e
// +build e2e

package natsjetstream_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	k8s "k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	nats "github.com/kedacore/keda/v2/tests/scalers/nats_jetstream"
)

// Load env variables from .env files
var _ = godotenv.Load("../../.env")

const (
	testName = "nats-jetstream-test"
)

var (
	testNamespace                = fmt.Sprintf("%s-ns", testName)
	natsNamespace                = fmt.Sprintf("%s-nats-ns", testName)
	natsAddress                  = fmt.Sprintf("nats://nats.%s.svc.cluster.local:4222", natsNamespace)
	natsServerMonitoringEndpoint = fmt.Sprintf("nats.%s.svc.cluster.local:8222", natsNamespace)
	messagePublishCount          = 1000
	deploymentName               = "sub"
	minReplicaCount              = 0
	maxReplicaCount              = 2
)

type templateData struct {
	TestNamespace                string
	NatsAddress                  string
	NatsServerMonitoringEndpoint string
	NumberOfMessages             int
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sub
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: sub
  template:
    metadata:
      labels:
        app: sub
    spec:
      containers:
      - name: sub
        image: "goku321/nats-consumer:v0.9"
        imagePullPolicy: Always
        command: ["./main"]
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
`

	publishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: pub
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: pub
        image: "goku321/nats-publisher:v0.2"
        imagePullPolicy: Always
        command: ["./main"]
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
        - name: NUM_MESSAGES
          value: "{{.NumberOfMessages}}"
      restartPolicy: Never
  backoffLimit: 4
`

	activationPublishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: pub0
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: pub
        image: "goku321/nats-publisher:v0.2"
        imagePullPolicy: Always
        command: ["./main"]
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
        - name: NUM_MESSAGES
          value: "{{.NumberOfMessages}}"
      restartPolicy: Never
  backoffLimit: 4
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: nats-jetstream-scaledobject
  namespace: {{.TestNamespace}}
spec:
  pollingInterval: 3
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 2
  scaleTargetRef:
    name: sub
  triggers:
  - type: nats-jetstream
    metadata:
      natsServerMonitoringEndpoint: "{{.NatsServerMonitoringEndpoint}}"
      account: "$G"
      stream: "mystream"
      consumer: "PULL_CONSUMER"
      lagThreshold: "10"
      activationLagThreshold: "15"
  `
)

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:                testNamespace,
			NatsAddress:                  natsAddress,
			NatsServerMonitoringEndpoint: natsServerMonitoringEndpoint,
			NumberOfMessages:             messagePublishCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func TestNATSJetStreamScaler(t *testing.T) {
	// Create k8s resources.
	kc := GetKubernetesClient(t)

	// Deploy NATS server.
	nats.InstallServerWithJetStream(t, kc, natsNamespace)
	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "nats", natsNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create k8s resources for testing.
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Create stream and consumer.
	nats.InstallStreamAndConsumer(t, kc, testNamespace, natsAddress)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job should be success")

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// Cleanup.
	nats.RemoveServer(t, kc, natsNamespace)
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *k8s.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.NumberOfMessages = 10
	KubectlApplyWithTemplate(t, data, "activationPublishJobTemplate", activationPublishJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *k8s.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "publishJobTemplate", publishJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *k8s.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

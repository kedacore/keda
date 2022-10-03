//go:build e2e
// +build e2e

package pulsar_test

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
	testName = "pulsar-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	statefulSetName        = fmt.Sprintf("%s-sts", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	consumerDeploymentName = fmt.Sprintf("%s-consumer-deploy", testName)
	producerJobName        = fmt.Sprintf("%s-producer-job", testName)
	messageCount           = 3
	minReplicaCount        = 0
	maxReplicaCount        = 5
	msgBacklog             = 10
)

type templateData struct {
	TestNamespace          string
	StatefulSetName        string
	MessageCount           int
	ScaledObjectName       string
	ConsumerDeploymentName string
	ProducerJobName        string
	MinReplicaCount        int
	MaxReplicaCount        int
	MsgBacklog             int
}

const statefulsetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
 name: {{.StatefulSetName}}
 namespace: {{.TestNamespace}}
 labels:
  app: pulsar
spec:
  selector:
    matchLabels:
      app: pulsar
  replicas: 1
  serviceName: {{.StatefulSetName}}
  template:
    metadata:
      labels:
        app: pulsar
    spec:
      containers:
      - name: pulsar
        image: apachepulsar/pulsar:2.10.0
        imagePullPolicy: IfNotPresent
        readinessProbe:
          tcpSocket:
            port: 8080
        ports:
        - name: pulsar
          containerPort: 6650
          protocol: TCP
        - name: admin
          containerPort: 8080
          protocol: TCP
        env:
        - name: PULSAR_PREFIX_tlsRequireTrustedClientCertOnConnect
          value: "true"
        command:
        - sh
        - -c
        args: ["bin/apply-config-from-env.py conf/client.conf && bin/apply-config-from-env.py conf/standalone.conf && bin/pulsar standalone -nfw -nss"]
`

const consumerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ConsumerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: pulsar-consumer
spec:
  selector:
    matchLabels:
      app: pulsar-consumer
  template:
    metadata:
      labels:
        app: pulsar-consumer
    spec:
      containers:
        - name: pulsar-consumer
          image: ghcr.io/pulsar-sigs/pulsar-client:v0.3.1
          imagePullPolicy: IfNotPresent
          readinessProbe:
            tcpSocket:
              port: 9494
          args: ["consumer","--broker","pulsar://{{.StatefulSetName}}.{{.TestNamespace}}:6650","--topic","persistent://public/default/keda","--subscription-name","keda","--consume-time","200"]

`

const scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.ConsumerDeploymentName}}
  pollingInterval: 5 # Optional. Default: 30 seconds
  cooldownPeriod: 30 # Optional. Default: 300 seconds
  maxReplicaCount: {{.MaxReplicaCount}}
  minReplicaCount: {{.MinReplicaCount}}
  triggers:
    - type: pulsar
      metadata:
        msgBacklog: "{{.MsgBacklog}}"
        activationMsgBacklogThreshold: "5"
        adminURL: http://{{.StatefulSetName}}.{{.TestNamespace}}:8080
        topic:  persistent://public/default/keda
        subscription: keda
          `

const publishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.ProducerJobName}}
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: pulsar-client
        image: ghcr.io/pulsar-sigs/pulsar-client:v0.3.1
        imagePullPolicy: IfNotPresent
        args: ["producer", "--broker","pulsar://{{.StatefulSetName}}.{{.TestNamespace}}:6650","--topic","persistent://public/default/keda","--message-num","{{.MessageCount}}"]
      restartPolicy: Never
  backoffLimit: 4
`

const serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.StatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: pulsar
    port: 6650
    targetPort: 6650
    protocol: TCP
  selector:
    app: pulsar
`

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:          testNamespace,
			StatefulSetName:        statefulSetName,
			MessageCount:           messageCount,
			ScaledObjectName:       scaledObjectName,
			ConsumerDeploymentName: consumerDeploymentName,
			ProducerJobName:        producerJobName,
			MinReplicaCount:        minReplicaCount,
			MaxReplicaCount:        maxReplicaCount,
			MsgBacklog:             msgBacklog,
		}, []Template{
			{Name: "statefulsetTemplate", Config: statefulsetTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
		}
}

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, statefulSetName, testNamespace, 1, 300, 1),
		"replica count should be 1 after 5 minute")

	KubectlApplyWithTemplate(t, data, "consumerTemplate", consumerTemplate)

	// run consumer for create subcription
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, testNamespace, 1, 300, 1),
		"replica count should be 1 after 5 minute")

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")

	testActivation(t, kc, data)
	// scale up
	testScaleUp(t, kc, data)
	// scale down
	testScaleDown(t, kc)

	// cleanup
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "publishJobTemplate", publishJobTemplate)

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	// publish message and less than MsgBacklog
	KubectlApplyWithTemplate(t, data, "publishJobTemplate", publishJobTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, consumerDeploymentName, testNamespace, minReplicaCount, 60)
	KubectlDeleteWithTemplate(t, data, "publishJobTemplate", publishJobTemplate)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	data.MessageCount = 100
	KubectlApplyWithTemplate(t, data, "publishJobTemplate", publishJobTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, testNamespace, 5, 300, 1),
		"replica count should be 5 after 5 minute")
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")
	// Check if deployment scale down to 0 after 5 minutes
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, consumerDeploymentName, testNamespace, 0, 300, 1),
		"Replica count should be 0 after 5 minutes")
}

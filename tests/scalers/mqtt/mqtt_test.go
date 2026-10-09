//go:build e2e
// +build e2e

package mqtt_test

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
	testName = "mqtt-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	mqttTopic        = "keda/test"
	mqttBrokerHost   = fmt.Sprintf("tcp://mosquitto.%s:1883", testNamespace)
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	MqttBrokerHost   string
	MqttTopic        string
}

const (
	mosquittoConfigTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: mosquitto-config
  namespace: {{.TestNamespace}}
data:
  mosquitto.conf: |
    listener 1883
    allow_anonymous true
`

	mosquittoDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mosquitto
  namespace: {{.TestNamespace}}
  labels:
    app: mosquitto
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mosquitto
  template:
    metadata:
      labels:
        app: mosquitto
    spec:
      containers:
      - name: mosquitto
        image: eclipse-mosquitto:2.0
        ports:
        - containerPort: 1883
        volumeMounts:
        - name: config
          mountPath: /mosquitto/config/mosquitto.conf
          subPath: mosquitto.conf
      volumes:
      - name: config
        configMap:
          name: mosquitto-config
`

	mosquittoServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: mosquitto
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: mosquitto
  ports:
  - port: 1883
    targetPort: 1883
`

	deploymentTemplate = `apiVersion: apps/v1
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
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 5
  cooldownPeriod: 10
  triggers:
  - type: mqtt
    metadata:
      brokerAddress: {{.MqttBrokerHost}}
      topic: {{.MqttTopic}}
      queryValue: "5"
      windowSeconds: "15"
`

	// The publisher sends one message per second for MessageCount
	// seconds rather than a single burst. The scaler subscribes
	// asynchronously after the ScaledObject is created, so a one-shot
	// burst could be delivered before the subscription exists and never
	// be counted; a steady stream is observed as soon as it does.
	publishJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: mqtt-publisher-{{.JobSuffix}}
  namespace: {{.TestNamespace}}
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: publisher
        image: eclipse-mosquitto:2.0
        command:
        - sh
        - -c
        - |
          for i in $(seq 1 {{.MessageCount}}); do
            mosquitto_pub -h mosquitto.{{.TestNamespace}} -t {{.MqttTopic}} -m "msg-$i"
            sleep 1
          done
`
)

func TestMqttScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "mosquitto", testNamespace, 1, 60, 3),
		"mosquitto broker should be up within 3 minutes")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out: publishing messages ---")
	// ~15 messages per 15s window against queryValue 5 asks for 3
	// replicas, capped at maxReplicaCount.
	jobData := publishJobData("scaleout", 300)
	KubectlApplyWithTemplate(t, jobData, "publishJobTemplate", publishJobTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)

	// Stop publishing so the window can drain for the scale-in phase.
	KubectlDeleteWithTemplate(t, jobData, "publishJobTemplate", publishJobTemplate)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in: waiting for window to drain ---")
	// No new messages; after the 15s window expires and pollingInterval ticks,
	// the count drops to 0 and KEDA scales back to 0.
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 120, 3),
		"replica count should be %d after 6 minutes", minReplicaCount)
}

type publishJobTemplateData struct {
	templateData
	JobSuffix    string
	MessageCount int
}

func publishJobData(suffix string, count int) publishJobTemplateData {
	return publishJobTemplateData{
		templateData: templateData{
			TestNamespace: testNamespace,
			MqttTopic:     mqttTopic,
		},
		JobSuffix:    suffix,
		MessageCount: count,
	}
}

func getTemplateData() (templateData, []Template) {
	data := templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		MqttBrokerHost:   mqttBrokerHost,
		MqttTopic:        mqttTopic,
	}
	return data, []Template{
		{Name: "mosquittoConfigTemplate", Config: mosquittoConfigTemplate},
		{Name: "mosquittoDeploymentTemplate", Config: mosquittoDeploymentTemplate},
		{Name: "mosquittoServiceTemplate", Config: mosquittoServiceTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

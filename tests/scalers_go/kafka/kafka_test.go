//go:build e2e
// +build e2e

package kafka_test

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
	testName = "kafka-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	kafkaName              = fmt.Sprintf("%s-kafka", testName)
	kafkaClientName        = fmt.Sprintf("%s-client", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	bootstrapServer        = fmt.Sprintf("%s-kafka-bootstrap.%s:9092", kafkaName, testNamespace)
	strimziOperatorVersion = "0.23.0"
	minReplicaCount        = 0
	maxReplicaCount        = 2
	topic1                 = "kafka-topic"
	topic2                 = "kafka-topic2"
)

type templateData struct {
	TestNamespace        string
	DeploymentName       string
	ScaledObjectName     string
	KafkaName            string
	KafkaTopicName       string
	KafkaTopicPartitions int
	KafkaClientName      string
	TopicName            string
	Topic1Name           string
	Topic2Name           string
	BootstrapServer      string
	ResetPolicy          string
	MinReplicaCount      int
	MaxReplicaCount      int
}

type templateValues map[string]string

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
      app: kafka-consumer
  template:
    metadata:
      labels:
        app: kafka-consumer
    spec:
      containers:
      # only recent version of kafka-console-consumer support flag "include"
      # old version's equiv flag will violate language-matters commit hook
      # work around -> create two consumer container joining the same group
      - name: kafka-consumer
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server {{.BootstrapServer}} --topic '{{.Topic1Name}}'  --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"  
      - name: kafka-consumer-2
        image: confluentinc/cp-kafka:5.2.1
        command:
          - sh
          - -c
          - "kafka-console-consumer --bootstrap-server {{.BootstrapServer}} --topic '{{.Topic2Name}}' --group multiTopic --from-beginning --consumer-property enable.auto.commit=false"
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
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
  - type: kafka
    metadata:
      topic: {{.TopicName}}
      bootstrapServers: {{.BootstrapServer}}
      consumerGroup: {{.ResetPolicy}}
      lagThreshold: '1'
      offsetResetPolicy: {{.ResetPolicy}}`

	kafkaClusterTemplate = `apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: {{.KafkaName}}
  namespace: {{.TestNamespace}}
spec:
  kafka:
    version: "2.6.0"
    replicas: 1
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: tls
        port: 9093
        type: internal
        tls: true
    config:
      offsets.topic.replication.factor: 1
      transaction.state.log.replication.factor: 1
      transaction.state.log.min.isr: 1
      log.message.format.version: "2.5"
    storage:
      type: ephemeral
  zookeeper:
    replicas: 1
    storage:
      type: ephemeral
  entityOperator:
    topicOperator: {}
    userOperator: {}
`

	kafkaTopicTemplate = `apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: {{.KafkaTopicName}}
  namespace: {{.TestNamespace}}
  labels:
    strimzi.io/cluster: {{.KafkaName}}
  namespace: {{.TestNamespace}}
spec:
  partitions: {{.KafkaTopicPartitions}}
  replicas: 1
  config:
    retention.ms: 604800000
    segment.bytes: 1073741824
`
	kafkaClientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.KafkaClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.KafkaClientName}}
    image: confluentinc/cp-kafka:5.2.1
    command:
      - sh
      - -c
      - "exec tail -f /dev/null`
)

func TestDatadogScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	installKafkaOperator(t)
	addCluster(t, data)
	addTopic(t, data, topic1, 3)
	addTopic(t, data, topic2, 3)

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testScaleUp(t, kc, data)
	testScaleDown(t, kc, data)

	// cleanup
	uninstallKafkaOperator(t)
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale up ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale down ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func installKafkaOperator(t *testing.T) {
	_, err := ExecuteCommand("helm repo add strimzi https://strimzi.io/charts/")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --namespace %s --wait %s strimzi/strimzi-kafka-operator --version %s`,
		testNamespace,
		testName,
		strimziOperatorVersion))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func uninstallKafkaOperator(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --namespace %s --wait %s`,
		testNamespace,
		testName))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func addTopic(t *testing.T, data templateData, name string, partitions int) {
	data.KafkaTopicName = name
	data.KafkaTopicPartitions = partitions
	KubectlApplyWithTemplate(t, data, "kafkaTopicTemplate", kafkaTopicTemplate)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl wait kafkatopic/%s --for=condition=Ready --timeout=300s --namespace %s", name, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func addCluster(t *testing.T, data templateData) {
	KubectlApplyWithTemplate(t, data, "kafkaClusterTemplate", kafkaClusterTemplate)
	_, err := ExecuteCommand(fmt.Sprintf("kubectl wait kafka/%s --for=condition=Ready --timeout=300s --namespace %s", kafkaName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getTemplateData() (templateData, map[string]string) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			KafkaName:        kafkaName,
			KafkaClientName:  kafkaClientName,
			BootstrapServer:  bootstrapServer,
			TopicName:        topic1,
			Topic1Name:       topic1,
			Topic2Name:       topic2,
			ResetPolicy:      "earliest",
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, templateValues{
			"deploymentTemplate": deploymentTemplate,
		}
}

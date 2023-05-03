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
	testNamespace                = fmt.Sprintf("%s-ns", testName)
	deploymentName               = fmt.Sprintf("%s-deployment", testName)
	kafkaName                    = fmt.Sprintf("%s-kafka", testName)
	kafkaClientName              = fmt.Sprintf("%s-client", testName)
	scaledObjectName             = fmt.Sprintf("%s-so", testName)
	bootstrapServer              = fmt.Sprintf("%s-kafka-bootstrap.%s:9092", kafkaName, testNamespace)
	strimziOperatorVersion       = "0.30.0"
	topic1                       = "kafka-topic"
	topic2                       = "kafka-topic2"
	zeroInvalidOffsetTopic       = "kafka-topic-zero-invalid-offset"
	oneInvalidOffsetTopic        = "kafka-topic-one-invalid-offset"
	invalidOffsetGroup           = "invalidOffset"
	persistentLagTopic           = "kafka-topic-persistent-lag"
	persistentLagGroup           = "persistentLag"
	persistentLagDeploymentGroup = "persistentLagDeploymentGroup"
	topicPartitions              = 3
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
	Params               string
	Commit               string
	ScaleToZeroOnInvalid string
	ExcludePersistentLag string
}

const (
	singleDeploymentTemplate = `
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
          - "kafka-console-consumer --bootstrap-server {{.BootstrapServer}} {{.Params}} --consumer-property enable.auto.commit={{.Commit}}"
`

	multiDeploymentTemplate = `
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

	singleScaledObjectTemplate = `
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
      activationLagThreshold: '1'
      offsetResetPolicy: {{.ResetPolicy}}`

	multiScaledObjectTemplate = `
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
      bootstrapServers: {{.BootstrapServer}}
      consumerGroup: multiTopic
      lagThreshold: '1'
      offsetResetPolicy: 'latest'`

	invalidOffsetScaledObjectTemplate = `
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
      scaleToZeroOnInvalidOffset: '{{.ScaleToZeroOnInvalid}}'
      offsetResetPolicy: 'latest'`

	persistentLagScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 15
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 30
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 30
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: kafka
    metadata:
      topic: {{.TopicName}}
      bootstrapServers: {{.BootstrapServer}}
      consumerGroup: {{.ResetPolicy}}
      lagThreshold: '1'
      excludePersistentLag: '{{.ExcludePersistentLag}}'
      offsetResetPolicy: 'latest'`

	kafkaClusterTemplate = `apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: {{.KafkaName}}
  namespace: {{.TestNamespace}}
spec:
  kafka:
    version: "3.1.0"
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
      - "exec tail -f /dev/null"`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	installKafkaOperator(t)
	addCluster(t, data)
	addTopic(t, data, topic1, topicPartitions)
	addTopic(t, data, topic2, topicPartitions)
	addTopic(t, data, zeroInvalidOffsetTopic, 1)
	addTopic(t, data, oneInvalidOffsetTopic, 1)
	addTopic(t, data, persistentLagTopic, topicPartitions)

	// test scaling
	testEarliestPolicy(t, kc, data)
	testLatestPolicy(t, kc, data)
	testMultiTopic(t, kc, data)
	testZeroOnInvalidOffset(t, kc, data)
	testOneOnInvalidOffset(t, kc, data)
	testPersistentLag(t, kc, data)

	// cleanup
	uninstallKafkaOperator(t)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testEarliestPolicy(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing earliest policy: scale out ---")
	data.Params = fmt.Sprintf("--topic %s --group earliest --from-beginning", topic1)
	data.Commit = StringFalse
	data.TopicName = topic1
	data.ResetPolicy = "earliest"
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "singleScaledObjectTemplate", singleScaledObjectTemplate)

	// Shouldn't scale pods applying earliest policy
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	// Shouldn't scale pods with only 1 message due to activation value
	publishMessage(t, topic1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	// Scale application with kafka messages
	publishMessage(t, topic1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be %d after 2 minute", 2)

	// Scale application beyond partition max.
	messages := 5
	for i := 0; i < messages; i++ {
		publishMessage(t, topic1)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, topicPartitions, 60, 2),
		"replica count should be %d after 2 minute", messages)

	KubectlDeleteWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "singleScaledObjectTemplate", singleScaledObjectTemplate)
}

func testLatestPolicy(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing latest policy: scale out ---")
	commitPartition(t, topic1, "latest")
	data.Params = fmt.Sprintf("--topic %s --group latest", topic1)
	data.Commit = StringFalse
	data.TopicName = topic1
	data.ResetPolicy = "latest"
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "singleScaledObjectTemplate", singleScaledObjectTemplate)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	// Shouldn't scale pods with only 1 message due to activation value
	publishMessage(t, topic1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	// Scale application with kafka messages
	publishMessage(t, topic1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be %d after 2 minute", 2)

	// Scale application beyond partition max.
	messages := 5
	for i := 0; i < messages; i++ {
		publishMessage(t, topic1)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, topicPartitions, 60, 2),
		"replica count should be %d after 2 minute", messages)

	KubectlDeleteWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "singleScaledObjectTemplate", singleScaledObjectTemplate)
}

func testMultiTopic(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing multi topic: scale out ---")
	commitPartition(t, topic1, "multiTopic")
	commitPartition(t, topic2, "multiTopic")
	data.Topic1Name = topic1
	data.Topic2Name = topic2
	KubectlApplyWithTemplate(t, data, "multiDeploymentTemplate", multiDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "multiScaledObjectTemplate", multiScaledObjectTemplate)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	// Scale application with kafka messages in topic 1
	publishMessage(t, topic1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be %d after 2 minute", 1)

	// Scale application with kafka messages in topic 2
	// // produce one more msg to the different topic within the same group
	// // will turn total consumer group lag to 2.
	// // with lagThreshold as 1 -> making hpa AverageValue to 1
	// // this should turn nb of replicas to 2
	// // as desiredReplicaCount = totalLag / avgThreshold
	publishMessage(t, topic2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be %d after 2 minute", 2)

	KubectlDeleteWithTemplate(t, data, "multiDeploymentTemplate", multiDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "multiScaledObjectTemplate", multiScaledObjectTemplate)
}

func testZeroOnInvalidOffset(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing zeroInvalidOffsetTopic: scale out ---")
	data.Params = fmt.Sprintf("--topic %s --group %s", zeroInvalidOffsetTopic, invalidOffsetGroup)
	data.Commit = StringTrue
	data.TopicName = zeroInvalidOffsetTopic
	data.ResetPolicy = invalidOffsetGroup
	data.ScaleToZeroOnInvalid = StringTrue
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)

	KubectlDeleteWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)
}

func testOneOnInvalidOffset(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing oneInvalidOffsetTopic: scale out ---")
	data.Params = fmt.Sprintf("--topic %s --group %s --from-beginning", oneInvalidOffsetTopic, invalidOffsetGroup)
	data.Commit = StringTrue
	data.TopicName = oneInvalidOffsetTopic
	data.ResetPolicy = invalidOffsetGroup
	data.ScaleToZeroOnInvalid = StringFalse
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)

	// Should scale to 1
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be %d after 2 minute", 1)

	commitPartition(t, oneInvalidOffsetTopic, invalidOffsetGroup)
	publishMessage(t, oneInvalidOffsetTopic)

	// Should scale to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 10),
		"replica count should be %d after 10 minute", 0)

	KubectlDeleteWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)
}

func publishMessage(t *testing.T, topic string) {
	_, _, err := ExecCommandOnSpecificPod(t, kafkaClientName, testNamespace, fmt.Sprintf(`echo "{"text": "foo"}" | kafka-console-producer --broker-list %s --topic %s`, bootstrapServer, topic))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func commitPartition(t *testing.T, topic string, group string) {
	_, _, err := ExecCommandOnSpecificPod(t, kafkaClientName, testNamespace, fmt.Sprintf(`kafka-console-consumer --bootstrap-server %s --topic %s --group %s --from-beginning --consumer-property enable.auto.commit=true --timeout-ms 15000`, bootstrapServer, topic, group))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testPersistentLag(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing  persistentLag: no scale out ---")

	// Simulate Consumption from topic by consumer group
	// To avoid edge case where where scaling could be effectively disabled (Consumer never makes a commit)
	data.Params = fmt.Sprintf("--topic %s --group %s --from-beginning", persistentLagTopic, persistentLagGroup)
	data.Commit = StringTrue
	data.TopicName = persistentLagTopic
	data.ResetPolicy = persistentLagGroup
	data.ExcludePersistentLag = StringTrue
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "persistentLagScaledObjectTemplate", persistentLagScaledObjectTemplate)

	// Scale application with kafka messages in persistentLagTopic
	publishMessage(t, persistentLagTopic)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be %d after 2 minute", 1)
	// Recreate Deployment to delibrately assign different consumer group to deployment and scaled object
	// This is to simulate inability to consume from topic
	// Scaled Object remains unchanged
	KubernetesScaleDeployment(t, kc, deploymentName, 0, testNamespace)
	assert.True(t, WaitForPodsTerminated(t, kc, "app=kafka-consumer", testNamespace, 60, 2),
		"pod should be terminated after %d minute", 2)

	data.Params = fmt.Sprintf("--topic %s --group %s --from-beginning", persistentLagTopic, persistentLagDeploymentGroup)
	KubectlApplyWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)

	messages := 5
	for i := 0; i < messages; i++ {
		publishMessage(t, persistentLagTopic)
	}

	// Persistent Lag should not scale pod above minimum replicas after 2 reconciliation cycles
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be %d after 2 minute", 1)

	// Shouldn't scale pods
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 1, 30)

	KubectlDeleteWithTemplate(t, data, "singleDeploymentTemplate", singleDeploymentTemplate)
	KubectlDeleteWithTemplate(t, data, "persistentLagScaledObjectTemplate", persistentLagScaledObjectTemplate)
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
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --namespace %s %s`,
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

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			KafkaName:        kafkaName,
			KafkaClientName:  kafkaClientName,
			BootstrapServer:  bootstrapServer,
			TopicName:        topic1,
			Topic1Name:       topic1,
			Topic2Name:       topic2,
			ResetPolicy:      "",
			ScaledObjectName: scaledObjectName,
		}, []Template{
			{Name: "kafkaClientTemplate", Config: kafkaClientTemplate},
		}
}

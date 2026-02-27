//go:build e2e
// +build e2e

package apache_iggy_test

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
	testName = "apache-iggy-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	iggyServerName   = fmt.Sprintf("%s-server", testName)
	iggyClientName   = fmt.Sprintf("%s-client", testName)
	iggyServiceName  = fmt.Sprintf("%s-service", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)

	iggyServerAddress = fmt.Sprintf("%s.%s.svc.cluster.local:8090", iggyServiceName, testNamespace)
	iggyImage         = "apache/iggy:0.6.0"

	// Stream
	streamID   = "1"
	streamName = "test-stream"

	// Topics — each test scenario gets its own topic to avoid cross-test interference
	basicTopicID    = "1"
	basicTopicName  = "basic-topic"
	topicPartitions = 3

	zeroInvalidOffsetTopicID   = "2"
	zeroInvalidOffsetTopicName = "zero-invalid-offset-topic"

	oneInvalidOffsetTopicID   = "3"
	oneInvalidOffsetTopicName = "one-invalid-offset-topic"

	persistentLagTopicID   = "4"
	persistentLagTopicName = "persistent-lag-topic"

	limitPartitionsTopicID   = "5"
	limitPartitionsTopicName = "limit-partitions-topic"

	evenDistributionTopicID         = "6"
	evenDistributionTopicName       = "even-distribution-topic"
	evenDistributionTopicPartitions = 10

	// Consumer groups — one per test scenario
	basicGroupID = "1"
	basicGroup   = "basic-group"

	zeroInvalidGroupID = "2"
	zeroInvalidGroup   = "zero-invalid-group"

	oneInvalidGroupID = "3"
	oneInvalidGroup   = "one-invalid-group"

	persistentLagGroupID = "4"
	persistentLagGroup   = "persistent-lag-group"

	limitPartitionsGroupID = "5"
	limitPartitionsGroup   = "limit-partitions-group"

	evenDistributionGroupID = "6"
	evenDistributionGroup   = "even-distribution-group"
)

type templateData struct {
	TestNamespace                      string
	DeploymentName                     string
	ScaledObjectName                   string
	IggyServerName                     string
	IggyClientName                     string
	IggyServiceName                    string
	IggyServerAddress                  string
	IggyImage                          string
	SecretName                         string
	TriggerAuthName                    string
	StreamID                           string
	TopicID                            string
	ConsumerGroupID                    string
	ScaleToZeroOnInvalid               string
	ExcludePersistentLag               string
	LimitToPartitionsWithLag           string
	EnsureEvenDistributionOfPartitions string
}

const (
	iggyServerDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.IggyServerName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.IggyServerName}}
  template:
    metadata:
      labels:
        app: {{.IggyServerName}}
    spec:
      containers:
      - name: iggy-server
        image: {{.IggyImage}}
        ports:
        - containerPort: 8090
          name: tcp
        readinessProbe:
          tcpSocket:
            port: 8090
          initialDelaySeconds: 5
          periodSeconds: 5
`

	iggyServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.IggyServiceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.IggyServerName}}
  ports:
  - port: 8090
    targetPort: 8090
    name: tcp
`

	iggyClientPodTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: {{.IggyClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: iggy-client
    image: {{.IggyImage}}
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  username: iggy
  password: iggy
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: username
    name: {{.SecretName}}
    key: username
  - parameter: password
    name: {{.SecretName}}
    key: password
`

	targetDeploymentTemplate = `apiVersion: apps/v1
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
      - name: consumer
        image: busybox
        command: ["sleep", "infinity"]
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 0
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: apache-iggy
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      serverAddress: {{.IggyServerAddress}}
      streamId: '{{.StreamID}}'
      topicId: '{{.TopicID}}'
      consumerGroupId: '{{.ConsumerGroupID}}'
      lagThreshold: '1'
      activationLagThreshold: '1'
`

	// Note: activationLagThreshold is intentionally omitted (defaults to 0) so that
	// lag=1 from the invalid-offset fallback path triggers activation (1 > 0 = true).
	invalidOffsetScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 0
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: apache-iggy
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      serverAddress: {{.IggyServerAddress}}
      streamId: '{{.StreamID}}'
      topicId: '{{.TopicID}}'
      consumerGroupId: '{{.ConsumerGroupID}}'
      lagThreshold: '1'
      scaleToZeroOnInvalidOffset: '{{.ScaleToZeroOnInvalid}}'
`

	persistentLagScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 0
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: apache-iggy
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      serverAddress: {{.IggyServerAddress}}
      streamId: '{{.StreamID}}'
      topicId: '{{.TopicID}}'
      consumerGroupId: '{{.ConsumerGroupID}}'
      lagThreshold: '1'
      excludePersistentLag: '{{.ExcludePersistentLag}}'
`

	limitPartitionsScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 0
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: apache-iggy
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      serverAddress: {{.IggyServerAddress}}
      streamId: '{{.StreamID}}'
      topicId: '{{.TopicID}}'
      consumerGroupId: '{{.ConsumerGroupID}}'
      lagThreshold: '1'
      activationLagThreshold: '1'
      limitToPartitionsWithLag: '{{.LimitToPartitionsWithLag}}'
`

	evenDistributionScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  pollingInterval: 5
  cooldownPeriod: 0
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 0
          policies:
          - type: Percent
            value: 100
            periodSeconds: 15
  triggers:
  - type: apache-iggy
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      serverAddress: {{.IggyServerAddress}}
      streamId: '{{.StreamID}}'
      topicId: '{{.TopicID}}'
      consumerGroupId: '{{.ConsumerGroupID}}'
      lagThreshold: '1'
      activationLagThreshold: '1'
      ensureEvenDistributionOfPartitions: '{{.EnsureEvenDistributionOfPartitions}}'
`
)

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Wait for iggy server to be ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, iggyServerName, testNamespace, 1, 60, 2),
		"iggy server should be ready")
	// Wait for iggy client pod to be ready
	assert.True(t, WaitForPodReady(t, kc, iggyClientName, testNamespace, 60, 2),
		"iggy client pod should be ready")

	// Wait for the server to be fully accepting connections
	ok, _, _, err := WaitForSuccessfulExecCommandOnSpecificPod(t, iggyClientName, testNamespace,
		fmt.Sprintf("iggy --tcp-server-address %s -u iggy -p iggy ping", iggyServerAddress), 30, 2)
	require.True(t, ok, "iggy server should respond to ping")
	require.NoError(t, err)

	// Create stream
	iggyCreateStream(t)

	// Create topics (each test gets its own topic)
	iggyCreateTopic(t, basicTopicID, basicTopicName, topicPartitions)
	iggyCreateTopic(t, zeroInvalidOffsetTopicID, zeroInvalidOffsetTopicName, 1)
	iggyCreateTopic(t, oneInvalidOffsetTopicID, oneInvalidOffsetTopicName, 1)
	iggyCreateTopic(t, persistentLagTopicID, persistentLagTopicName, topicPartitions)
	iggyCreateTopic(t, limitPartitionsTopicID, limitPartitionsTopicName, topicPartitions)
	iggyCreateTopic(t, evenDistributionTopicID, evenDistributionTopicName, evenDistributionTopicPartitions)

	// Create consumer groups (one per test scenario)
	iggyCreateConsumerGroup(t, basicTopicID, basicGroupID, basicGroup)
	iggyCreateConsumerGroup(t, zeroInvalidOffsetTopicID, zeroInvalidGroupID, zeroInvalidGroup)
	iggyCreateConsumerGroup(t, oneInvalidOffsetTopicID, oneInvalidGroupID, oneInvalidGroup)
	iggyCreateConsumerGroup(t, persistentLagTopicID, persistentLagGroupID, persistentLagGroup)
	iggyCreateConsumerGroup(t, limitPartitionsTopicID, limitPartitionsGroupID, limitPartitionsGroup)
	iggyCreateConsumerGroup(t, evenDistributionTopicID, evenDistributionGroupID, evenDistributionGroup)

	// Test scenarios
	testBasicLagScaling(t, kc, data)
	testScaleToZeroOnInvalidOffset(t, kc, data)
	testOneOnInvalidOffset(t, kc, data)
	testPersistentLag(t, kc, data)
	testLimitToPartitionsWithLag(t, kc, data)
	testEnsureEvenDistributionOfPartitions(t, kc, data)
}

// --- Test scenarios ---

func testBasicLagScaling(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing basic lag scaling ---")
	data.TopicID = basicTopicID
	data.ConsumerGroupID = basicGroupID

	// Initialize consumer group offsets at 0 for all partitions.
	// On an empty topic CurrentOffset=0, so lag = max(0-0, 0) = 0.
	iggyStoreConsumerOffsetAll(t, basicTopicID, basicGroupID, topicPartitions)

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// No messages published yet, lag=0 across all partitions — should stay at 0
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 1 message to partition 1 — total lag=1, activationLagThreshold=1, 1 is NOT > 1
	iggyPublishMessage(t, basicTopicID, 1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 1 more to partition 2 — total lag=2, 2 > 1 → active, scale to 2
	iggyPublishMessage(t, basicTopicID, 2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2")

	// Publish 5 more spread across partitions — total lag=7, but capped at partition count (3)
	for i := 0; i < 5; i++ {
		iggyPublishMessage(t, basicTopicID, (i%topicPartitions)+1)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, topicPartitions, 60, 2),
		"replica count should be capped at partition count %d", topicPartitions)
}

func testScaleToZeroOnInvalidOffset(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaleToZeroOnInvalidOffset ---")
	data.TopicID = zeroInvalidOffsetTopicID
	data.ConsumerGroupID = zeroInvalidGroupID
	data.ScaleToZeroOnInvalid = StringTrue

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)

	// No committed offsets, scaleToZeroOnInvalidOffset=true → lag=0 → inactive → stay at 0
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
}

func testOneOnInvalidOffset(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing oneOnInvalidOffset ---")
	data.TopicID = oneInvalidOffsetTopicID
	data.ConsumerGroupID = oneInvalidGroupID
	data.ScaleToZeroOnInvalid = StringFalse

	// Publish 1 message so latest offset is not 0
	iggyPublishMessage(t, oneInvalidOffsetTopicID, 1)

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "invalidOffsetScaledObjectTemplate", invalidOffsetScaledObjectTemplate)

	// No committed offsets, scaleToZeroOnInvalidOffset=false → lag=1 per partition → active → scale to 1
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1")

	// Store a very large offset to guarantee stored >= current regardless of the topic's
	// actual current offset value. This ensures lag = max(current - stored, 0) = 0.
	// The Iggy CLI requires explicit offset values (no "commit to current" shorthand).
	iggyStoreConsumerOffset(t, oneInvalidOffsetTopicID, oneInvalidGroupID, 1, 999999)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 10),
		"replica count should be 0")
}

func testPersistentLag(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing persistent lag ---")

	// Store initial offsets at 0 for all partitions to establish baseline
	iggyStoreConsumerOffsetAll(t, persistentLagTopicID, persistentLagGroupID, topicPartitions)

	data.TopicID = persistentLagTopicID
	data.ConsumerGroupID = persistentLagGroupID
	data.ExcludePersistentLag = StringTrue

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "persistentLagScaledObjectTemplate", persistentLagScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "persistentLagScaledObjectTemplate", persistentLagScaledObjectTemplate)

	// Publish messages to create lag. First poll records previousOffset per partition.
	iggyPublishMessage(t, persistentLagTopicID, 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 after initial lag")

	// Publish more messages WITHOUT updating offsets — simulates stuck consumer.
	for i := 0; i < 5; i++ {
		iggyPublishMessage(t, persistentLagTopicID, (i%topicPartitions)+1)
	}

	// After 2+ polling cycles, scaler detects storedOffset hasn't changed → persistent lag.
	// totalLag=0 (persistent excluded), but totalLagWithPersistent > 0 so isActive=true → stays at 1.
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 1, 30)
}

func testLimitToPartitionsWithLag(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing limitToPartitionsWithLag ---")

	// Store offsets at 0 on empty topic — lag=0
	iggyStoreConsumerOffsetAll(t, limitPartitionsTopicID, limitPartitionsGroupID, topicPartitions)

	data.TopicID = limitPartitionsTopicID
	data.ConsumerGroupID = limitPartitionsGroupID
	data.LimitToPartitionsWithLag = StringTrue

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "limitPartitionsScaledObjectTemplate", limitPartitionsScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "limitPartitionsScaledObjectTemplate", limitPartitionsScaledObjectTemplate)

	// No lag — stay at 0
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 1 message to partition 1 — total lag=1, 1 is NOT > activationLagThreshold(1)
	iggyPublishMessage(t, limitPartitionsTopicID, 1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 5 more to partition 1 — total lag on partition 1 is high,
	// but limitToPartitionsWithLag caps replicas at partitions-with-lag (1) → scale to 1
	for i := 0; i < 5; i++ {
		iggyPublishMessage(t, limitPartitionsTopicID, 1)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 with lag on 1 partition")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 1, 60)

	// Publish 5 messages to partition 2 — now 2 partitions have lag → scale to 2
	for i := 0; i < 5; i++ {
		iggyPublishMessage(t, limitPartitionsTopicID, 2)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 with lag on 2 partitions")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 2, 60)
}

func testEnsureEvenDistributionOfPartitions(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing ensureEvenDistributionOfPartitions ---")

	// Store offsets at 0 on empty topic — lag=0
	iggyStoreConsumerOffsetAll(t, evenDistributionTopicID, evenDistributionGroupID, evenDistributionTopicPartitions)

	data.TopicID = evenDistributionTopicID
	data.ConsumerGroupID = evenDistributionGroupID
	data.EnsureEvenDistributionOfPartitions = StringTrue

	KubectlApplyWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "targetDeploymentTemplate", targetDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "evenDistributionScaledObjectTemplate", evenDistributionScaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "evenDistributionScaledObjectTemplate", evenDistributionScaledObjectTemplate)

	// No lag — stay at 0
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 1 message — lag=1, 1 is NOT > activationLagThreshold(1) → stay at 0
	iggyPublishMessage(t, evenDistributionTopicID, 1)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Publish 4 more — total lag=5
	// Factors of 10: 1, 2, 5, 10. desiredReplicas from lag=5/threshold=1 = 5.
	// 5 is a factor of 10, so ensureEvenDistribution picks 5 → scale to 5
	for i := 0; i < 4; i++ {
		iggyPublishMessage(t, evenDistributionTopicID, (i%evenDistributionTopicPartitions)+1)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, 2),
		"replica count should be 5 (factor of 10)")

	// Publish 3 more — total lag=8
	// desiredReplicas from lag=8/threshold=1 = 8. Next factor of 10 >= 8 is 10 → scale to 10
	for i := 0; i < 3; i++ {
		iggyPublishMessage(t, evenDistributionTopicID, (i%evenDistributionTopicPartitions)+1)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 10, 60, 2),
		"replica count should be 10 (next factor of 10 after 8)")
}

// --- Helper functions ---

func iggyCmd(t *testing.T, args string) (string, string) {
	cmd := fmt.Sprintf("iggy --tcp-server-address %s -u iggy -p iggy %s", iggyServerAddress, args)
	stdout, stderr, err := ExecCommandOnSpecificPod(t, iggyClientName, testNamespace, cmd)
	assert.NoErrorf(t, err, "iggy command failed: %s\nstdout: %s\nstderr: %s", cmd, stdout, stderr)
	return stdout, stderr
}

func iggyCreateStream(t *testing.T) {
	t.Log("--- creating iggy stream ---")
	iggyCmd(t, fmt.Sprintf("stream create -s %s %s", streamID, streamName))
}

func iggyCreateTopic(t *testing.T, topicID, topicName string, partitions int) {
	t.Logf("--- creating iggy topic %s with %d partitions ---", topicName, partitions)
	iggyCmd(t, fmt.Sprintf("topic create -t %s %s %s %d none", topicID, streamID, topicName, partitions))
}

func iggyCreateConsumerGroup(t *testing.T, topicID, groupID, groupName string) {
	t.Logf("--- creating iggy consumer group %s for topic %s ---", groupName, topicID)
	iggyCmd(t, fmt.Sprintf("consumer-group create -g %s %s %s %s", groupID, streamID, topicID, groupName))
}

func iggyPublishMessage(t *testing.T, topicID string, partitionID int) {
	iggyCmd(t, fmt.Sprintf("message send -p %d %s %s \"test-message\"", partitionID, streamID, topicID))
}

func iggyStoreConsumerOffset(t *testing.T, topicID, groupID string, partitionID, offset int) {
	iggyCmd(t, fmt.Sprintf("consumer-offset set -k consumer-group %s %s %s %d %d", groupID, streamID, topicID, partitionID, offset))
}

func iggyStoreConsumerOffsetAll(t *testing.T, topicID, groupID string, partitions int) {
	for i := 1; i <= partitions; i++ {
		iggyStoreConsumerOffset(t, topicID, groupID, i, 0)
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:     testNamespace,
			DeploymentName:    deploymentName,
			ScaledObjectName:  scaledObjectName,
			IggyServerName:    iggyServerName,
			IggyClientName:    iggyClientName,
			IggyServiceName:   iggyServiceName,
			IggyServerAddress: iggyServerAddress,
			IggyImage:         iggyImage,
			SecretName:        secretName,
			TriggerAuthName:   triggerAuthName,
			StreamID:          streamID,
		}, []Template{
			{Name: "iggyServerDeploymentTemplate", Config: iggyServerDeploymentTemplate},
			{Name: "iggyServiceTemplate", Config: iggyServiceTemplate},
			{Name: "iggyClientPodTemplate", Config: iggyClientPodTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		}
}

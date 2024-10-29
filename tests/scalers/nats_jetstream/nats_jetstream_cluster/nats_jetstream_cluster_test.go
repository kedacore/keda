//go:build e2e
// +build e2e

package natsjetstream_cluster_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8s "k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	nats "github.com/kedacore/keda/v2/tests/scalers/nats_jetstream/helper"
)

// Load env variables from .env files
var _ = godotenv.Load("../../.env")

const (
	testName = "nats-jetstream-cluster"
)

var (
	testNamespace                        = fmt.Sprintf("%s-test-ns", testName)
	natsNamespace                        = fmt.Sprintf("%s-nats-ns", testName)
	natsAddress                          = fmt.Sprintf("nats://%s.%s.svc.cluster.local:4222", nats.NatsJetStreamName, natsNamespace)
	natsServerMonitoringEndpoint         = fmt.Sprintf("%s.%s.svc.cluster.local:8222", nats.NatsJetStreamName, natsNamespace)
	natsServerHeadlessMonitoringEndpoint = fmt.Sprintf("%s-headless.%s.svc.cluster.local:8222", nats.NatsJetStreamName, natsNamespace)
	natsHelmRepo                         = "https://nats-io.github.io/k8s/helm/charts/"
	natsServerReplicas                   = 3
	messagePublishCount                  = 300
	deploymentName                       = "sub"
	minReplicaCount                      = 0
	maxReplicaCount                      = 2
)

func TestNATSJetStreamScalerClusterWithStreamReplicas(t *testing.T) {
	testNATSJetStreamScalerClusterWithStreamReplicas(t, false)
}

func TestNATSJetStreamScalerClusterWithStreamReplicasWithNoAdvertise(t *testing.T) {
	testNATSJetStreamScalerClusterWithStreamReplicas(t, true)
}

func testNATSJetStreamScalerClusterWithStreamReplicas(t *testing.T, noAdvertise bool) {
	// Create k8s resources.
	kc := GetKubernetesClient(t)
	testData, testTemplates := nats.GetJetStreamDeploymentTemplateData(testNamespace, natsAddress, natsServerMonitoringEndpoint, messagePublishCount)
	t.Cleanup(func() {
		removeStreamAndConsumer(t, 1, testData.NatsStream, testNamespace, natsAddress)
		DeleteKubernetesResources(t, testNamespace, testData, testTemplates)

		removeClusterWithJetStream(t)
		DeleteNamespace(t, natsNamespace)
		deleted := WaitForNamespaceDeletion(t, natsNamespace)
		assert.Truef(t, deleted, "%s namespace not deleted", natsNamespace)
	})

	// Deploy NATS server.
	installClusterWithJetStream(t, kc, noAdvertise)
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, nats.NatsJetStreamName, natsNamespace, natsServerReplicas, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create k8s resources for testing.

	CreateKubernetesResources(t, kc, testNamespace, testData, testTemplates)

	// Create 3 replica stream with consumer
	testData.NatsStream = "case1"
	installStreamAndConsumer(t, 3, testData.NatsStream, testNamespace, natsAddress)
	KubectlApplyWithTemplate(t, testData, "scaledObjectTemplate", nats.ScaledObjectTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 3 stream replicas should be success")

	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)

	// Remove 3 replica stream with consumer
	removeStreamAndConsumer(t, 3, testData.NatsStream, testNamespace, natsAddress)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 0, 60, 3),
		"job count in namespace should be 0")

	// Create single replica stream with consumer
	testData.NatsStream = "case2"
	installStreamAndConsumer(t, 1, testData.NatsStream, testNamespace, natsAddress)
	KubectlApplyWithTemplate(t, testData, "scaledObjectTemplate", nats.ScaledObjectTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 1 stream replica should be success")

	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)
}

func TestNATSv2_10JetStreamScalerClusterWithStreamReplicas(t *testing.T) {
	// Create k8s resources.
	kc := GetKubernetesClient(t)
	testData, testTemplates := nats.GetJetStreamDeploymentTemplateData(testNamespace, natsAddress, natsServerHeadlessMonitoringEndpoint, messagePublishCount)
	t.Cleanup(func() {
		removeStreamAndConsumer(t, 1, testData.NatsStream, testNamespace, natsAddress)
		DeleteKubernetesResources(t, testNamespace, testData, testTemplates)

		removeClusterWithJetStream(t)
		DeleteNamespace(t, natsNamespace)
		deleted := WaitForNamespaceDeletion(t, natsNamespace)
		assert.Truef(t, deleted, "%s namespace not deleted", natsNamespace)
	})
	// Deploy NATS server.
	installClusterWithJetStreaV2_10(t, kc)
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, nats.NatsJetStreamName, natsNamespace, natsServerReplicas, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create k8s resources for testing.
	CreateKubernetesResources(t, kc, testNamespace, testData, testTemplates)

	// Create 3 replica stream with consumer
	testData.NatsStream = "case1"
	installStreamAndConsumer(t, 3, testData.NatsStream, testNamespace, natsAddress)
	KubectlApplyWithTemplate(t, testData, "scaledObjectTemplate", nats.ScaledObjectTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 3 stream replicas should be success")

	testActivation(t, kc, testData)
	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)

	// Remove 3 replica stream with consumer
	removeStreamAndConsumer(t, 3, testData.NatsStream, testNamespace, natsAddress)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 0, 60, 3),
		"job count in namespace should be 0")

	// Create single replica stream with consumer
	testData.NatsStream = "case2"
	installStreamAndConsumer(t, 1, testData.NatsStream, testNamespace, natsAddress)
	KubectlApplyWithTemplate(t, testData, "scaledObjectTemplate", nats.ScaledObjectTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 1 stream replica should be success")

	testActivation(t, kc, testData)
	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)
}

// installStreamAndConsumer creates stream and consumer job.
func installStreamAndConsumer(t *testing.T, streamReplicas int, stream, namespace, natsAddress string) {
	data := nats.JetStreamTemplateData{
		TestNamespace:  namespace,
		NatsAddress:    natsAddress,
		NatsConsumer:   nats.NatsJetStreamConsumerName,
		NatsStream:     stream,
		StreamReplicas: streamReplicas,
	}

	KubectlApplyWithTemplate(t, data, "streamAndConsumerTemplate", nats.StreamAndConsumerTemplate)
}

// removeStreamAndConsumer deletes stream and consumer job.
func removeStreamAndConsumer(t *testing.T, streamReplicas int, stream, namespace, natsAddress string) {
	data := nats.JetStreamTemplateData{
		TestNamespace:  namespace,
		NatsAddress:    natsAddress,
		NatsConsumer:   nats.NatsJetStreamConsumerName,
		NatsStream:     stream,
		StreamReplicas: streamReplicas,
	}

	KubectlApplyWithTemplate(t, data, "deleteStreamTemplate", nats.DeleteStreamTemplate)
}

// installClusterWithJetStream install the nats helm chart with clustered jetstream enabled
func installClusterWithJetStream(t *testing.T, kc *k8s.Clientset, noAdvertise bool) {
	removeNATSPods(t)
	CreateNamespace(t, kc, natsNamespace)
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add %s %s", nats.NatsJetStreamName, natsHelmRepo))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --version %s --set %s --set %s --set %s --set %s --set %s --wait --namespace %s %s nats/nats`,
		nats.NatsJetStreamChartVersion,
		"nats.jetstream.enabled=true",
		"nats.jetstream.fileStorage.enabled=false",
		"cluster.enabled=true",
		fmt.Sprintf("replicas=%d", natsServerReplicas),
		fmt.Sprintf("cluster.noAdvertise=%t", noAdvertise),
		natsNamespace,
		nats.NatsJetStreamName))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

// installClusterWithJetStreaV2_10 install the nats helm chart with clustered jetstream enabled using v2.10
func installClusterWithJetStreaV2_10(t *testing.T, kc *k8s.Clientset) {
	removeNATSPods(t)
	CreateNamespace(t, kc, natsNamespace)
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add %s %s", nats.NatsJetStreamName, natsHelmRepo))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --version %s --set %s --set %s --set %s --set %s --set %s --set %s --set %s --wait --namespace %s %s nats/nats`,
		nats.Natsv2_10JetStreamChartVersion,
		"config.jetstream.enabled=true",
		"config.jetstream.fileStore.enabled=false",
		"config.jetstream.memoryStore.enabled=true",
		"config.cluster.enabled=true",
		"service.enabled=true",
		"service.ports.monitor.enabled=true",
		fmt.Sprintf("config.cluster.replicas=%d", natsServerReplicas),
		natsNamespace,
		nats.NatsJetStreamName))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

// removeClusterWithJetStream uninstall the nats helm chart
func removeClusterWithJetStream(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --namespace %s %s`, natsNamespace, nats.NatsJetStreamName))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	removeNATSPods(t)
}

// removeNATSPods delete the server pods in case they're stuck
func removeNATSPods(t *testing.T) {
	DeletePodsInNamespaceBySelector(t, KubeClient, "app.kubernetes.io/name=nats", natsNamespace)
	assert.True(t, WaitForPodCountInNamespace(t, KubeClient, natsNamespace, 0, 30, 2))
}

func testActivation(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing activation ---")
	data.NumberOfMessages = 10
	KubectlApplyWithTemplate(t, data, "activationPublishJobTemplate", nats.ActivationPublishJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing scale out ---")
	// We force the change of consumer leader to ensure that KEDA detects the change and
	// handles it properly
	KubectlApplyWithTemplate(t, data, "stepDownTemplate", nats.StepDownConsumer)

	KubectlApplyWithTemplate(t, data, "publishJobTemplate", nats.PublishJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *k8s.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

//go:build e2e
// +build e2e

package natsjetstream_cluster_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
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
	testNamespace                = fmt.Sprintf("%s-test-ns", testName)
	natsNamespace                = fmt.Sprintf("%s-nats-ns", testName)
	natsAddress                  = fmt.Sprintf("nats://%s.%s.svc.cluster.local:4222", nats.NatsJetStreamName, natsNamespace)
	natsServerMonitoringEndpoint = fmt.Sprintf("%s.%s.svc.cluster.local:8222", nats.NatsJetStreamName, natsNamespace)
	natsHelmRepo                 = "https://nats-io.github.io/k8s/helm/charts/"
	natsServerReplicas           = 3
	messagePublishCount          = 300
	deploymentName               = "sub"
	minReplicaCount              = 0
	maxReplicaCount              = 2
)

func TestNATSJetStreamScalerClusterWithStreamReplicas(t *testing.T) {
	// Create k8s resources.
	kc := GetKubernetesClient(t)

	// Deploy NATS server.
	installClusterWithJetStream(t, kc)
	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, nats.NatsJetStreamName, natsNamespace, natsServerReplicas, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create k8s resources for testing.
	testData, testTemplates := nats.GetJetStreamDeploymentTemplateData(testNamespace, natsAddress, natsServerMonitoringEndpoint, messagePublishCount)
	CreateKubernetesResources(t, kc, testNamespace, testData, testTemplates)

	// Create 3 replica stream with consumer
	installStreamAndConsumer(t, 3, testNamespace, natsAddress)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 3 stream replicas should be success")

	testActivation(t, kc, testData)
	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)

	// Remove 3 replica stream with consumer
	removeStreamAndConsumer(t, 3, testNamespace, natsAddress)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 0, 60, 3),
		"job count in namespace should be 0")

	// Create stream and consumer with 2 stream replicas
	installStreamAndConsumer(t, 2, testNamespace, natsAddress)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 2 stream replicas should be success")

	testActivation(t, kc, testData)
	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)

	// Remove 2 replica stream with consumer
	removeStreamAndConsumer(t, 2, testNamespace, natsAddress)
	assert.True(t, WaitForJobCount(t, kc, testNamespace, 0, 60, 3),
		"job count in namespace should be 0")

	// Create single replica stream with consumer
	installStreamAndConsumer(t, 1, testNamespace, natsAddress)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job with 1 stream replica should be success")

	testActivation(t, kc, testData)
	testScaleOut(t, kc, testData)
	testScaleIn(t, kc)

	// Cleanup nats namespace
	removeClusterWithJetStream(t)
	DeleteNamespace(t, kc, natsNamespace)
	deleted := WaitForNamespaceDeletion(t, kc, natsNamespace)
	assert.Truef(t, deleted, "%s namespace not deleted", natsNamespace)

	// Cleanup test namespace
	removeStreamAndConsumer(t, 1, testNamespace, natsAddress)
	DeleteKubernetesResources(t, kc, testNamespace, testData, testTemplates)
}

// installStreamAndConsumer creates stream and consumer job.
func installStreamAndConsumer(t *testing.T, streamReplicas int, namespace, natsAddress string) {
	data := nats.JetStreamTemplateData{
		TestNamespace:  namespace,
		NatsAddress:    natsAddress,
		NatsConsumer:   nats.NatsJetStreamConsumerName,
		NatsStream:     nats.NatsJetStreamStreamName,
		StreamReplicas: streamReplicas,
	}

	KubectlApplyWithTemplate(t, data, "streamAndConsumerTemplate", nats.StreamAndConsumerTemplate)
}

// removeStreamAndConsumer deletes stream and consumer job.
func removeStreamAndConsumer(t *testing.T, streamReplicas int, namespace, natsAddress string) {
	data := nats.JetStreamTemplateData{
		TestNamespace:  namespace,
		NatsAddress:    natsAddress,
		NatsConsumer:   nats.NatsJetStreamConsumerName,
		NatsStream:     nats.NatsJetStreamStreamName,
		StreamReplicas: streamReplicas,
	}

	KubectlDeleteWithTemplate(t, data, "streamAndConsumerTemplate", nats.StreamAndConsumerTemplate)
}

// installClusterWithJetStream install the nats helm chart with clustered jetstream enabled
func installClusterWithJetStream(t *testing.T, kc *k8s.Clientset) {
	CreateNamespace(t, kc, natsNamespace)
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add %s %s", nats.NatsJetStreamName, natsHelmRepo))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --version %s --set %s --set %s --set %s --set %s --wait --namespace %s %s nats/nats`,
		nats.NatsJetStreamChartVersion,
		"nats.jetstream.enabled=true",
		"nats.jetstream.fileStorage.enabled=false",
		"cluster.enabled=true",
		fmt.Sprintf("replicas=%d", natsServerReplicas),
		natsNamespace,
		nats.NatsJetStreamName))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

// removeClusterWithJetStream uninstall the nats helm chart
func removeClusterWithJetStream(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --namespace %s %s`, natsNamespace, nats.NatsJetStreamName))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testActivation(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing activation ---")
	data.NumberOfMessages = 10
	KubectlApplyWithTemplate(t, data, "activationPublishJobTemplate", nats.ActivationPublishJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "publishJobTemplate", nats.PublishJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *k8s.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

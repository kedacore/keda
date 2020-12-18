package scalers

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

const (
	topicName         = "testtopic"
	subscriptionName  = "testsubscription"
	queueName         = "testqueue"
	connectionSetting = "none"
	namespaceName     = "ns"
	messageCount      = "1000"
)

type parseServiceBusMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	entityType  entityType
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type azServiceBusMetricIdentifier struct {
	metadataTestData *parseServiceBusMetadataTestData
	name             string
}

// not testing connections so it doesn't matter what the resolved env value is for this
var sampleResolvedEnv = map[string]string{
	connectionSetting: "none",
}

var parseServiceBusMetadataDataset = []parseServiceBusMetadataTestData{
	{map[string]string{}, true, none, map[string]string{}, ""},
	// properly formed queue
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting}, false, queue, map[string]string{}, ""},
	// properly formed queue with message count
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "messageCount": messageCount}, false, queue, map[string]string{}, ""},
	// properly formed topic & subscription
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, false, subscription, map[string]string{}, ""},
	// properly formed topic & subscription with message count
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "messageCount": messageCount}, false, subscription, map[string]string{}, ""},
	// queue and topic specified
	{map[string]string{"queueName": queueName, "topicName": topicName, "connectionFromEnv": connectionSetting}, true, none, map[string]string{}, ""},
	// queue and subscription specified
	{map[string]string{"queueName": queueName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, true, none, map[string]string{}, ""},
	// topic but no subscription specified
	{map[string]string{"topicName": topicName, "connectionFromEnv": connectionSetting}, true, none, map[string]string{}, ""},
	// subscription but no topic specified
	{map[string]string{"subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, true, none, map[string]string{}, ""},
	// connection not set
	{map[string]string{"queueName": queueName}, true, queue, map[string]string{}, ""},
	// connection set in auth params
	{map[string]string{"queueName": queueName}, false, queue, map[string]string{"connection": connectionSetting}, ""},
	// pod identity but missing namespace
	{map[string]string{"queueName": queueName}, true, queue, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// correct pod identity
	{map[string]string{"queueName": queueName, "namespace": namespaceName}, false, queue, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
}

var azServiceBusMetricIdentifiers = []azServiceBusMetricIdentifier{
	{&parseServiceBusMetadataDataset[1], "azure-servicebus-testqueue"},
	{&parseServiceBusMetadataDataset[3], "azure-servicebus-testtopic-testsubscription"},
}

var commonHTTPClient = &http.Client{
	Timeout: 300 * time.Millisecond,
}

var getServiceBusLengthTestScalers = []azureServiceBusScaler{
	{
		metadata: &azureServiceBusMetadata{
			entityType: queue,
			queueName:  queueName,
		},
		httpClient: commonHTTPClient,
	},
	{
		metadata: &azureServiceBusMetadata{
			entityType:       subscription,
			topicName:        topicName,
			subscriptionName: subscriptionName,
		},
		httpClient: commonHTTPClient,
	},
	{
		metadata: &azureServiceBusMetadata{
			entityType:       subscription,
			topicName:        topicName,
			subscriptionName: subscriptionName,
		},
		podIdentity: kedav1alpha1.PodIdentityProviderAzure,
		httpClient:  commonHTTPClient,
	},
}

func TestParseServiceBusMetadata(t *testing.T) {
	for _, testData := range parseServiceBusMetadataDataset {
		meta, err := parseAzureServiceBusMetadata(&ScalerConfig{ResolvedEnv: sampleResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams, PodIdentity: testData.podIdentity})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta != nil && meta.entityType != testData.entityType {
			t.Errorf("Expected entity type %v but got %v\n", testData.entityType, meta.entityType)
		}
	}
}

func TestGetServiceBusLength(t *testing.T) {
	t.Log("This test will use the environment variable SERVICEBUS_CONNECTION_STRING if it is set")
	t.Log("If set, it will connect to the servicebus namespace specified by the connection string & check:")
	t.Logf("\tQueue '%s' has 1 message\n", queueName)
	t.Logf("\tTopic '%s' with subscription '%s' has 1 message\n", topicName, subscriptionName)

	connectionString := os.Getenv("SERVICEBUS_CONNECTION_STRING")

	for _, scaler := range getServiceBusLengthTestScalers {
		if connectionString != "" {
			// Can actually test that numbers return
			scaler.metadata.connection = connectionString
			length, err := scaler.GetAzureServiceBusLength(context.TODO())

			if err != nil {
				t.Errorf("Expected success but got error: %s", err)
			}

			if length != 1 {
				t.Errorf("Expected 1 message, got %d", length)
			}
		} else {
			// Just test error message
			length, err := scaler.GetAzureServiceBusLength(context.TODO())

			if length != -1 || err == nil {
				t.Errorf("Expected error but got success")
			}
		}
	}
}

func TestAzServiceBusGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azServiceBusMetricIdentifiers {
		meta, err := parseAzureServiceBusMetadata(&ScalerConfig{ResolvedEnv: sampleResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, PodIdentity: testData.metadataTestData.podIdentity})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzServiceBusScalerScaler := azureServiceBusScaler{
			metadata:    meta,
			podIdentity: testData.metadataTestData.podIdentity,
			httpClient:  http.DefaultClient,
		}

		metricSpec := mockAzServiceBusScalerScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

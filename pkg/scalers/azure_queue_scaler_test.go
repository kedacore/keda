package scalers

import (
	"net/http"
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

var testAzQueueResolvedEnv = map[string]string{
	"CONNECTION": "SAMPLE",
}

type parseAzQueueMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type azQueueMetricIdentifier struct {
	metadataTestData *parseAzQueueMetadataTestData
	name             string
}

var testAzQueueMetadata = []parseAzQueueMetadataTestData{
	// nothing passed
	{map[string]string{}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
	// properly formed
	{map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "5"}, false, testAzQueueResolvedEnv, map[string]string{}, ""},
	// Empty queueName
	{map[string]string{"connectionFromEnv": "CONNECTION", "queueName": ""}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
	// improperly formed queueLength
	{map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "AA"}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
	// Deprecated: useAAdPodIdentity with account name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "sample_acc", "queueName": "sample_queue"}, false, testAzQueueResolvedEnv, map[string]string{}, ""},
	// Deprecated: useAAdPodIdentity without account name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "", "queueName": "sample_queue"}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
	// Deprecated useAAdPodIdentity without queue name
	{map[string]string{"useAAdPodIdentity": "true", "accountName": "sample_acc", "queueName": ""}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
	// podIdentity = azure with account name
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure without account name
	{map[string]string{"accountName": "", "queueName": "sample_queue"}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure without queue name
	{map[string]string{"accountName": "sample_acc", "queueName": ""}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// connection from authParams
	{map[string]string{"queueName": "sample", "queueLength": "5"}, false, testAzQueueResolvedEnv, map[string]string{"connection": "value"}, kedav1alpha1.PodIdentityProviderNone},
}

var azQueueMetricIdentifiers = []azQueueMetricIdentifier{
	{&testAzQueueMetadata[1], "azure-queue-sample"},
	{&testAzQueueMetadata[4], "azure-queue-sample_queue"},
}

func TestAzQueueParseMetadata(t *testing.T) {
	for _, testData := range testAzQueueMetadata {
		_, podIdentity, err := parseAzureQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams, PodIdentity: testData.podIdentity})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success. testData: %v", testData)
		}
		if testData.podIdentity != "" && testData.podIdentity != podIdentity && err == nil {
			t.Error("Expected success but got error: podIdentity value is not returned as expected")
		}
	}
}

func TestAzQueueGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azQueueMetricIdentifiers {
		meta, podIdentity, err := parseAzureQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, PodIdentity: testData.metadataTestData.podIdentity})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzQueueScaler := azureQueueScaler{
			metadata:    meta,
			podIdentity: podIdentity,
			httpClient:  http.DefaultClient,
		}

		metricSpec := mockAzQueueScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

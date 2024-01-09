/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalers

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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
	triggerIndex     int
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
	// improperly formed activationQueueLength
	{map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "1", "activationQueueLength": "AA"}, true, testAzQueueResolvedEnv, map[string]string{}, ""},
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
	// podIdentity = azure with cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "AzurePublicCloud"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure with invalid cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "InvalidCloud"}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure with private cloud and endpoint suffix
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": "queue.core.private.cloud"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure with private cloud and no endpoint suffix
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": ""}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure with endpoint suffix and no cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "", "endpointSuffix": "ignored"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzure},
	// podIdentity = azure-workload with account name
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload without account name
	{map[string]string{"accountName": "", "queueName": "sample_queue"}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload without queue name
	{map[string]string{"accountName": "sample_acc", "queueName": ""}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload with cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "AzurePublicCloud"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload with invalid cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "InvalidCloud"}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload with private cloud and endpoint suffix
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": "queue.core.private.cloud"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload with private cloud and no endpoint suffix
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": ""}, true, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// podIdentity = azure-workload with endpoint suffix and no cloud
	{map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "", "endpointSuffix": "ignored"}, false, testAzQueueResolvedEnv, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// connection from authParams
	{map[string]string{"queueName": "sample", "queueLength": "5"}, false, testAzQueueResolvedEnv, map[string]string{"connection": "value"}, kedav1alpha1.PodIdentityProviderNone},
}

var azQueueMetricIdentifiers = []azQueueMetricIdentifier{
	{&testAzQueueMetadata[1], 0, "s0-azure-queue-sample"},
	{&testAzQueueMetadata[5], 1, "s1-azure-queue-sample_queue"},
}

func TestAzQueueParseMetadata(t *testing.T) {
	for _, testData := range testAzQueueMetadata {
		_, podIdentity, err := parseAzureQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadata,
			ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity}},
			logr.Discard())
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success. testData: %v", testData)
		}
		if testData.podIdentity != "" && testData.podIdentity != podIdentity.Provider && err == nil {
			t.Error("Expected success but got error: podIdentity value is not returned as expected")
		}
	}
}

func TestAzQueueGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azQueueMetricIdentifiers {
		meta, podIdentity, err := parseAzureQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata,
			ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity}, TriggerIndex: testData.triggerIndex},
			logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzQueueScaler := azureQueueScaler{
			metadata:    meta,
			podIdentity: podIdentity,
			httpClient:  http.DefaultClient,
		}

		metricSpec := mockAzQueueScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

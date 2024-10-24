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
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	for i, testData := range testAzQueueMetadata {
		testName := fmt.Sprintf("test case %d", i)
		switch i {
		case 0:
			testName = "nothing passed"
		case 1:
			testName = "properly formed"
		case 2:
			testName = "empty queueName"
		case 3:
			testName = "improperly formed queueLength"
		case 4:
			testName = "improperly formed activationQueueLength"
		case 5:
			testName = "podIdentity azure-workload with account name"
		case 6:
			testName = "podIdentity azure-workload without account name"
		case 7:
			testName = "podIdentity azure-workload without queue name"
		case 8:
			testName = "podIdentity azure-workload with cloud"
		case 9:
			testName = "podIdentity azure-workload with invalid cloud"
		case 10:
			testName = "podIdentity azure-workload with private cloud and endpoint suffix"
		case 11:
			testName = "podIdentity azure-workload with private cloud and no endpoint suffix"
		case 12:
			testName = "podIdentity azure-workload with endpoint suffix and no cloud"
		case 13:
			testName = "connection from authParams"
		}

		t.Run(testName, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testData.resolvedEnv,
				AuthParams:      testData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity},
			}

			_, podIdentity, err := parseAzureQueueMetadata(config)
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Errorf("Expected error but got success. testData: %v", testData)
			}
			if testData.podIdentity != "" && testData.podIdentity != podIdentity.Provider && err == nil {
				t.Error("Expected success but got error: podIdentity value is not returned as expected")
			}
		})
	}
}

func TestAzQueueGetMetricSpecForScaling(t *testing.T) {
	for i, testData := range azQueueMetricIdentifiers {
		testName := fmt.Sprintf("test case %d", i)
		switch i {
		case 0:
			testName = "properly formed queue metric"
		case 1:
			testName = "azure-workload queue metric"
		}

		t.Run(testName, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				ResolvedEnv:     testData.metadataTestData.resolvedEnv,
				AuthParams:      testData.metadataTestData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity},
				TriggerIndex:    testData.triggerIndex,
			}

			meta, _, err := parseAzureQueueMetadata(config)
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			mockAzQueueScaler := azureQueueScaler{
				metadata:   meta,
				logger:     logr.Discard(),
				metricType: v2.AverageValueMetricType,
			}

			metricSpec := mockAzQueueScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			assert.Equal(t, testData.name, metricName)
		})
	}
}

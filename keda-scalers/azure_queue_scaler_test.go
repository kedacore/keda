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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var testAzQueueResolvedEnv = map[string]string{
	"CONNECTION": "SAMPLE",
}

type parseAzQueueMetadataTestData struct {
	name        string
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type azQueueMetricIdentifier struct {
	name             string
	metadataTestData *parseAzQueueMetadataTestData
	triggerIndex     int
	metricName       string
}

var testAzQueueMetadata = []parseAzQueueMetadataTestData{
	{
		name:        "nothing passed",
		metadata:    map[string]string{},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "properly formed",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "5"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "empty queueName",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": ""},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "improperly formed queueLength",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "AA"},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "improperly formed activationQueueLength",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "1", "activationQueueLength": "AA"},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "podIdentity azure-workload with account name",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload without account name",
		metadata:    map[string]string{"accountName": "", "queueName": "sample_queue"},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload without queue name",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": ""},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload with cloud",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "AzurePublicCloud"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload with invalid cloud",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "InvalidCloud"},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload with private cloud and endpoint suffix",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": "queue.core.private.cloud"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload with private cloud and no endpoint suffix",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "Private", "endpointSuffix": ""},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "podIdentity azure-workload with endpoint suffix and no cloud",
		metadata:    map[string]string{"accountName": "sample_acc", "queueName": "sample_queue", "cloud": "", "endpointSuffix": "ignored"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name:        "connection from authParams",
		metadata:    map[string]string{"queueName": "sample", "queueLength": "5"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{"connection": "value"},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name:        "valid queueLengthStrategy all",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "5", "queueLengthStrategy": "all"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "valid queueLengthStrategy visibleonly",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "5", "queueLengthStrategy": "visibleonly"},
		isError:     false,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name:        "invalid queueLengthStrategy",
		metadata:    map[string]string{"connectionFromEnv": "CONNECTION", "queueName": "sample", "queueLength": "5", "queueLengthStrategy": "invalid"},
		isError:     true,
		resolvedEnv: testAzQueueResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
}

var azQueueMetricIdentifiers = []azQueueMetricIdentifier{
	{
		name:             "properly formed queue metric",
		metadataTestData: &testAzQueueMetadata[1],
		triggerIndex:     0,
		metricName:       "s0-azure-queue-sample",
	},
	{
		name:             "azure-workload queue metric",
		metadataTestData: &testAzQueueMetadata[5],
		triggerIndex:     1,
		metricName:       "s1-azure-queue-sample_queue",
	},
}

type mockAzureQueueClient struct {
	peekMessageCount int
	totalMessages    int32
}

func (m *mockAzureQueueClient) getMessageCount(visibleOnly bool) int64 {
	if visibleOnly {
		if m.peekMessageCount >= 32 {
			return int64(m.totalMessages)
		}
		return int64(m.peekMessageCount)
	}
	return int64(m.totalMessages)
}

func TestAzQueueParseMetadata(t *testing.T) {
	for _, testData := range testAzQueueMetadata {
		t.Run(testData.name, func(t *testing.T) {
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
	for _, testData := range azQueueMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
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
			assert.Equal(t, testData.metricName, metricName)
		})
	}
}

func TestAzQueueGetMessageCount(t *testing.T) {
	testCases := []struct {
		name          string
		strategy      string
		peekMessages  int
		totalMessages int32
		expectedCount int64
	}{
		{
			name:          "default strategy (all)",
			strategy:      "",
			peekMessages:  10,
			totalMessages: 100,
			expectedCount: 100,
		},
		{
			name:          "explicit all strategy",
			strategy:      "all",
			peekMessages:  10,
			totalMessages: 100,
			expectedCount: 100,
		},
		{
			name:          "visibleonly strategy with less than 32 messages",
			strategy:      "visibleonly",
			peekMessages:  10,
			totalMessages: 100,
			expectedCount: 10,
		},
		{
			name:          "visibleonly strategy with 32 or more messages",
			strategy:      "visibleonly",
			peekMessages:  35,
			totalMessages: 100,
			expectedCount: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockAzureQueueClient{
				peekMessageCount: tc.peekMessages,
				totalMessages:    tc.totalMessages,
			}

			count := mockClient.getMessageCount(tc.strategy == "visibleonly")
			assert.Equal(t, tc.expectedCount, count)
		})
	}
}

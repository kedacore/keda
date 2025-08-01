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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	topicName         = "testtopic"
	subscriptionName  = "testsubscription"
	queueName         = "testqueue"
	connectionSetting = "none"
	namespaceName     = "ns"
	messageCount      = "1000"
	defaultSuffix     = "ns.servicebus.windows.net"
)

type parseServiceBusMetadataTestData struct {
	metadata                map[string]string
	isError                 bool
	entityType              entityType
	fullyQualifiedNamespace string
	authParams              map[string]string
	podIdentity             kedav1alpha1.PodIdentityProvider
}

type azServiceBusMetricIdentifier struct {
	metadataTestData *parseServiceBusMetadataTestData
	triggerIndex     int
	name             string
}

// not testing connections so it doesn't matter what the resolved env value is for this
var sampleResolvedEnv = map[string]string{
	connectionSetting: "none",
}

// namespace example for setting up metric name
var connectionResolvedEnv = map[string]string{
	connectionSetting: "Endpoint=sb://namespacename.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=c29tZXJlYWxseWltcG9ydGFudGtleQ==",
}

var parseServiceBusMetadataDataset = []parseServiceBusMetadataTestData{
	{map[string]string{}, true, none, "", map[string]string{}, ""},
	// properly formed queue
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting}, false, queue, defaultSuffix, map[string]string{}, ""},
	// properly formed queue with message count
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "messageCount": messageCount}, false, queue, defaultSuffix, map[string]string{}, ""},
	// properly formed topic & subscription
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, false, subscription, defaultSuffix, map[string]string{}, ""},
	// properly formed topic & subscription with message count
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "messageCount": messageCount}, false, subscription, defaultSuffix, map[string]string{}, ""},
	// queue and topic specified
	{map[string]string{"queueName": queueName, "topicName": topicName, "connectionFromEnv": connectionSetting}, true, none, "", map[string]string{}, ""},
	// queue and subscription specified
	{map[string]string{"queueName": queueName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, true, none, "", map[string]string{}, ""},
	// topic but no subscription specified
	{map[string]string{"topicName": topicName, "connectionFromEnv": connectionSetting}, true, none, "", map[string]string{}, ""},
	// subscription but no topic specified
	{map[string]string{"subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting}, true, none, "", map[string]string{}, ""},
	// valid cloud
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "cloud": "AzureChinaCloud"}, false, queue, "servicebus.chinacloudapi.cn", map[string]string{}, ""},
	// invalid cloud
	{map[string]string{"queueName": queueName, "cloud": "InvalidCloud"}, true, none, "", map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// private cloud with endpoint suffix
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "cloud": "Private", "endpointSuffix": "servicebus.private.cloud"}, false, queue, "servicebus.private.cloud", map[string]string{}, ""},
	// private cloud without endpoint suffix
	{map[string]string{"queueName": queueName, "cloud": "Private"}, true, none, "", map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// endpoint suffix without cloud
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "endpointSuffix": "ignored"}, false, queue, defaultSuffix, map[string]string{}, ""},
	// connection not set
	{map[string]string{"queueName": queueName}, true, queue, "", map[string]string{}, ""},
	// connection set in auth params
	{map[string]string{"queueName": queueName}, false, queue, defaultSuffix, map[string]string{"connection": connectionSetting}, ""},
	// workload identity but missing namespace
	{map[string]string{"queueName": queueName}, true, queue, "", map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// correct workload identity
	{map[string]string{"queueName": queueName, "namespace": namespaceName}, false, queue, defaultSuffix, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// invalid activation message count
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "messageCount": messageCount, "activationMessageCount": "AA"}, true, queue, defaultSuffix, map[string]string{}, ""},
	// queue with incorrect useRegex value
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "ababa"}, true, queue, defaultSuffix, map[string]string{}, ""},
	// properly formed queues with regex
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "false"}, false, queue, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": avgOperation}, false, queue, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": sumOperation}, false, queue, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": maxOperation}, false, queue, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "true"}, false, queue, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"queueName": queueName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": "random"}, true, queue, defaultSuffix, map[string]string{}, ""},
	// queue with invalid regex string
	{map[string]string{"queueName": "*", "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": "avg"}, true, queue, defaultSuffix, map[string]string{}, ""},

	// subscription with incorrect useRegex value
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "ababa"}, true, subscription, defaultSuffix, map[string]string{}, ""},
	// properly formed subscriptions with regex
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "false"}, false, subscription, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": avgOperation}, false, subscription, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": sumOperation}, false, subscription, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": maxOperation}, false, subscription, defaultSuffix, map[string]string{}, ""},
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": "random"}, true, subscription, defaultSuffix, map[string]string{}, ""},
	// subscription with invalid regex string
	{map[string]string{"topicName": topicName, "subscriptionName": "*", "connectionFromEnv": connectionSetting, "useRegex": "true", "operation": "avg"}, true, subscription, defaultSuffix, map[string]string{}, ""},
}

var azServiceBusMetricIdentifiers = []azServiceBusMetricIdentifier{
	{&parseServiceBusMetadataDataset[1], 0, "s0-azure-servicebus-testqueue"},
	{&parseServiceBusMetadataDataset[3], 1, "s1-azure-servicebus-testtopic"},
}

var getServiceBusLengthTestScalers = []azureServiceBusScaler{
	{
		metadata: &azureServiceBusMetadata{
			EntityType: queue,
			QueueName:  queueName,
		},
	},
	{
		metadata: &azureServiceBusMetadata{
			EntityType:       subscription,
			TopicName:        topicName,
			SubscriptionName: subscriptionName,
		},
	},
	{
		metadata: &azureServiceBusMetadata{
			EntityType:       subscription,
			TopicName:        topicName,
			SubscriptionName: subscriptionName,
		},
		podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	},
}

func TestParseServiceBusMetadata(t *testing.T) {
	for index, testData := range parseServiceBusMetadataDataset {
		meta, err := parseAzureServiceBusMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: testData.authParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity}, TriggerIndex: 0})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		fmt.Print(index)
		if meta != nil {
			if meta.EntityType != testData.entityType {
				t.Errorf("Expected entity type %v but got %v\n", testData.entityType, meta.EntityType)
			}
			if testData.podIdentity != "" && meta.FullyQualifiedNamespace != testData.fullyQualifiedNamespace {
				t.Errorf("Expected endpoint suffix %v but got %v\n", testData.fullyQualifiedNamespace, meta.FullyQualifiedNamespace)
			}
		}
	}
}

func TestGetServiceBusAdminClientIsCached(t *testing.T) {
	testData := azServiceBusMetricIdentifiers[0]
	meta, err := parseAzureServiceBusMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: connectionResolvedEnv,
		TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams,
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity}, TriggerIndex: testData.triggerIndex})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}
	mockAzServiceBusScalerScaler := azureServiceBusScaler{
		metadata:    meta,
		podIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity},
	}

	_, _ = mockAzServiceBusScalerScaler.getServiceBusAdminClient()
	assert.NotNil(t, mockAzServiceBusScalerScaler.client)
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
			scaler.metadata.Connection = connectionString
			length, err := scaler.getAzureServiceBusLength(context.TODO())

			if err != nil {
				t.Errorf("Expected success but got error: %s", err)
			}

			if length != 1 {
				t.Errorf("Expected 1 message, got %d", length)
			}
		} else {
			// Just test error message
			length, err := scaler.getAzureServiceBusLength(context.TODO())

			if length != -1 || err == nil {
				t.Errorf("Expected error but got success")
			}
		}
	}
}

func TestAzServiceBusGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azServiceBusMetricIdentifiers {
		meta, err := parseAzureServiceBusMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: connectionResolvedEnv,
			TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity}, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzServiceBusScalerScaler := azureServiceBusScaler{
			metadata:    meta,
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity},
		}

		metricSpec := mockAzServiceBusScalerScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

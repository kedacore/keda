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

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testAzureResourceManagerEndpoint = "testAzureResourceManagerEndpoint"
)

type parseAzMonitorMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type azMonitorMetricIdentifier struct {
	metadataTestData *parseAzMonitorMetadataTestData
	triggerIndex     int
	name             string
}

var testAzMonitorResolvedEnv = map[string]string{
	"CLIENT_ID":       "xxx",
	"CLIENT_PASSWORD": "yyy",
}

var testParseAzMonitorMetadata = []parseAzMonitorMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}, map[string]string{}, ""},
	// properly formed
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5", "metricNamespace": "namespace"}, false, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// no optional parameters
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5"}, false, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// incorrectly formatted resourceURI
	{map[string]string{"resourceURI": "bad/format", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// improperly formatted aggregationInterval
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:1", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing resourceURI
	{map[string]string{"tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing tenantId
	{map[string]string{"resourceURI": "test/resource/uri", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing subscriptionId
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing resourceGroupName
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing metricName
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing metricAggregationType
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// filter included
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricFilter": "namespace eq 'default'", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5"}, false, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing activeDirectoryClientId
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing activeDirectoryClientPassword
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// missing targetValue
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// invalid activationTargetValue
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5", "activationTargetValue": "A"}, true, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// connection from authParams
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "targetValue": "5"}, false, map[string]string{}, map[string]string{"activeDirectoryClientId": "zzz", "activeDirectoryClientPassword": "password"}, ""},
	// wrong podIdentity
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "targetValue": "5"}, true, map[string]string{}, map[string]string{}, kedav1alpha1.PodIdentityProvider("notAzure")},
	// connection with workload Identity
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "targetValue": "5"}, false, map[string]string{}, map[string]string{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
	// wrong workload Identity
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "targetValue": "5"}, true, map[string]string{}, map[string]string{}, kedav1alpha1.PodIdentityProvider("notAzureWorkload")},
	// known azure cloud
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5", "metricNamespace": "namespace", "cloud": "azureChinaCloud"}, false, testAzMonitorResolvedEnv, map[string]string{}, ""},
	// private cloud
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPasswordFromEnv": "CLIENT_PASSWORD", "targetValue": "5", "metricNamespace": "namespace", "cloud": "private",
		"azureResourceManagerEndpoint": testAzureResourceManagerEndpoint}, false, testAzMonitorResolvedEnv, map[string]string{}, ""},
}

var azMonitorMetricIdentifiers = []azMonitorMetricIdentifier{
	{&testParseAzMonitorMetadata[1], 0, "s0-azure-monitor-metric"},
	{&testParseAzMonitorMetadata[1], 1, "s1-azure-monitor-metric"},
}

func TestAzMonitorParseMetadata(t *testing.T) {
	for _, testData := range testParseAzMonitorMetadata {
		_, err := parseAzureMonitorMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv,
			AuthParams: testData.authParams, PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success. testData: %v", testData)
		}
	}
}

func TestAzMonitorGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azMonitorMetricIdentifiers {
		meta, err := parseAzureMonitorMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata,
			ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity}, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzMonitorScaler := azureMonitorScaler{"", meta, logr.Discard(), nil}

		metricSpec := mockAzMonitorScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

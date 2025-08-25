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

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type parseDataExplorerMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type dataExplorerMetricIdentifier struct {
	metadataTestData *parseDataExplorerMetadataTestData
	triggerIndex     int
	name             string
}

var (
	aadAppClientID          = "eebdbbab-cf74-4791-a5c6-1ef5d90b1fa8"
	aadAppSecret            = "test_app_secret"
	activeDirectoryEndpoint = "activeDirectoryEndpoint"
	azureTenantID           = "8fe57c22-02b1-4b87-8c24-ae21dea4fa6a"
	databaseName            = "test_database"
	dataExplorerQuery       = "print 3"
	dataExplorerThreshold   = "1"
	dataExplorerEndpoint    = "https://test-keda-e2e.eastus.kusto.windows.net"
)

// Valid auth params with aad application and passwd
var dataExplorerResolvedEnv = map[string]string{
	"tenantId":     azureTenantID,
	"clientId":     aadAppClientID,
	"clientSecret": aadAppSecret,
}

var testDataExplorerMetadataWithClientAndSecret = []parseDataExplorerMetadataTestData{
	// Empty metadata - fail
	{map[string]string{}, true},
	// Missing tenantId - fail
	{map[string]string{"tenantId": "", "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing clientId - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": "", "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing clientSecret - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": "", "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing endpoint - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": "", "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing databaseName - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": "", "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing query - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": "", "threshold": dataExplorerThreshold}, true},
	// Missing threshold - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": ""}, true},
	// Invalid activationThreshold - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": "1", "activationThreshold": "A"}, true},
	// known cloud
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold,
		"cloud": "azureChinaCloud"}, false},
	// private cloud
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold,
		"cloud": "private", "activeDirectoryEndpoint": activeDirectoryEndpoint}, false},
	// private cloud - missing active directory endpoint - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold,
		"cloud": "private"}, true},
	// All parameters set - pass
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, false},
	// False because we should not get clientSecret from TriggerMetadata
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecret": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
}

var testDataExplorerMetadataWithPodIdentity = []parseDataExplorerMetadataTestData{
	// Empty metadata - fail
	{map[string]string{}, true},
	// Missing endpoint - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": "", "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, true},
	// Missing query - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": "", "threshold": dataExplorerThreshold}, true},
	// Missing threshold - fail
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": ""}, true},
	// All parameters set - pass
	{map[string]string{"tenantId": azureTenantID, "clientId": aadAppClientID, "clientSecretFromEnv": aadAppSecret, "endpoint": dataExplorerEndpoint, "databaseName": databaseName, "query": dataExplorerQuery, "threshold": dataExplorerThreshold}, false},
}

var testDataExplorerMetricIdentifiers = []dataExplorerMetricIdentifier{
	{&testDataExplorerMetadataWithClientAndSecret[len(testDataExplorerMetadataWithClientAndSecret)-2], 0, GenerateMetricNameWithIndex(0, kedautil.NormalizeString(fmt.Sprintf("%s-%s", adxName, databaseName)))},
	{&testDataExplorerMetadataWithPodIdentity[len(testDataExplorerMetadataWithPodIdentity)-1], 1, GenerateMetricNameWithIndex(1, kedautil.NormalizeString(fmt.Sprintf("%s-%s", adxName, databaseName)))},
}

func TestDataExplorerParseMetadata(t *testing.T) {
	// Auth through clientId, clientSecret and tenantId
	for id, testData := range testDataExplorerMetadataWithClientAndSecret {
		_, err := parseAzureDataExplorerMetadata(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:     dataExplorerResolvedEnv,
				TriggerMetadata: testData.metadata,
				AuthParams:      map[string]string{},
				PodIdentity:     kedav1alpha1.AuthPodIdentity{}},
			logr.Discard())

		if err != nil && !testData.isError {
			t.Errorf("Test case %d: expected success but got error %v", id, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test case %d: expected error but got success", id)
		}
	}

	// Auth through Pod Identity
	for _, testData := range testDataExplorerMetadataWithPodIdentity {
		_, err := parseAzureDataExplorerMetadata(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:     dataExplorerResolvedEnv,
				TriggerMetadata: testData.metadata,
				AuthParams:      map[string]string{},
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// Auth through Workload Identity
	for _, testData := range testDataExplorerMetadataWithPodIdentity {
		_, err := parseAzureDataExplorerMetadata(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:     dataExplorerResolvedEnv,
				TriggerMetadata: testData.metadata,
				AuthParams:      map[string]string{},
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestDataExplorerGetMetricSpecForScaling(t *testing.T) {
	for id, testData := range testDataExplorerMetricIdentifiers {
		meta, err := parseAzureDataExplorerMetadata(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:     dataExplorerResolvedEnv,
				TriggerMetadata: testData.metadataTestData.metadata,
				AuthParams:      map[string]string{},
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerIndex:    testData.triggerIndex},
			logr.Discard())
		if err != nil {
			t.Errorf("Test case %d: failed to parse metadata: %v", id, err)
		}

		mockDataExplorerScaler := azureDataExplorerScaler{
			metadata:  meta,
			client:    nil,
			name:      "mock_scaled_object",
			namespace: "mock_namespace",
			logger:    logr.Discard(),
		}

		metricSpec := mockDataExplorerScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Test case %d: wrong External metric source name: %v", id, metricName)
		}
	}
}

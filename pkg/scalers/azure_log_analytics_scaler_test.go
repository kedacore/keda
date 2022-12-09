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

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	tenantID                    = "d248da64-0e1e-4f79-b8c6-72ab7aa055eb"
	clientID                    = "41826dd4-9e0a-4357-a5bd-a88ad771ea7d"
	clientSecret                = "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs"
	workspaceID                 = "074dd9f8-c368-4220-9400-acb6e80fc325"
	testLogAnalyticsResourceURL = "testLogAnalyticsResourceURL"
	testActiveDirectoryEndpoint = "testActiveDirectoryEndpoint"
)

type parseLogAnalyticsMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type LogAnalyticsMetricIdentifier struct {
	metadataTestData *parseLogAnalyticsMetadataTestData
	scalerIndex      int
	name             string
}

var (
	query = "let x = 10; let y = 1; print MetricValue = x, Threshold = y;"
)

// Faked parameters
var sampleLogAnalyticsResolvedEnv = map[string]string{
	tenantID:     "d248da64-0e1e-4f79-b8c6-72ab7aa055eb",
	clientID:     "41826dd4-9e0a-4357-a5bd-a88ad771ea7d",
	clientSecret: "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs",
	workspaceID:  "074dd9f8-c368-4220-9400-acb6e80fc325",
}

// A complete valid authParams with username and passwd (Faked)
var LogAnalyticsAuthParams = map[string]string{
	"tenantId":     "d248da64-0e1e-4f79-b8c6-72ab7aa055eb",
	"clientId":     "41826dd4-9e0a-4357-a5bd-a88ad771ea7d",
	"clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs",
	"workspaceId":  "074dd9f8-c368-4220-9400-acb6e80fc325",
}

// An invalid authParams without username and passwd
var emptyLogAnalyticsAuthParams = map[string]string{
	"tenantId":     "",
	"clientId":     "",
	"clientSecret": "",
	"workspaceId":  "",
}

var testLogAnalyticsMetadata = []parseLogAnalyticsMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing tenantId should fail
	{map[string]string{"tenantId": "", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing clientId, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing clientSecret, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing workspaceId, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "", "query": query, "threshold": "1900000000"}, true},
	// Missing query, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": "", "threshold": "1900000000"}, true},
	// Missing threshold, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": ""}, true},
	// Invalid activation threshold, should fail
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1", "activationThreshold": "A"}, true},
	// All parameters set, should succeed
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, false},
	// Known Azure Cloud
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000", "cloud": "azurePublicCloud"}, false},
	// Private Cloud
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000", "cloud": "private", "logAnalyticsResourceURL": testLogAnalyticsResourceURL, "activeDirectoryEndpoint": testActiveDirectoryEndpoint}, false},
	// Private Cloud missing log analytics resource url
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000", "cloud": "private", "activeDirectoryEndpoint": testActiveDirectoryEndpoint}, true},
	// Private Cloud missing active directory endpoint
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000", "cloud": "private", "logAnalyticsResourceURL": testLogAnalyticsResourceURL}, true},
	// Unsupported cloud
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000", "cloud": "azureGermanCloud"}, true},
}

var LogAnalyticsMetricIdentifiers = []LogAnalyticsMetricIdentifier{
	{&testLogAnalyticsMetadata[8], 0, "s0-azure-log-analytics-074dd9f8-c368-4220-9400-acb6e80fc325"},
	{&testLogAnalyticsMetadata[8], 1, "s1-azure-log-analytics-074dd9f8-c368-4220-9400-acb6e80fc325"},
}

var testLogAnalyticsMetadataWithEmptyAuthParams = []parseLogAnalyticsMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing query, should fail
	{map[string]string{"query": "", "threshold": "1900000000"}, true},
	// Missing threshold, should fail
	{map[string]string{"query": query, "threshold": ""}, true},
	// All parameters set, should succeed
	{map[string]string{"query": query, "threshold": "1900000000"}, true},
}

var testLogAnalyticsMetadataWithAuthParams = []parseLogAnalyticsMetadataTestData{
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, false},
}

var testLogAnalyticsMetadataWithPodIdentity = []parseLogAnalyticsMetadataTestData{
	{map[string]string{"workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, false},
}

func TestLogAnalyticsParseMetadata(t *testing.T) {
	for _, testData := range testLogAnalyticsMetadata {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: nil, PodIdentity: kedav1alpha1.AuthPodIdentity{}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testLogAnalyticsMetadataWithEmptyAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: emptyLogAnalyticsAuthParams, PodIdentity: kedav1alpha1.AuthPodIdentity{}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testLogAnalyticsMetadataWithAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: LogAnalyticsAuthParams, PodIdentity: kedav1alpha1.AuthPodIdentity{}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with podIdentity params should not fail
	for _, testData := range testLogAnalyticsMetadataWithPodIdentity {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: LogAnalyticsAuthParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzure}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with workload identity params should not fail
	for _, testData := range testLogAnalyticsMetadataWithPodIdentity {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: LogAnalyticsAuthParams,
			PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestLogAnalyticsGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range LogAnalyticsMetricIdentifiers {
		meta, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil,
			PodIdentity: kedav1alpha1.AuthPodIdentity{}, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockLogAnalyticsScaler := azureLogAnalyticsScaler{
			metadata:   meta,
			name:       "test-so",
			namespace:  "test-ns",
			httpClient: http.DefaultClient,
		}

		metricSpec := mockLogAnalyticsScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

type parseMetadataMetricNameTestData struct {
	metadata    map[string]string
	scalerIndex int
	metricName  string
}

var testParseMetadataMetricName = []parseMetadataMetricNameTestData{
	// WorkspaceID
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, 0, "azure-log-analytics-074dd9f8-c368-4220-9400-acb6e80fc325"},
	// Custom Name
	{map[string]string{"metricName": "testName", "tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, 1, "azure-log-analytics-testName"},
}

func TestLogAnalyticsParseMetadataMetricName(t *testing.T) {
	for _, testData := range testParseMetadataMetricName {
		meta, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv,
			TriggerMetadata: testData.metadata, AuthParams: nil,
			PodIdentity: kedav1alpha1.AuthPodIdentity{}, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Error("Expected success but got error", err)
		}
		if meta.metricName != testData.metricName {
			t.Errorf("Expected %s but got %s", testData.metricName, meta.metricName)
		}
	}
}

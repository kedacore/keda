package scalers

import (
	"net/http"
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

const (
	tenantID     = "d248da64-0e1e-4f79-b8c6-72ab7aa055eb"
	clientID     = "41826dd4-9e0a-4357-a5bd-a88ad771ea7d"
	clientSecret = "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs"
	workspaceID  = "074dd9f8-c368-4220-9400-acb6e80fc325"
)

type parseLogAnalyticsMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type LogAnalyticsMetricIdentifier struct {
	metadataTestData *parseLogAnalyticsMetadataTestData
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
	// All parameters set, should succeed
	{map[string]string{"tenantId": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientId": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceId": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, false},
	// All parameters set, should succeed
	{map[string]string{"tenantIdFromEnv": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientIdFromEnv": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecretFromEnv": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceIdFromEnv": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, false},
}

var LogAnalyticsMetricIdentifiers = []LogAnalyticsMetricIdentifier{
	{&testLogAnalyticsMetadata[7], "azure-log-analytics-074dd9f8-c368-4220-9400-acb6e80fc325"},
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
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: nil, PodIdentity: ""})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testLogAnalyticsMetadataWithEmptyAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: emptyLogAnalyticsAuthParams, PodIdentity: ""})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testLogAnalyticsMetadataWithAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: LogAnalyticsAuthParams, PodIdentity: ""})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with podIdentity params should not fail
	for _, testData := range testLogAnalyticsMetadataWithPodIdentity {
		_, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: LogAnalyticsAuthParams, PodIdentity: kedav1alpha1.PodIdentityProviderAzure})
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
		meta, err := parseAzureLogAnalyticsMetadata(&ScalerConfig{ResolvedEnv: sampleLogAnalyticsResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, PodIdentity: ""})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		cache := &sessionCache{metricValue: 1, metricThreshold: 2}
		mockLogAnalyticsScaler := azureLogAnalyticsScaler{
			metadata:   meta,
			cache:      cache,
			name:       "test-so",
			namespace:  "test-ns",
			httpClient: http.DefaultClient,
		}

		metricSpec := mockLogAnalyticsScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

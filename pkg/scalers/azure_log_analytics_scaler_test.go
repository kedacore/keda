package scalers

import (
	"testing"
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
	query = "Perf | where InstanceName contains \"web\" | where CounterName == \"cpuUsageNanoCores\" | summarize arg_max(TimeGenerated, *) by InstanceName, CounterName | project CounterValue | limit 1"
)

//Faked parameters
var sampleLogAnalyticsResolvedEnv = map[string]string{
	tenantID:     "d248da64-0e1e-4f79-b8c6-72ab7aa055eb",
	clientID:     "41826dd4-9e0a-4357-a5bd-a88ad771ea7d",
	clientSecret: "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs",
	workspaceID:  "074dd9f8-c368-4220-9400-acb6e80fc325",
}

// A complete valid authParams with username and passwd (Faked)
var LogAnalyticsAuthParams = map[string]string{
	"tenantID":     "d248da64-0e1e-4f79-b8c6-72ab7aa055eb",
	"clientID":     "41826dd4-9e0a-4357-a5bd-a88ad771ea7d",
	"clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs",
	"workspaceID":  "074dd9f8-c368-4220-9400-acb6e80fc325",
}

// An invalid authParams without username and passwd
var emptyLogAnalyticsAuthParams = map[string]string{
	"tenantID":     "",
	"clientID":     "",
	"clientSecret": "",
	"workspaceID":  "",
}

var testLogAnalyticsMetadata = []parseLogAnalyticsMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing tenantID should fail
	{map[string]string{"tenantID": "", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing clientID, should fail
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing clientSecret, should fail
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
	// Missing workspaceID, should fail
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "", "query": query, "threshold": "1900000000"}, true},
	// Missing query, should fail
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": "", "threshold": "1900000000"}, true},
	// Missing threshold, should fail
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": ""}, true},
	//All parameters set, should succeed
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
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
	//All parameters set, should succeed
	{map[string]string{"query": query, "threshold": "1900000000"}, true},
}

var testLogAnalyticsMetadataWithAuthParams = []parseLogAnalyticsMetadataTestData{
	{map[string]string{"tenantID": "d248da64-0e1e-4f79-b8c6-72ab7aa055eb", "clientID": "41826dd4-9e0a-4357-a5bd-a88ad771ea7d", "clientSecret": "U6DtAX5r6RPZxd~l12Ri3X8J9urt5Q-xs", "workspaceID": "074dd9f8-c368-4220-9400-acb6e80fc325", "query": query, "threshold": "1900000000"}, true},
}

func TestLogAnalyticsParseMetadata(t *testing.T) {
	for _, testData := range testLogAnalyticsMetadata {
		_, err := parseAzureLogAnalyticsMetadata(sampleLogAnalyticsResolvedEnv, testData.metadata, nil)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testLogAnalyticsMetadataWithEmptyAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(sampleLogAnalyticsResolvedEnv, testData.metadata, emptyLogAnalyticsAuthParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testLogAnalyticsMetadataWithAuthParams {
		_, err := parseAzureLogAnalyticsMetadata(sampleLogAnalyticsResolvedEnv, testData.metadata, LogAnalyticsAuthParams)
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
		meta, err := parseAzureLogAnalyticsMetadata(sampleLogAnalyticsResolvedEnv, testData.metadataTestData.metadata, nil)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		cache := &sessionCache{metricValue: 1, metricThreshold: 2}
		mockLogAnalyticsScaler := azureLogAnalyticsScaler{meta, cache}

		metricSpec := mockLogAnalyticsScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

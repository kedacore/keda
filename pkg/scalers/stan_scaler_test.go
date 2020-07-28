package scalers

import (
	"testing"
)

type parseStanMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type stanMetricIdentifier struct {
	metadataTestData *parseStanMetadataTestData
	name             string
}

var testStanMetadata = []parseStanMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing subject name, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable"}, true},
	// Missing durable name, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "subject": "mySubject"}, true},
	// Missing nats server monitoring endpoint, should fail
	{map[string]string{"queueGroup": "grp1", "subject": "mySubject"}, true},
	// All good.
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject"}, false},
}

var stanMetricIdentifiers = []stanMetricIdentifier{
	{&testStanMetadata[4], "stan-grp1-ImDurable-mySubject"},
}

func TestStanParseMetadata(t *testing.T) {
	for _, testData := range testStanMetadata {
		_, err := parseStanMetadata(testData.metadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestStanGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range stanMetricIdentifiers {
		meta, err := parseStanMetadata(testData.metadataTestData.metadata)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockStanScaler := stanScaler{nil, meta}

		metricSpec := mockStanScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

package scalers

import (
	"testing"

	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseApacheIggyMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var validApacheIggyMetadata = map[string]string{
	"serverAddress":   "localhost:8090",
	"streamId":        "test-stream",
	"topicId":         "test-topic",
	"consumerGroupId": "test-group",
}

var parseApacheIggyMetadataTestDataset = []parseApacheIggyMetadataTestData{
	// success - username/password auth
	{validApacheIggyMetadata, map[string]string{"username": "admin", "password": "admin"}, false},
	// success - PAT auth
	{validApacheIggyMetadata, map[string]string{"accessToken": "my-token"}, false},
	// success - custom lagThreshold
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "100",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - custom activationLagThreshold
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "activationLagThreshold": "5",
	}, map[string]string{"accessToken": "tok"}, false},
	// failure - missing serverAddress
	{map[string]string{"streamId": "s", "topicId": "t", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing streamId
	{map[string]string{"serverAddress": "localhost:8090", "topicId": "t", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing topicId
	{map[string]string{"serverAddress": "localhost:8090", "streamId": "s", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing consumerGroupId
	{map[string]string{"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - no auth provided
	{validApacheIggyMetadata, map[string]string{}, true},
	// failure - both auth methods provided
	{validApacheIggyMetadata, map[string]string{
		"username": "admin", "password": "admin", "accessToken": "tok",
	}, true},
	// failure - username without password
	{validApacheIggyMetadata, map[string]string{"username": "admin"}, true},
	// failure - password without username
	{validApacheIggyMetadata, map[string]string{"password": "admin"}, true},
	// failure - lagThreshold is 0
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "0",
	}, map[string]string{"accessToken": "tok"}, true},
	// failure - lagThreshold is negative
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "-1",
	}, map[string]string{"accessToken": "tok"}, true},
	// failure - activationLagThreshold is negative
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "activationLagThreshold": "-1",
	}, map[string]string{"accessToken": "tok"}, true},
}

func TestApacheIggyParseMetadata(t *testing.T) {
	for idx, testData := range parseApacheIggyMetadataTestDataset {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
		}
		_, err := parseApacheIggyMetadata(config)
		if err != nil && !testData.isError {
			t.Errorf("test index %d: expected success but got error: %v", idx, err)
		}
		if err == nil && testData.isError {
			t.Errorf("test index %d: expected error but got success", idx)
		}
	}
}

func TestApacheIggyDefaultValues(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validApacheIggyMetadata,
		AuthParams:      map[string]string{"accessToken": "tok"},
	}
	meta, err := parseApacheIggyMetadata(config)
	if err != nil {
		t.Fatalf("expected success but got error: %v", err)
	}
	if meta.LagThreshold != 10 {
		t.Errorf("expected default lagThreshold 10, got %d", meta.LagThreshold)
	}
	if meta.ActivationLagThreshold != 0 {
		t.Errorf("expected default activationLagThreshold 0, got %d", meta.ActivationLagThreshold)
	}
}

type apacheIggyMetricIdentifier struct {
	metadataTestData *parseApacheIggyMetadataTestData
	triggerIndex     int
	name             string
}

var apacheIggyMetricIdentifiers = []apacheIggyMetricIdentifier{
	{&parseApacheIggyMetadataTestDataset[0], 0, "s0-iggy-test-stream-test-topic"},
	{&parseApacheIggyMetadataTestDataset[0], 1, "s1-iggy-test-stream-test-topic"},
	{&parseApacheIggyMetadataTestDataset[1], 0, "s0-iggy-test-stream-test-topic"},
}

func TestApacheIggyGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range apacheIggyMetricIdentifiers {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		}
		meta, err := parseApacheIggyMetadata(config)
		if err != nil {
			t.Fatal("could not parse metadata:", err)
		}

		mockScaler := apacheIggyScaler{
			metadata:   meta,
			metricType: v2.AverageValueMetricType,
		}

		metricSpec := mockScaler.GetMetricSpecForScaling(t.Context())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("expected %s, got %s", testData.name, metricName)
		}
	}
}

type apacheIggyLagTestData struct {
	description    string
	partitionLags  []int64
	lagThreshold   int64
	activationLag  int64
	expectedMetric int64
	expectedActive bool
}

var apacheIggyLagTestDataset = []apacheIggyLagTestData{
	{
		description:    "no lag, inactive",
		partitionLags:  []int64{0, 0, 0},
		lagThreshold:   10,
		activationLag:  0,
		expectedMetric: 0,
		expectedActive: false,
	},
	{
		description:    "some lag, active",
		partitionLags:  []int64{5, 3, 2},
		lagThreshold:   10,
		activationLag:  0,
		expectedMetric: 10,
		expectedActive: true,
	},
	{
		description:    "lag below activation threshold, inactive",
		partitionLags:  []int64{2, 1, 0},
		lagThreshold:   10,
		activationLag:  5,
		expectedMetric: 3,
		expectedActive: false,
	},
	{
		description:    "lag exceeds partition cap",
		partitionLags:  []int64{50, 50, 50},
		lagThreshold:   10,
		activationLag:  0,
		expectedMetric: 30,
		expectedActive: true,
	},
	{
		description:    "single partition with lag",
		partitionLags:  []int64{7},
		lagThreshold:   10,
		activationLag:  0,
		expectedMetric: 7,
		expectedActive: true,
	},
	{
		description:    "zero partitions",
		partitionLags:  []int64{},
		lagThreshold:   10,
		activationLag:  0,
		expectedMetric: 0,
		expectedActive: false,
	},
}

func TestApacheIggyCalculateLag(t *testing.T) {
	for _, testData := range apacheIggyLagTestDataset {
		t.Run(testData.description, func(t *testing.T) {
			metric, active := calculateIggyLag(
				testData.partitionLags,
				testData.lagThreshold,
				testData.activationLag,
			)
			if metric != testData.expectedMetric {
				t.Errorf("expected metric %d, got %d", testData.expectedMetric, metric)
			}
			if active != testData.expectedActive {
				t.Errorf("expected active %v, got %v", testData.expectedActive, active)
			}
		})
	}
}

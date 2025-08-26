package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testPubSubResolvedEnv = map[string]string{
	"SAMPLE_CREDS":        "{}",
	"MY_ENV_SUBSCRIPTION": "myEnvSubscription",
	"MY_ENV_TOPIC":        "myEnvTopic",
}

type parsePubSubMetadataTestData struct {
	testName   string
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpPubSubMetricIdentifier struct {
	metadataTestData *parsePubSubMetadataTestData
	triggerIndex     int
	name             string
}

type gcpPubSubSubscription struct {
	metadataTestData *parsePubSubMetadataTestData
	triggerIndex     int
	name             string
	projectID        string
}

var testPubSubMetadata = []parsePubSubMetadataTestData{
	{"empty", map[string]string{}, map[string]string{}, true},
	// all properly formed with deprecated field
	{"all properly formed with deprecated field", nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// all properly formed with subscriptionName
	{"all properly formed with subscriptionName", nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "5"}, false},
	// all properly formed with oldest unacked message age mode
	{"all properly formed with oldest unacked message age mode", nil, map[string]string{"subscriptionName": "mysubscription", "mode": "OldestUnackedMessageAge", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing subscriptionName
	{"missing subscriptionName", nil, map[string]string{"subscriptionName": "", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{"missing credentials", nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": ""}, true},
	// malformed subscriptionSize
	{"malformed subscriptionSize", nil, map[string]string{"subscriptionName": "mysubscription", "value": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed mode
	{"malformed mode", nil, map[string]string{"subscriptionName": "", "mode": "AA", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetValue
	{"malformed activationTargetValue", nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "AA"}, true},
	// Credentials from AuthParams
	{"Credentials from AuthParams", map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"subscriptionName": "mysubscription", "value": "7"}, false},
	// Credentials from AuthParams with empty creds
	{"Credentials from AuthParams with empty creds", map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7"}, true},
	// with full link to subscription
	{"with full link to subscription", nil, map[string]string{"subscriptionName": "projects/myproject/subscriptions/mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// with full (bad) link to subscription
	{"with full (bad) link to subscription", nil, map[string]string{"subscriptionName": "projects/myproject/mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// properly formed float value and activationTargetValue
	{"properly formed float value and activationTargetValue", nil, map[string]string{"subscriptionName": "mysubscription", "value": "7.1", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "2.1"}, false},
	// All optional omitted
	{"All optional omitted", nil, map[string]string{"subscriptionName": "mysubscription", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// value omitted when mode present
	{"value omitted when mode present", nil, map[string]string{"subscriptionName": "mysubscription", "mode": "SubscriptionSize", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// all properly formed with topicName
	{"all properly formed with topicName", nil, map[string]string{"topicName": "mytopic", "mode": "MessageSizes", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// with full link to topic
	{"with full link to topic", nil, map[string]string{"topicName": "projects/myproject/topics/mytopic", "mode": "MessageSizes", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// with full (bad) link to topic
	{"with full (bad) link to topic", nil, map[string]string{"topicName": "projects/myproject/mytopic", "mode": "MessageSizes", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// both subscriptionName and topicName present
	{"both subscriptionName and topicName present", nil, map[string]string{"subscriptionName": "mysubscription", "topicName": "mytopic", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// both subscriptionName and topicName missing
	{"both subscriptionName and topicName missing", nil, map[string]string{"value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// both subscriptionSize and topicName present
	{"both subscriptionSize and topicName present", nil, map[string]string{"subscriptionSize": "7", "topicName": "mytopic", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// both subscriptionName and subscriptionNameFromEnv present
	{"both subscriptionName and subscriptionNameFromEnv present", nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionNameFromEnv": "MY_ENV_SUBSCRIPTION", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// both topicName and topicNameFromEnv present
	{"both topicName and topicNameFromEnv present", nil, map[string]string{"topicName": "mytopic", "topicNameFromEnv": "MY_ENV_TOPIC", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// subscriptionNameFromEnv present
	{"subscriptionNameFromEnv present", nil, map[string]string{"subscriptionNameFromEnv": "MY_ENV_SUBSCRIPTION", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// topicNameFromEnv present
	{"topicNameFromEnv present", nil, map[string]string{"topicNameFromEnv": "MY_ENV_TOPIC", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
}

var gcpPubSubMetricIdentifiers = []gcpPubSubMetricIdentifier{
	{&testPubSubMetadata[1], 0, "s0-gcp-ps-mysubscription"},
	{&testPubSubMetadata[1], 1, "s1-gcp-ps-mysubscription"},
	{&testPubSubMetadata[16], 0, "s0-gcp-ps-mytopic"},
	{&testPubSubMetadata[16], 1, "s1-gcp-ps-mytopic"},
}

var gcpResourceNameTests = []gcpPubSubSubscription{
	{&testPubSubMetadata[11], 1, "mysubscription", "myproject"},
	{&testPubSubMetadata[12], 1, "projects/myproject/mysubscription", ""},
	{&testPubSubMetadata[17], 1, "mytopic", "myproject"},
	{&testPubSubMetadata[18], 1, "projects/myproject/mytopic", ""},
	{&testPubSubMetadata[24], 1, "myEnvSubscription", ""},
	{&testPubSubMetadata[25], 1, "myEnvTopic", ""},
}

var gcpSubscriptionDefaults = []gcpPubSubSubscription{
	{&testPubSubMetadata[14], 0, "", ""},
	{&testPubSubMetadata[15], 0, "", ""},
}

func TestPubSubParseMetadata(t *testing.T) {
	for _, testData := range testPubSubMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			_, err := parsePubSubMetadata(&scalersconfig.ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testPubSubResolvedEnv})
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

func TestPubSubMetadataDefaultValues(t *testing.T) {
	for _, testData := range gcpSubscriptionDefaults {
		t.Run(testData.metadataTestData.testName, func(t *testing.T) {
			metaData, err := parsePubSubMetadata(&scalersconfig.ScalerConfig{AuthParams: testData.metadataTestData.authParams, TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv})
			if err != nil {
				t.Error("Expected success but got error", err)
			}
			if pubSubDefaultModeSubscriptionSize != metaData.Mode {
				t.Errorf(`Expected mode "%s" but got "%s"`, pubSubDefaultModeSubscriptionSize, metaData.Mode)
			}
			if pubSubDefaultValue != metaData.Value {
				t.Errorf(`Expected value "%d" but got "%f"`, pubSubDefaultValue, metaData.Value)
			}
		})
	}
}

func TestGcpPubSubGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpPubSubMetricIdentifiers {
		t.Run(testData.metadataTestData.testName, func(t *testing.T) {
			meta, err := parsePubSubMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv, TriggerIndex: testData.triggerIndex})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockGcpPubSubScaler := pubsubScaler{nil, "", meta, logr.Discard()}

			metricSpec := mockGcpPubSubScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			if metricName != testData.name {
				t.Error("Wrong External metric source name:", metricName)
			}
		})
	}
}

func TestGcpPubSubResourceName(t *testing.T) {
	for _, testData := range gcpResourceNameTests {
		t.Run(testData.metadataTestData.testName, func(t *testing.T) {
			meta, err := parsePubSubMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv, TriggerIndex: testData.triggerIndex})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockGcpPubSubScaler := pubsubScaler{nil, "", meta, logr.Discard()}
			resourceID, projectID := getResourceData(&mockGcpPubSubScaler)

			if resourceID != testData.name || projectID != testData.projectID {
				t.Error("Wrong Resource parsing:", resourceID, projectID)
			}
		})
	}
}

func TestGcpPubSubSnakeCase(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"PullAckRequestCount", "pull_ack_request_count"},
		{"AckLatencies", "ack_latencies"},
		{"AckMessageCount", "ack_message_count"},
		{"BacklogBytes", "backlog_bytes"},
		{"NumOutstandingMessages", "num_outstanding_messages"},
		{"NumUndeliveredMessages", "num_undelivered_messages"},
	}

	for _, tc := range testCases {
		got := snakeCase(tc.input)
		if got != tc.want {
			t.Fatalf(`want "%s" but got "%s"`, tc.want, got)
		}
	}
}

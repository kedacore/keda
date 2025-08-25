package scalers

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/keda-scalers/gcp"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var testGcpCloudTasksResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parseGcpCloudTasksMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
	expected   *gcpCloudTaskMetadata
	comment    string
}

type gcpCloudTasksMetricIdentifier struct {
	metadataTestData *parseGcpCloudTasksMetadataTestData
	triggerIndex     int
	name             string
}

var testGcpCloudTasksMetadata = []parseGcpCloudTasksMetadataTestData{

	{map[string]string{}, map[string]string{}, true, nil, "error case"},

	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectID": "myproject", "activationValue": "5"}, false, &gcpCloudTaskMetadata{
		Value:           7,
		ActivationValue: 5,
		FilterDuration:  0,
		QueueName:       "myQueue",
		ProjectID:       "myproject",
		gcpAuthorization: &gcp.AuthorizationMetadata{
			GoogleApplicationCredentials: "{}",
			PodIdentityProviderEnabled:   false,
		},
		triggerIndex: 0}, "all properly formed"},

	{nil, map[string]string{"queueName": "", "value": "7", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true, nil, "missing subscriptionName"},

	{nil, map[string]string{"queueName": "myQueue", "value": "7", "projectID": "myproject", "credentialsFromEnv": ""}, true, nil, "missing credentials"},

	{nil, map[string]string{"queueName": "myQueue", "value": "AA", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true, nil, "malformed subscriptionSize"},

	{nil, map[string]string{"queueName": "", "mode": "AA", "value": "7", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true, nil, "malformed mode"},

	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectID": "myproject", "activationValue": "AA"}, true, nil, "malformed activationTargetValue"},

	{map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"queueName": "myQueue", "value": "7", "projectID": "myproject"}, false, &gcpCloudTaskMetadata{
		Value:           7,
		ActivationValue: 0,
		FilterDuration:  0,
		QueueName:       "myQueue",
		ProjectID:       "myproject",
		gcpAuthorization: &gcp.AuthorizationMetadata{
			GoogleApplicationCredentials: "Creds",
			PodIdentityProviderEnabled:   false,
		},
		triggerIndex: 0}, "Credentials from AuthParams"},

	{map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"queueName": "myQueue", "subscriptionSize": "7", "projectID": "myproject"}, true, nil, "Credentials from AuthParams with empty creds"},

	{nil, map[string]string{"queueName": "mysubscription", "value": "7.1", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "2.1", "projectID": "myproject"}, false, &gcpCloudTaskMetadata{
		Value:           7.1,
		ActivationValue: 2.1,
		FilterDuration:  0,
		QueueName:       "mysubscription",
		ProjectID:       "myproject",
		gcpAuthorization: &gcp.AuthorizationMetadata{
			GoogleApplicationCredentials: "{}",
			PodIdentityProviderEnabled:   false,
		},
		triggerIndex: 0}, "properly formed float value and activationTargetValue"},

	{nil, map[string]string{"queueName": "myQueue", "projectID": "myProject", "credentialsFromEnv": "SAMPLE_CREDS"}, false, &gcpCloudTaskMetadata{
		Value:           100,
		ActivationValue: 0,
		FilterDuration:  0,
		QueueName:       "myQueue",
		ProjectID:       "myProject",
		gcpAuthorization: &gcp.AuthorizationMetadata{
			GoogleApplicationCredentials: "{}",
			PodIdentityProviderEnabled:   false,
		},
		triggerIndex: 0}, "test default value (100) when value is not provided"},

	{nil, map[string]string{"queueName": "myQueue", "projectID": "myProject", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "5"}, false, &gcpCloudTaskMetadata{
		Value:           100,
		ActivationValue: 5,
		FilterDuration:  0,
		QueueName:       "myQueue",
		ProjectID:       "myProject",
		gcpAuthorization: &gcp.AuthorizationMetadata{
			GoogleApplicationCredentials: "{}",
			PodIdentityProviderEnabled:   false,
		},
		triggerIndex: 0}, "test default value with specified activationVal"},

	{nil, map[string]string{"queueName": "myQueue", "projectID": "myProject", "credentialsFromEnv": "SAMPLE_CREDS", "filterDuration": "invalid"}, true, nil, "test invalid filterDuration with default values"},
}

var gcpCloudTasksMetricIdentifiers = []gcpCloudTasksMetricIdentifier{
	{&testGcpCloudTasksMetadata[1], 0, "s0-gcp-ct-myQueue"},
	{&testGcpCloudTasksMetadata[1], 1, "s1-gcp-ct-myQueue"},
}

func TestGcpCloudTasksParseMetadata(t *testing.T) {
	for _, testData := range testGcpCloudTasksMetadata {
		t.Run(testData.comment, func(t *testing.T) {
			metadata, err := parseGcpCloudTasksMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      testData.authParams,
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testGcpCloudTasksResolvedEnv,
			})

			if err != nil && !testData.isError {
				t.Errorf("Expected success but got error")
			}

			if testData.isError && err == nil {
				t.Errorf("Expected error but got success")
			}

			if !testData.isError && !reflect.DeepEqual(testData.expected, metadata) {
				t.Fatalf("Expected %#v but got %+#v", testData.expected, metadata)
			}
		})
	}
}

func TestGcpCloudTasksGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpCloudTasksMetricIdentifiers {
		meta, err := parseGcpCloudTasksMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testGcpCloudTasksResolvedEnv, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpCloudTasksScaler := gcpCloudTasksScaler{nil, "", meta, logr.Discard()}

		metricSpec := mockGcpCloudTasksScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

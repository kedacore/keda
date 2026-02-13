package scalers

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testStackdriverResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parseStackdriverMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
	expected   *stackdriverMetadata
	comment    string
}

var sdFilter = "metric.type=\"storage.googleapis.com/storage/object_count\" resource.type=\"gcs_bucket\""

var testStackdriverMetadata = []parseStackdriverMetadataTestData{

	{
		authParams: map[string]string{},
		metadata:   map[string]string{},
		isError:    true,
		expected:   nil,
		comment:    "error case - empty metadata",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"projectId":             "myProject",
			"filter":                sdFilter,
			"targetValue":           "7",
			"credentialsFromEnv":    "SAMPLE_CREDS",
			"activationTargetValue": "5",
		},
		isError: false,
		expected: &stackdriverMetadata{
			ProjectID:             "myProject",
			Filter:                sdFilter,
			TargetValue:           7,
			ActivationTargetValue: 5,
			Credentials:           "{}",
			metricName:            "s0-gcp-stackdriver-myProject",
			aggregation:           nil,
			gcpAuthorization: &gcp.AuthorizationMetadata{
				GoogleApplicationCredentials: "{}",
				PodIdentityProviderEnabled:   false,
			},
		},
		comment: "all properly formed",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"projectId":          "myProject",
			"filter":             sdFilter,
			"credentialsFromEnv": "SAMPLE_CREDS",
		},
		isError: false,
		expected: &stackdriverMetadata{
			ProjectID:             "myProject",
			Filter:                sdFilter,
			TargetValue:           5,
			ActivationTargetValue: 0,
			Credentials:           "{}",
			metricName:            "s0-gcp-stackdriver-myProject",
			aggregation:           nil,
			gcpAuthorization: &gcp.AuthorizationMetadata{
				GoogleApplicationCredentials: "{}",
				PodIdentityProviderEnabled:   false,
			},
		},
		comment: "required fields only with defaults",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"projectId":          "myProject",
			"filter":             sdFilter,
			"credentialsFromEnv": "SAMPLE_CREDS",
			"valueIfNull":        "1.5",
		},
		isError: false,
		expected: &stackdriverMetadata{
			ProjectID:             "myProject",
			Filter:                sdFilter,
			TargetValue:           5,
			ActivationTargetValue: 0,
			Credentials:           "{}",
			ValueIfNull:           func() *float64 { v := 1.5; return &v }(),
			metricName:            "s0-gcp-stackdriver-myProject",
			aggregation:           nil,
			gcpAuthorization: &gcp.AuthorizationMetadata{
				GoogleApplicationCredentials: "{}",
				PodIdentityProviderEnabled:   false,
			},
		},
		comment: "with valueIfNull configuration",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"filter":             sdFilter,
			"credentialsFromEnv": "SAMPLE_CREDS",
		},
		isError:  true,
		expected: nil,
		comment:  "error case - missing projectId",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"projectId":          "myProject",
			"credentialsFromEnv": "SAMPLE_CREDS",
		},
		isError:  true,
		expected: nil,
		comment:  "error case - missing filter",
	},
	{
		authParams: nil,
		metadata: map[string]string{
			"projectId": "myProject",
			"filter":    sdFilter,
		},
		isError:  true,
		expected: nil,
		comment:  "error case - missing credentials",
	},
}

func TestStackdriverParseMetadata(t *testing.T) {
	for _, testData := range testStackdriverMetadata {
		t.Run(testData.comment, func(t *testing.T) {
			metadata, err := parseStackdriverMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      testData.authParams,
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testStackdriverResolvedEnv,
			})

			if err != nil && !testData.isError {
				t.Errorf("Expected success but got error: %v", err)
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

var gcpStackdriverMetricIdentifiers = []struct {
	comment      string
	triggerIndex int
	metadata     map[string]string
	expectedName string
}{
	{
		comment:      "basic metric name",
		triggerIndex: 0,
		metadata: map[string]string{
			"projectId":          "myProject",
			"filter":             sdFilter,
			"credentialsFromEnv": "SAMPLE_CREDS",
		},
		expectedName: "s0-gcp-stackdriver-myProject",
	},
	{
		comment:      "metric name with different index",
		triggerIndex: 1,
		metadata: map[string]string{
			"projectId":          "myProject",
			"filter":             sdFilter,
			"credentialsFromEnv": "SAMPLE_CREDS",
		},
		expectedName: "s1-gcp-stackdriver-myProject",
	},
}

func TestGcpStackdriverGetMetricSpecForScaling(t *testing.T) {
	for _, test := range gcpStackdriverMetricIdentifiers {
		t.Run(test.comment, func(t *testing.T) {
			meta, err := parseStackdriverMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: test.metadata,
				ResolvedEnv:     testStackdriverResolvedEnv,
				TriggerIndex:    test.triggerIndex,
			})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			mockScaler := stackdriverScaler{
				metadata: meta,
				logger:   logr.Discard(),
			}

			metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			if metricName != test.expectedName {
				t.Errorf("Wrong metric name - got %s, want %s", metricName, test.expectedName)
			}
		})
	}
}

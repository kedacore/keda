package scalers

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testAWSSQSRoleArn         = "none"
	testAWSSQSExternalID      = "test-external-id"
	testAWSSQSAccessKeyID     = "none"
	testAWSSQSSecretAccessKey = "none"
	testAWSSQSSessionToken    = "none"

	testAWSSQSProperQueueURL    = "https://sqs.eu-west-1.amazonaws.com/account_id/DeleteArtifactQ"
	testAWSSQSImproperQueueURL1 = "https://sqs.eu-west-1.amazonaws.com/account_id"
	testAWSSQSImproperQueueURL2 = "https://sqs.eu-west-1.amazonaws.com"
	testAWSSimpleQueueURL       = "my-queue"

	testAWSSQSErrorQueueURL   = "https://sqs.eu-west-1.amazonaws.com/account_id/Error"
	testAWSSQSBadDataQueueURL = "https://sqs.eu-west-1.amazonaws.com/account_id/BadData"

	testAWSSQSApproximateNumberOfMessagesVisible    = 200
	testAWSSQSApproximateNumberOfMessagesNotVisible = 100
	testAWSSQSApproximateNumberOfMessagesDelayed    = 50
)

var testAWSSQSEmptyResolvedEnv = map[string]string{}

var testAWSSQSResolvedEnv = map[string]string{
	"QUEUE_URL": testAWSSQSProperQueueURL,
}

var testAWSSQSAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSSQSAccessKeyID,
	"awsSecretAccessKey": testAWSSQSSecretAccessKey,
}

var (
	testAWSSQSRoleArnPtr    = testAWSSQSRoleArn
	testAWSSQSExternalIDPtr = testAWSSQSExternalID
)

type parseAWSSQSMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	podIdentity kedav1alpha1.AuthPodIdentity
	isError     bool
	comment     string
}

type awsSQSMetricIdentifier struct {
	metadataTestData *parseAWSSQSMetadataTestData
	triggerIndex     int
	name             string
}

type mockSqs struct {
}

func (m *mockSqs) GetQueueAttributes(_ context.Context, input *sqs.GetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	switch *input.QueueUrl {
	case testAWSSQSErrorQueueURL:
		return nil, errors.New("some error")
	case testAWSSQSBadDataQueueURL:
		return &sqs.GetQueueAttributesOutput{
			Attributes: map[string]string{
				"ApproximateNumberOfMessages":           "NotInt",
				"ApproximateNumberOfMessagesNotVisible": "NotInt",
			},
		}, nil
	}

	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			"ApproximateNumberOfMessages":           strconv.Itoa(testAWSSQSApproximateNumberOfMessagesVisible),
			"ApproximateNumberOfMessagesNotVisible": strconv.Itoa(testAWSSQSApproximateNumberOfMessagesNotVisible),
			"ApproximateNumberOfMessagesDelayed":    strconv.Itoa(testAWSSQSApproximateNumberOfMessagesDelayed),
		},
	}, nil
}

var testAWSSQSMetadata = []parseAWSSQSMetadataTestData{
	{map[string]string{},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"metadata empty"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1",
		"awsEndpoint": "http://localhost:4566"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue and region with custom endpoint"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL1,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"improperly formed queue, missing queueName"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL2,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"improperly formed queue, missing path"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   ""},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"properly formed queue, empty region"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue, integer queueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "a",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"invalid integer value for queueLength"},
	{map[string]string{
		"queueURL":              testAWSSQSProperQueueURL,
		"queueLength":           "1",
		"activationQueueLength": "1",
		"awsRegion":             "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue, integer activationQueueLength"},
	{map[string]string{
		"queueURL":              testAWSSQSProperQueueURL,
		"queueLength":           "1",
		"activationQueueLength": "a",
		"awsRegion":             "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"invalid integer value for activationQueueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"with AWS static credentials from TriggerAuthentication"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"with AWS temporary credentials from TriggerAuthentication"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"with AWS static credentials from TriggerAuthentication, missing Access Key Id"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": "",
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"with AWS temporary credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"with AWS temporary credentials from TriggerAuthentication, missing Access Key Id"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": "",
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"with AWS static credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{
			Provider: kedav1alpha1.PodIdentityProviderAws,
			RoleArn:  &testAWSSQSRoleArnPtr,
		},
		false,
		"with AWS Role from Pod Identity"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{
			Provider:      kedav1alpha1.PodIdentityProviderAws,
			RoleArn:       &testAWSSQSRoleArnPtr,
			AwsExternalID: &testAWSSQSExternalIDPtr,
		},
		false,
		"with AWS Role and External ID from Pod Identity"},
	{map[string]string{
		"queueURL":      testAWSSQSProperQueueURL,
		"queueLength":   "1",
		"awsRegion":     "eu-west-1",
		"identityOwner": "operator"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": "",
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"with AWS Role assigned on KEDA operator itself"},
	{map[string]string{
		"queueURL":      testAWSSQSProperQueueURL,
		"queueLength":   "1",
		"awsRegion":     "eu-west-1",
		"identityOwner": "operator"},
		map[string]string{
			"awsRoleArn":         testAWSSQSRoleArn,
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": "",
		},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"with AWS Role assigned on KEDA operator itself (deprecated path)"},
	{map[string]string{
		"queueURL":    testAWSSimpleQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":        testAWSSimpleQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":        testAWSSimpleQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURLFromEnv": "QUEUE_URL",
		"queueLength":     "1",
		"awsRegion":       "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"properly formed queue loaded from env"},
	{map[string]string{
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"missing queue url from both queueURL and queueURLFromEnv"},
	{map[string]string{
		"queueURLFromEnv": "QUEUE_URL",
		"queueLength":     "1",
		"awsRegion":       "eu-west-1"},
		testAWSSQSAuthentication,
		map[string]string{
			"QUEUE_URL": "",
		},
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		true,
		"empty QUEUE_URL env value"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{},
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{
			Provider:      kedav1alpha1.PodIdentityProviderAws,
			AwsExternalID: &testAWSSQSExternalIDPtr,
		},
		false,
		"with External ID but missing Role ARN (still valid, just won't use external ID)"},
}

var awsSQSMetricIdentifiers = []awsSQSMetricIdentifier{
	{&testAWSSQSMetadata[1], 0, "s0-aws-sqs-DeleteArtifactQ"},
	{&testAWSSQSMetadata[1], 1, "s1-aws-sqs-DeleteArtifactQ"},
}

var awsSQSGetMetricTestData = []*parseAWSSQSMetadataTestData{
	{map[string]string{
		"queueURL":        testAWSSQSProperQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaleOnInFlight disabled"},
	{map[string]string{
		"queueURL":        testAWSSQSProperQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaleOnInFlight enabled"},
	{map[string]string{
		"queueURL":       testAWSSQSProperQueueURL,
		"queueLength":    "1",
		"awsRegion":      "eu-west-1",
		"scaleOnDelayed": "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaleOnDelayed disabled"},
	{map[string]string{
		"queueURL":       testAWSSQSProperQueueURL,
		"queueLength":    "1",
		"awsRegion":      "eu-west-1",
		"scaleOnDelayed": "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaleOnDelayed enabled"},
	{map[string]string{
		"queueURL":        testAWSSQSProperQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "false",
		"scaleOnDelayed":  "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaledOnInFlight and scaleOnDelayed disabled"},
	{map[string]string{
		"queueURL":        testAWSSQSProperQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "true",
		"scaleOnDelayed":  "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"not error with scaledOnInFlight and scaleOnDelayed enabled"},
	{map[string]string{
		"queueURL":        testAWSSQSErrorQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"error queue"},
	{map[string]string{
		"queueURL":        testAWSSQSBadDataQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		false,
		"bad data"},
}

func TestSQSParseMetadata(t *testing.T) {
	for _, testData := range testAWSSQSMetadata {
		_, err := parseAwsSqsQueueMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams, PodIdentity: testData.podIdentity})
		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}

func TestAWSSQSGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsSQSMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsSqsQueueMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, PodIdentity: testData.metadataTestData.podIdentity, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSSQSScaler := awsSqsQueueScaler{"", meta, &mockSqs{}, logr.Discard()}

		metricSpec := mockAWSSQSScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAWSSQSScalerGetMetrics(t *testing.T) {
	for index, testData := range awsSQSGetMetricTestData {
		meta, err := parseAwsSqsQueueMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams, PodIdentity: testData.podIdentity, TriggerIndex: index})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		scaler := awsSqsQueueScaler{"", meta, &mockSqs{}, logr.Discard()}

		value, _, err := scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		switch meta.QueueURL {
		case testAWSSQSErrorQueueURL:
			assert.Error(t, err, "expect error because of sqs api error")
		case testAWSSQSBadDataQueueURL:
			assert.Error(t, err, "expect error because of bad data return from sqs")
		default:
			expectedMessages := testAWSSQSApproximateNumberOfMessagesVisible
			if meta.ScaleOnInFlight {
				expectedMessages += testAWSSQSApproximateNumberOfMessagesNotVisible
			}
			if meta.ScaleOnDelayed {
				expectedMessages += testAWSSQSApproximateNumberOfMessagesDelayed
			}
			assert.EqualValues(t, int64(expectedMessages), value[0].Value.Value())
		}
	}
}

func TestProcessQueueLengthFromSqsQueueAttributesOutput(t *testing.T) {
	scalerCreationFunc := func() *awsSqsQueueScaler {
		return &awsSqsQueueScaler{
			metadata: &awsSqsQueueMetadata{
				awsSqsQueueMetricNames: []types.QueueAttributeName{types.QueueAttributeNameApproximateNumberOfMessages, types.QueueAttributeNameApproximateNumberOfMessagesNotVisible, types.QueueAttributeNameApproximateNumberOfMessagesDelayed},
			},
		}
	}

	tests := map[string]struct {
		s           *awsSqsQueueScaler
		attributes  *sqs.GetQueueAttributesOutput
		expected    int64
		errExpected bool
	}{
		"properly formed queue attributes": {
			s: scalerCreationFunc(),
			attributes: &sqs.GetQueueAttributesOutput{
				Attributes: map[string]string{
					"ApproximateNumberOfMessages":           "1",
					"ApproximateNumberOfMessagesNotVisible": "0",
					"ApproximateNumberOfMessagesDelayed":    "0",
				},
			},
			expected:    1,
			errExpected: false,
		},
		"missing ApproximateNumberOfMessages": {
			s: scalerCreationFunc(),
			attributes: &sqs.GetQueueAttributesOutput{
				Attributes: map[string]string{},
			},
			expected:    -1,
			errExpected: true,
		},
		"invalid ApproximateNumberOfMessages": {
			s: scalerCreationFunc(),
			attributes: &sqs.GetQueueAttributesOutput{
				Attributes: map[string]string{
					"ApproximateNumberOfMessages":           "NotInt",
					"ApproximateNumberOfMessagesNotVisible": "0",
					"ApproximateNumberOfMessagesDelayed":    "0",
				},
			},
			expected:    -1,
			errExpected: true,
		},
		"32 bit int upper bound": {
			s: scalerCreationFunc(),
			attributes: &sqs.GetQueueAttributesOutput{
				Attributes: map[string]string{
					"ApproximateNumberOfMessages":           "2147483647",
					"ApproximateNumberOfMessagesNotVisible": "0",
					"ApproximateNumberOfMessagesDelayed":    "0",
				},
			},
			expected:    2147483647,
			errExpected: false,
		},
		"32 bit int upper bound + 1": {
			s: scalerCreationFunc(),
			attributes: &sqs.GetQueueAttributesOutput{
				Attributes: map[string]string{
					"ApproximateNumberOfMessages":           "2147483648",
					"ApproximateNumberOfMessagesNotVisible": "0",
					"ApproximateNumberOfMessagesDelayed":    "0",
				},
			},
			expected:    2147483648,
			errExpected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := test.s.processQueueLengthFromSqsQueueAttributesOutput(test.attributes)

			if test.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.expected, result)
		})
	}
}

func TestQueueURLFromEnvResolution(t *testing.T) {
	testCases := []struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		expectedURL string
		expectError bool
	}{
		{
			name: "direct queueURL",
			metadata: map[string]string{
				"queueURL":  testAWSSQSProperQueueURL,
				"awsRegion": "eu-west-1",
			},
			resolvedEnv: map[string]string{},
			expectedURL: testAWSSQSProperQueueURL,
			expectError: false,
		},
		{
			name: "queueURL from environment variable",
			metadata: map[string]string{
				"queueURLFromEnv": "QUEUE_URL",
				"awsRegion":       "eu-west-1",
			},
			resolvedEnv: map[string]string{
				"QUEUE_URL": testAWSSQSProperQueueURL,
			},
			expectedURL: testAWSSQSProperQueueURL,
			expectError: false,
		},
		{
			name: "missing environment variable",
			metadata: map[string]string{
				"queueURLFromEnv": "MISSING_ENV_VAR",
				"awsRegion":       "eu-west-1",
			},
			resolvedEnv: map[string]string{
				"QUEUE_URL": testAWSSQSProperQueueURL,
			},
			expectedURL: "",
			expectError: true,
		},
		{
			name: "empty environment variable value",
			metadata: map[string]string{
				"queueURLFromEnv": "EMPTY_ENV_VAR",
				"awsRegion":       "eu-west-1",
			},
			resolvedEnv: map[string]string{
				"EMPTY_ENV_VAR": "",
			},
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meta, err := parseAwsSqsQueueMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				ResolvedEnv:     tc.resolvedEnv,
				AuthParams:      testAWSSQSAuthentication,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
			})

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedURL, meta.QueueURL)
			}
		})
	}
}

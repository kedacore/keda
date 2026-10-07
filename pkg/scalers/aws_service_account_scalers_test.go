package scalers

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestAWSServiceAccountScalerRouting(t *testing.T) {
	t.Setenv("AWS_PROFILE", "missing-operator-profile")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing-config"))
	for _, test := range []struct {
		name     string
		metadata map[string]string
		parse    func(*scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error)
	}{
		{
			name: "SQS", metadata: map[string]string{"queueURL": "tenant-queue"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseAwsSqsQueueMetadata(c)
				if err != nil {
					return awsutils.AuthorizationMetadata{}, err
				}
				return m.awsAuthorization, nil
			},
		},
		{
			name: "Kinesis", metadata: map[string]string{"streamName": "tenant-stream"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseAwsKinesisStreamMetadata(c)
				if err != nil {
					return awsutils.AuthorizationMetadata{}, err
				}
				return m.awsAuthorization, nil
			},
		},
		{
			name: "DynamoDB Streams", metadata: map[string]string{"tableName": "tenant-table"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseAwsDynamoDBStreamsMetadata(c)
				if err != nil {
					return awsutils.AuthorizationMetadata{}, err
				}
				return m.awsAuthorization, nil
			},
		},
		{
			name: "DynamoDB", metadata: map[string]string{
				"tableName": "tenant-table", "keyConditionExpression": "#status = :queued", "targetValue": "5",
				"expressionAttributeNames": `{"#status":"status"}`, "expressionAttributeValues": `{":queued":{"S":"queued"}}`,
			},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseAwsDynamoDBMetadata(c)
				if err != nil {
					return awsutils.AuthorizationMetadata{}, err
				}
				return m.awsAuthorization, nil
			},
		},
		{
			name: "CloudWatch", metadata: map[string]string{"expression": "SUM(METRICS())", "targetMetricValue": "5", "minMetricValue": "0"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseAwsCloudwatchMetadata(c)
				if err != nil {
					return awsutils.AuthorizationMetadata{}, err
				}
				return m.awsAuthorization, nil
			},
		},
		{
			name: "Apache Kafka", metadata: map[string]string{"bootstrapServers": "broker:9092", "consumerGroup": "group", "tls": "enable", "sasl": "aws_msk_iam"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseApacheKafkaMetadata(c)
				return m.AWSAuthorization, err
			},
		},
		{
			name: "Kafka", metadata: map[string]string{"bootstrapServers": "broker:9092", "consumerGroup": "group", "tls": "enable", "sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam"},
			parse: func(c *scalersconfig.ScalerConfig) (awsutils.AuthorizationMetadata, error) {
				m, err := parseKafkaMetadata(c, logr.Discard())
				return m.awsAuthorization, err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			test.metadata["awsRegion"] = "us-east-1"
			config := &scalersconfig.ScalerConfig{
				TriggerUniqueKey: "aws-service-account-" + test.name, TriggerMetadata: test.metadata,
				PodIdentity: kedav1alpha1.AuthPodIdentity{
					Provider: kedav1alpha1.PodIdentityProviderAws, ServiceAccountName: new("scaling-reader"), RoleArn: new("arn:aws:iam::123456789012:role/scaling-reader"),
				},
				ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
					Namespace: "tenant", ServiceAccountName: "scaling-reader", Audience: "sts.amazonaws.com",
					GetToken: func(context.Context) (string, error) {
						requests++
						return "", errors.New("selected service account unavailable")
					},
				},
			}
			auth, err := test.parse(config)
			require.NoError(t, err)
			assert.Same(t, config.ServiceAccountTokenProvider, auth.ServiceAccountTokenProvider)
			t.Cleanup(func() { awsutils.ClearAwsConfig(auth) })
			cfg, err := awsutils.GetAwsConfig(t.Context(), auth)
			require.NoError(t, err)
			_, err = cfg.Credentials.Retrieve(t.Context())
			require.ErrorContains(t, err, "selected service account unavailable")
			assert.Equal(t, 1, requests)
			config.ServiceAccountTokenProvider = nil
			_, err = test.parse(config)
			require.ErrorContains(t, err, "resolved token provider")
		})
	}
}

func TestAWSServiceAccountKafkaRejectsOtherAuthentication(t *testing.T) {
	for _, saslType := range []string{"", "plaintext"} {
		t.Run(saslType, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{"bootstrapServers": "broker:9092", "consumerGroup": "group", "tls": "enable", "sasl": saslType},
				AuthParams:      map[string]string{"username": "operator-user", "password": "operator-password"},
				PodIdentity: kedav1alpha1.AuthPodIdentity{
					Provider: kedav1alpha1.PodIdentityProviderAws, ServiceAccountName: new("scaling-reader"), RoleArn: new("arn:aws:iam::123456789012:role/scaling-reader"),
				},
			}
			_, err := parseKafkaMetadata(config, logr.Discard())
			require.ErrorContains(t, err, "serviceAccountName requires AWS MSK IAM")
			_, err = parseApacheKafkaMetadata(config)
			require.ErrorContains(t, err, "serviceAccountName requires AWS MSK IAM")
		})
	}
}

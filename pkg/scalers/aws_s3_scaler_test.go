/*
Copyright 2025 The KEDA Authors

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
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testAWSS3AccessKeyID     = "none"
	testAWSS3SecretAccessKey = "none"

	// Bucket names double as scenario selectors for the mock client below.
	testAWSS3Bucket          = "test-bucket"
	testAWSS3ErrorBucket     = "error-bucket"
	testAWSS3EmptyBucket     = "empty-bucket"
	testAWSS3GlobBucket      = "glob-bucket"
	testAWSS3PaginatedBucket = "paginated-bucket"
)

var testAWSS3EmptyResolvedEnv = map[string]string{}

var testAWSS3Authentication = map[string]string{
	"awsAccessKeyID":     testAWSS3AccessKeyID,
	"awsSecretAccessKey": testAWSS3SecretAccessKey,
}

type parseAWSS3MetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	isError     bool
	comment     string
}

type awsS3MetricIdentifier struct {
	metadataTestData *parseAWSS3MetadataTestData
	triggerIndex     int
	name             string
}

// mockS3 satisfies S3WrapperClient and returns deterministic pages keyed off the
// requested bucket, so pagination and glob matching can be exercised offline.
type mockS3 struct{}

func obj(key string) types.Object {
	return types.Object{Key: aws.String(key)}
}

func (m *mockS3) ListObjectsV2(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	switch *input.Bucket {
	case testAWSS3ErrorBucket:
		return nil, errors.New("some error")
	case testAWSS3EmptyBucket:
		return &s3.ListObjectsV2Output{Contents: []types.Object{}}, nil
	case testAWSS3GlobBucket:
		// Recursive listing (no delimiter is set by the scaler in glob mode).
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{
				obj("logs/a.txt"),
				obj("logs/b.txt"),
				obj("logs/c.json"),
				obj("logs/nested/d.txt"),
			},
		}, nil
	case testAWSS3PaginatedBucket:
		// First page is truncated; the SDK paginator re-requests with the token.
		if input.ContinuationToken == nil {
			return &s3.ListObjectsV2Output{
				Contents:              []types.Object{obj("1"), obj("2")},
				IsTruncated:           aws.Bool(true),
				NextContinuationToken: aws.String("page2"),
			}, nil
		}
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{obj("3"), obj("4"), obj("5")},
		}, nil
	default:
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{obj("x"), obj("y"), obj("z")},
		}, nil
	}
}

var testAWSS3Metadata = []parseAWSS3MetadataTestData{
	{
		map[string]string{},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		true,
		"metadata empty",
	},
	{
		map[string]string{
			"bucketName": testAWSS3Bucket,
			"awsRegion":  "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		false,
		"properly formed bucket and region",
	},
	{
		map[string]string{
			"bucketName":        testAWSS3Bucket,
			"targetObjectCount": "10",
			"awsRegion":         "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		false,
		"properly formed with target object count",
	},
	{
		map[string]string{
			"bucketName":  testAWSS3Bucket,
			"prefix":      "logs/",
			"delimiter":   "/",
			"awsRegion":   "eu-west-1",
			"awsEndpoint": "http://localhost:4566",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		false,
		"properly formed with prefix, delimiter and custom endpoint",
	},
	{
		map[string]string{
			"bucketName": testAWSS3Bucket,
			"recursive":  "true",
			"awsRegion":  "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		false,
		"recursive listing",
	},
	{
		map[string]string{
			"bucketName":  testAWSS3Bucket,
			"globPattern": "logs/*.txt",
			"awsRegion":   "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		false,
		"valid glob pattern",
	},
	{
		map[string]string{
			"bucketName":  testAWSS3Bucket,
			"globPattern": "[",
			"awsRegion":   "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		true,
		"invalid glob pattern",
	},
	{
		map[string]string{
			"awsRegion": "eu-west-1",
		},
		testAWSS3Authentication,
		testAWSS3EmptyResolvedEnv,
		true,
		"missing bucketName",
	},
	{
		map[string]string{
			"bucketName": testAWSS3Bucket,
		},
		map[string]string{
			"awsRegion":          "eu-west-1",
			"awsAccessKeyID":     testAWSS3AccessKeyID,
			"awsSecretAccessKey": testAWSS3SecretAccessKey,
		},
		testAWSS3EmptyResolvedEnv,
		false,
		"awsRegion supplied via authParams",
	},
	{
		map[string]string{
			"bucketName":    testAWSS3Bucket,
			"awsRegion":     "eu-west-1",
			"identityOwner": "operator",
		},
		map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": "",
		},
		testAWSS3EmptyResolvedEnv,
		false,
		"with AWS role assigned to the KEDA operator",
	},
}

var awsS3MetricIdentifiers = []awsS3MetricIdentifier{
	{&testAWSS3Metadata[1], 0, "s0-aws-s3-test-bucket"},
	{&testAWSS3Metadata[1], 1, "s1-aws-s3-test-bucket"},
}

func TestS3ParseMetadata(t *testing.T) {
	for _, testData := range testAWSS3Metadata {
		_, err := parseAwsS3Metadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			ResolvedEnv:     testData.resolvedEnv,
			AuthParams:      testData.authParams,
		}, logr.Discard())
		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}

func TestS3ParseMetadataDefaults(t *testing.T) {
	meta, err := parseAwsS3Metadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"bucketName": testAWSS3Bucket, "awsRegion": "eu-west-1"},
		AuthParams:      testAWSS3Authentication,
	}, logr.Discard())
	assert.NoError(t, err)
	assert.Equal(t, int64(5), meta.TargetObjectCount)
	assert.Equal(t, int64(0), meta.ActivationTargetObjectCount)
	assert.Equal(t, "/", meta.Delimiter)
	assert.Equal(t, int32(1000), meta.MaxKeys)
	assert.Nil(t, meta.globPattern)
}

func TestS3RecursiveClearsDelimiter(t *testing.T) {
	meta, err := parseAwsS3Metadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"bucketName": testAWSS3Bucket, "awsRegion": "eu-west-1", "recursive": "true"},
		AuthParams:      testAWSS3Authentication,
	}, logr.Discard())
	assert.NoError(t, err)
	assert.Equal(t, "", meta.Delimiter)
}

func TestAWSS3GetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsS3MetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsS3Metadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			ResolvedEnv:     testData.metadataTestData.resolvedEnv,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockScaler := awsS3Scaler{
			metricType:      "",
			metadata:        meta,
			s3WrapperClient: &mockS3{},
			logger:          logr.Discard(),
		}

		metricSpec := mockScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

type awsS3GetMetricTestCase struct {
	metadata       map[string]string
	expectedCount  int64
	expectedActive bool
	isError        bool
	comment        string
}

var awsS3GetMetricTestData = []awsS3GetMetricTestCase{
	{
		map[string]string{"bucketName": testAWSS3Bucket, "awsRegion": "eu-west-1"},
		3, true, false, "delimiter listing counts immediate objects",
	},
	{
		map[string]string{"bucketName": testAWSS3EmptyBucket, "awsRegion": "eu-west-1"},
		0, false, false, "empty bucket is inactive",
	},
	{
		// gobwas/glob without a separator lets '*' span '/', mirroring the
		// Azure Blob scaler, so this matches every .txt key including the nested one.
		map[string]string{"bucketName": testAWSS3GlobBucket, "globPattern": "logs/*.txt", "awsRegion": "eu-west-1"},
		3, true, false, "glob matches all .txt keys",
	},
	{
		map[string]string{"bucketName": testAWSS3GlobBucket, "globPattern": "*.json", "awsRegion": "eu-west-1"},
		1, true, false, "glob matches only the .json key",
	},
	{
		map[string]string{"bucketName": testAWSS3PaginatedBucket, "awsRegion": "eu-west-1"},
		5, true, false, "pagination sums objects across pages",
	},
	{
		map[string]string{"bucketName": testAWSS3EmptyBucket, "activationTargetObjectCount": "0", "awsRegion": "eu-west-1"},
		0, false, false, "activation threshold not crossed",
	},
	{
		map[string]string{"bucketName": testAWSS3Bucket, "activationTargetObjectCount": "5", "awsRegion": "eu-west-1"},
		3, false, false, "count below activation threshold is inactive",
	},
	{
		map[string]string{"bucketName": testAWSS3ErrorBucket, "awsRegion": "eu-west-1"},
		0, false, true, "list error is surfaced",
	},
}

func TestAWSS3ScalerGetMetrics(t *testing.T) {
	for _, testData := range awsS3GetMetricTestData {
		meta, err := parseAwsS3Metadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			AuthParams:      testAWSS3Authentication,
		}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		scaler := awsS3Scaler{
			metricType:      "",
			metadata:        meta,
			s3WrapperClient: &mockS3{},
			logger:          logr.Discard(),
		}

		value, active, err := scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		if testData.isError {
			assert.Error(t, err, "expected error for: %s", testData.comment)
			continue
		}
		assert.NoError(t, err, "unexpected error for: %s", testData.comment)
		assert.EqualValues(t, testData.expectedCount, value[0].Value.Value(), testData.comment)
		assert.Equal(t, testData.expectedActive, active, testData.comment)
	}
}

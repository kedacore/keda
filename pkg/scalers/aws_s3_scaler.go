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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
	"github.com/gobwas/glob"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsS3Scaler struct {
	metricType      v2.MetricTargetType
	metadata        *awsS3Metadata
	s3WrapperClient S3WrapperClient
	logger          logr.Logger
}

type awsS3Metadata struct {
	TargetObjectCount           int64  `keda:"name=targetObjectCount,           order=triggerMetadata, default=5"`
	ActivationTargetObjectCount int64  `keda:"name=activationTargetObjectCount, order=triggerMetadata, default=0"`
	BucketName                  string `keda:"name=bucketName,                  order=triggerMetadata"`
	Prefix                      string `keda:"name=prefix,                      order=triggerMetadata, optional"`
	Delimiter                   string `keda:"name=delimiter,                   order=triggerMetadata, default=/"`
	Recursive                   bool   `keda:"name=recursive,                   order=triggerMetadata, optional"`
	GlobPattern                 string `keda:"name=globPattern,                 order=triggerMetadata, optional"`
	MaxKeys                     int32  `keda:"name=maxKeys,                     order=triggerMetadata, default=1000"`
	AwsRegion                   string `keda:"name=awsRegion,                   order=triggerMetadata;authParams"`
	AwsEndpoint                 string `keda:"name=awsEndpoint,                 order=triggerMetadata, optional"`
	awsAuthorization            awsutils.AuthorizationMetadata
	triggerIndex                int
	globPattern                 *glob.Glob
}

// NewAwsS3Scaler creates a new awsS3Scaler
func NewAwsS3Scaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_s3_scaler")

	meta, err := parseAwsS3Metadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing S3 metadata: %w", err)
	}

	s3Client, err := createS3Client(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error when creating s3 client: %w", err)
	}
	return &awsS3Scaler{
		metricType: metricType,
		metadata:   meta,
		s3WrapperClient: &s3WrapperClient{
			s3Client: s3Client,
		},
		logger: logger,
	}, nil
}

// S3WrapperClient is a thin interface over the S3 ListObjectsV2 operation so the
// scaler can be unit tested with a mock. It intentionally matches
// s3.ListObjectsV2APIClient so it can be handed to the SDK paginator directly.
type S3WrapperClient interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type s3WrapperClient struct {
	s3Client *s3.Client
}

func (w s3WrapperClient) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return w.s3Client.ListObjectsV2(ctx, params, optFns...)
}

func parseAwsS3Metadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*awsS3Metadata, error) {
	meta := &awsS3Metadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing S3 metadata: %w", err)
	}

	// A recursive listing is expressed to S3 by using an empty delimiter, which
	// flattens the virtual folder hierarchy. This mirrors the Azure Blob scaler.
	if meta.Recursive {
		meta.Delimiter = ""
	}

	if meta.GlobPattern != "" {
		g, err := glob.Compile(meta.GlobPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern - %w", err)
		}
		meta.globPattern = &g
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AwsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.awsAuthorization = auth
	meta.triggerIndex = config.TriggerIndex

	logger.V(1).Info("parsed S3 metadata", "bucketName", meta.BucketName, "prefix", meta.Prefix, "delimiter", meta.Delimiter, "recursive", meta.Recursive, "globPattern", meta.GlobPattern)

	return meta, nil
}

func createS3Client(ctx context.Context, metadata *awsS3Metadata) (*s3.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(*cfg, func(options *s3.Options) {
		if metadata.AwsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.AwsEndpoint)
			// Custom endpoints (e.g. LocalStack, MinIO) generally require
			// path-style addressing since they don't serve virtual-hosted URLs.
			options.UsePathStyle = true
		}
	}), nil
}

func (s *awsS3Scaler) Close(context.Context) error {
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsS3Scaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-s3-%s", s.metadata.BucketName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetObjectCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns the number of objects in the bucket and whether
// the scaler is active, or an error if there is a problem getting the metric.
func (s *awsS3Scaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	objectCount, err := s.getS3ObjectCount(ctx)
	if err != nil {
		s.logger.Error(err, "Error getting object count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(objectCount))

	return []external_metrics.ExternalMetricValue{metric}, objectCount > s.metadata.ActivationTargetObjectCount, nil
}

// getS3ObjectCount lists the bucket and returns the number of matching objects.
//
// When a globPattern is set, the whole prefix subtree is listed recursively and
// every object key is matched against the pattern. Otherwise objects are listed
// with the configured delimiter and the immediate objects (S3 Contents, not the
// CommonPrefixes) are counted — matching the Azure Blob scaler's semantics.
func (s *awsS3Scaler) getS3ObjectCount(ctx context.Context) (int64, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.metadata.BucketName),
	}
	if s.metadata.Prefix != "" {
		input.Prefix = aws.String(s.metadata.Prefix)
	}
	if s.metadata.MaxKeys > 0 {
		input.MaxKeys = aws.Int32(s.metadata.MaxKeys)
	}

	if s.metadata.globPattern != nil {
		return s.countWithGlob(ctx, input)
	}

	if s.metadata.Delimiter != "" {
		input.Delimiter = aws.String(s.metadata.Delimiter)
	}

	var count int64
	paginator := s3.NewListObjectsV2Paginator(s.s3WrapperClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return -1, err
		}
		count += int64(len(page.Contents))
	}
	return count, nil
}

func (s *awsS3Scaler) countWithGlob(ctx context.Context, input *s3.ListObjectsV2Input) (int64, error) {
	globPattern := *s.metadata.globPattern
	var count int64
	// Glob matching needs the full key set, so we list recursively (no delimiter).
	input.Delimiter = nil
	paginator := s3.NewListObjectsV2Paginator(s.s3WrapperClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return -1, err
		}
		for _, object := range page.Contents {
			if object.Key != nil && globPattern.Match(*object.Key) {
				count++
			}
		}
	}
	return count, nil
}

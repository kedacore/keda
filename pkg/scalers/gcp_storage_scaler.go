package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/go-logr/logr"
	"google.golang.org/api/iterator"
	option "google.golang.org/api/option"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// Default for how many objects per a single scaled processor
	defaultTargetObjectCount = 100
	// A limit on iterating bucket objects
	defaultMaxBucketItemsToScan = 1000
)

type gcsScaler struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	metricType v2.MetricTargetType
	metadata   *gcsMetadata
	logger     logr.Logger
}

type gcsMetadata struct {
	bucketName                  string
	gcpAuthorization            *gcp.AuthorizationMetadata
	maxBucketItemsToScan        int64
	metricName                  string
	targetObjectCount           int64
	activationTargetObjectCount int64
	blobDelimiter               string
	blobPrefix                  string
}

// NewGcsScaler creates a new gcsScaler
func NewGcsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_storage_scaler")

	meta, err := parseGcsMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing GCP storage metadata: %w", err)
	}

	ctx := context.Background()

	var client *storage.Client

	switch {
	case meta.gcpAuthorization.PodIdentityProviderEnabled:
		client, err = storage.NewClient(ctx)
	case meta.gcpAuthorization.GoogleApplicationCredentialsFile != "":
		client, err = storage.NewClient(
			ctx, option.WithCredentialsFile(meta.gcpAuthorization.GoogleApplicationCredentialsFile))
	default:
		client, err = storage.NewClient(
			ctx, option.WithCredentialsJSON([]byte(meta.gcpAuthorization.GoogleApplicationCredentials)))
	}

	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	bucket := client.Bucket(meta.bucketName)
	if bucket == nil {
		return nil, fmt.Errorf("failed to create a handle to bucket %s", meta.bucketName)
	}

	logger.Info(fmt.Sprintf("Metadata %v", meta))

	return &gcsScaler{
		client:     client,
		bucket:     bucket,
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseGcsMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*gcsMetadata, error) {
	meta := gcsMetadata{}
	meta.targetObjectCount = defaultTargetObjectCount
	meta.maxBucketItemsToScan = defaultMaxBucketItemsToScan

	if val, ok := config.TriggerMetadata["bucketName"]; ok {
		if val == "" {
			logger.Error(nil, "no bucket name given")
			return nil, fmt.Errorf("no bucket name given")
		}

		meta.bucketName = val
	} else {
		logger.Error(nil, "no bucket name given")
		return nil, fmt.Errorf("no bucket name given")
	}

	if val, ok := config.TriggerMetadata["targetObjectCount"]; ok {
		targetObjectCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing targetObjectCount")
			return nil, fmt.Errorf("error parsing targetObjectCount: %w", err)
		}

		meta.targetObjectCount = targetObjectCount
	}

	meta.activationTargetObjectCount = 0
	if val, ok := config.TriggerMetadata["activationTargetObjectCount"]; ok {
		activationTargetObjectCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetObjectCount parsing error %w", err)
		}
		meta.activationTargetObjectCount = activationTargetObjectCount
	}

	if val, ok := config.TriggerMetadata["maxBucketItemsToScan"]; ok {
		maxBucketItemsToScan, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing maxBucketItemsToScan")
			return nil, fmt.Errorf("error parsing maxBucketItemsToScan: %w", err)
		}

		meta.maxBucketItemsToScan = maxBucketItemsToScan
	}

	if val, ok := config.TriggerMetadata["blobDelimiter"]; ok {
		meta.blobDelimiter = val
	}

	if val, ok := config.TriggerMetadata["blobPrefix"]; ok {
		meta.blobPrefix = val
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	var metricName = kedautil.NormalizeString(fmt.Sprintf("gcp-storage-%s", meta.bucketName))
	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, metricName)

	return &meta, nil
}

func (s *gcsScaler) Close(context.Context) error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *gcsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetObjectCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns the number of items in the bucket (up to s.metadata.maxBucketItemsToScan)
func (s *gcsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	items, err := s.getItemCount(ctx, s.metadata.maxBucketItemsToScan)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(items))

	return []external_metrics.ExternalMetricValue{metric}, items > s.metadata.activationTargetObjectCount, nil
}

// getItemCount gets the number of items in the bucket, up to maxCount
func (s *gcsScaler) getItemCount(ctx context.Context, maxCount int64) (int64, error) {
	query := &storage.Query{Delimiter: s.metadata.blobDelimiter, Prefix: s.metadata.blobPrefix}
	err := query.SetAttrSelection([]string{"Size"})
	if err != nil {
		s.logger.Error(err, "failed to set attribute selection")
		return 0, err
	}

	it := s.bucket.Objects(ctx, query)
	var count int64

	for count < maxCount {
		item, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			if errors.Is(err, storage.ErrBucketNotExist) {
				s.logger.Info("Bucket " + s.metadata.bucketName + " doesn't exist")
				return 0, nil
			}
			s.logger.Error(err, "failed to enumerate items in bucket "+s.metadata.bucketName)
			return count, err
		}
		// The folder is retrieved as an entity, so if size is 0
		// we can skip it
		if item.Size == 0 {
			continue
		}
		count++
	}

	s.logger.V(1).Info(fmt.Sprintf("Counted %d items with a limit of %d", count, maxCount))
	return count, nil
}

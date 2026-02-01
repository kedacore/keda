package scalers

import (
	"context"
	"errors"
	"fmt"

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

type gcsScaler struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	metricType v2.MetricTargetType
	metadata   *gcsMetadata
	logger     logr.Logger
}

type gcsMetadata struct {
	BucketName                  string `keda:"name=bucketName,                  order=triggerMetadata"`
	TargetObjectCount           int64  `keda:"name=targetObjectCount,           order=triggerMetadata, default=100"`
	ActivationTargetObjectCount int64  `keda:"name=activationTargetObjectCount, order=triggerMetadata, default=0"`
	MaxBucketItemsToScan        int64  `keda:"name=maxBucketItemsToScan,        order=triggerMetadata, default=1000"`
	BlobDelimiter               string `keda:"name=blobDelimiter,               order=triggerMetadata, optional"`
	BlobPrefix                  string `keda:"name=blobPrefix,                  order=triggerMetadata, optional"`

	Credentials            string `keda:"name=credentials, order=triggerMetadata;resolvedEnv, optional"`
	CredentialsFromEnvFile string `keda:"name=credentialsFromEnvFile, order=triggerMetadata;resolvedEnv, optional"`

	gcpAuthorization *gcp.AuthorizationMetadata
	metricName       string
	triggerIndex     int
}

func (g *gcsMetadata) Validate() error {
	if g.TargetObjectCount <= 0 {
		return fmt.Errorf("targetObjectCount must be a positive number")
	}
	if g.ActivationTargetObjectCount < 0 {
		return fmt.Errorf("activationTargetObjectCount must be a positive number")
	}
	if g.MaxBucketItemsToScan <= 0 {
		return fmt.Errorf("maxBucketItemsToScan must be a positive number")
	}

	return nil
}

// NewGcsScaler creates a new gcsScaler
func NewGcsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_storage_scaler")

	meta, err := parseGcsMetadata(config)
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
			ctx, option.WithAuthCredentialsFile(option.ServiceAccount, meta.gcpAuthorization.GoogleApplicationCredentialsFile))
	default:
		client, err = storage.NewClient(
			ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(meta.gcpAuthorization.GoogleApplicationCredentials)))
	}

	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	bucket := client.Bucket(meta.BucketName)
	if bucket == nil {
		return nil, fmt.Errorf("failed to create a handle to bucket %s", meta.BucketName)
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

func parseGcsMetadata(config *scalersconfig.ScalerConfig) (*gcsMetadata, error) {
	meta := &gcsMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing gcs metadata: %w", err)
	}

	if err := meta.Validate(); err != nil {
		return nil, err
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	var metricName = kedautil.NormalizeString(fmt.Sprintf("gcp-storage-%s", meta.BucketName))
	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, metricName)

	return meta, nil
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
		Target: GetMetricTarget(s.metricType, s.metadata.TargetObjectCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns the number of items in the bucket (up to s.metadata.MaxBucketItemsToScan)
func (s *gcsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	items, err := s.getItemCount(ctx, s.metadata.MaxBucketItemsToScan)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(items))

	return []external_metrics.ExternalMetricValue{metric}, items > s.metadata.ActivationTargetObjectCount, nil
}

// getItemCount gets the number of items in the bucket, up to maxCount
func (s *gcsScaler) getItemCount(ctx context.Context, maxCount int64) (int64, error) {
	query := &storage.Query{Delimiter: s.metadata.BlobDelimiter, Prefix: s.metadata.BlobPrefix}
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
				s.logger.Info("Bucket " + s.metadata.BucketName + " doesn't exist")
				return 0, nil
			}
			s.logger.Error(err, "failed to enumerate items in bucket "+s.metadata.BucketName)
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

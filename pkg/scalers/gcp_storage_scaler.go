package scalers

import (
	"context"
	"fmt"
	"strconv"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	option "google.golang.org/api/option"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetLength = 1000
)

type gcsScaler struct {
	client   *storage.Client
	bucket   *storage.BucketHandle
	metadata *gcsMetadata
}

type gcsMetadata struct {
	bucketName       string
	gcpAuthorization *gcpAuthorizationMetadata
	targetLength     int
	scalerIndex      int
}

var gcsLog = logf.Log.WithName("gcp_storage_scaler")

// NewGcsScaler creates a new gcsScaler
func NewGcsScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseGcsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing GCP storage metadata: %s", err)
	}

	ctx := context.Background()

	var client *storage.Client

	if meta.gcpAuthorization.podIdentityProviderEnabled {
		client, err = storage.NewClient(ctx, option.WithScopes("ScopeReadOnly"))
	} else if meta.gcpAuthorization.GoogleApplicationCredentialsFile != "" {
		client, err = storage.NewClient(
			ctx, option.WithCredentialsFile(meta.gcpAuthorization.GoogleApplicationCredentialsFile))
	} else {
		client, err = storage.NewClient(
			ctx, option.WithCredentialsJSON([]byte(meta.gcpAuthorization.GoogleApplicationCredentials)))
	}

	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}

	bucket := client.Bucket(meta.bucketName)
	if bucket == nil {
		return nil, fmt.Errorf("Failed to create a handle to bucket %s", meta.bucketName)
	}

	return &gcsScaler{
		client:   client,
		bucket:   bucket,
		metadata: meta,
	}, nil
}

func parseGcsMetadata(config *ScalerConfig) (*gcsMetadata, error) {
	meta := gcsMetadata{}
	meta.targetLength = defaultTargetLength

	if val, ok := config.TriggerMetadata["bucketName"]; ok {
		if val == "" {
			gcsLog.Error(nil, "no bucket name given")
			return nil, fmt.Errorf("no bucket name given")
		}

		meta.bucketName = val
	} else {
		gcsLog.Error(nil, "no bucket name given")
		return nil, fmt.Errorf("no bucket name given")
	}

	if val, ok := config.TriggerMetadata["targetLength"]; ok {
		targetLength, err := strconv.Atoi(val)
		if err != nil {
			gcsLog.Error(err, "Error parsing targetLength")
			return nil, fmt.Errorf("error parsing targetLength: %s", err.Error())
		}

		meta.targetLength = targetLength
	}

	auth, err := getGcpAuthorization(config, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *gcsScaler) IsActive(ctx context.Context) (bool, error) {
	items, err := s.getItemCount(ctx, 1)
	if err != nil {
		return false, err
	}

	return items > 0, nil
}

func (s *gcsScaler) Close(context.Context) error {
	if s.client != nil {
		s.client.Close()
	}
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *gcsScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	// Construct the target value as a quantity
	targetValueQty := resource.NewQuantity(int64(s.metadata.targetLength), resource.DecimalSI)

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-storage-%s", s.metadata.bucketName))),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetValueQty,
		},
	}

	// Create the metric spec for the HPA
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics connects to Stack Driver and finds the size of the pub sub subscription
func (s *gcsScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	items, err := s.getItemCount(ctx, s.metadata.targetLength)
	if err != nil {
		return nil, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(items), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// getItemCount gets the number of items in the bucket, up to maxCount
func (s *gcsScaler) getItemCount(ctx context.Context, maxCount int) (int, error) {
	query := &storage.Query{Prefix: ""}
	query.SetAttrSelection([]string{"Name"})
	it := s.bucket.Objects(ctx, query)
	count := 0

	for count < maxCount {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			gcsLog.Error(err, "Failed to enumerate items in bucket")
			return count, err
		}
		count++
	}

	return count, nil
}

package scalers

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/go-logr/logr"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const spannerMetricNamePrefix = "gcp-spanner"

type spannerScaler struct {
	client     *spanner.Client
	metricType v2.MetricTargetType
	metadata   *spannerMetadata
	logger     logr.Logger
}

type spannerMetadata struct {
	ProjectID       string  `keda:"name=projectId,              order=triggerMetadata"`
	InstanceID      string  `keda:"name=instanceId,             order=triggerMetadata"`
	DatabaseID      string  `keda:"name=databaseId,             order=triggerMetadata"`
	Query           string  `keda:"name=query,                  order=triggerMetadata"`
	TargetValue     float64 `keda:"name=targetValue,            order=triggerMetadata, default=5"`
	ActivationValue float64 `keda:"name=activationValue,        order=triggerMetadata, default=0"`

	Credentials            string `keda:"name=credentials,            order=triggerMetadata;resolvedEnv, optional"`
	CredentialsFromEnvFile string `keda:"name=credentialsFromEnvFile, order=triggerMetadata;resolvedEnv, optional"`

	gcpAuthorization *gcp.AuthorizationMetadata
	metricName       string
	triggerIndex     int
}

func (m *spannerMetadata) Validate() error {
	if m.TargetValue <= 0 {
		return fmt.Errorf("targetValue must be greater than 0, got %v", m.TargetValue)
	}
	return nil
}

// NewGcpSpannerScaler creates a new spannerScaler.
func NewGcpSpannerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_spanner_scaler")

	meta, err := parseSpannerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Spanner metadata: %w", err)
	}

	client, err := newSpannerClient(context.Background(), meta)
	if err != nil {
		return nil, fmt.Errorf("error creating Spanner client: %w", err)
	}

	return &spannerScaler{
		client:     client,
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseSpannerMetadata(config *scalersconfig.ScalerConfig) (*spannerMetadata, error) {
	meta := &spannerMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, err
	}

	meta.metricName = GenerateMetricNameWithIndex(
		config.TriggerIndex,
		kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s",
			spannerMetricNamePrefix,
			meta.InstanceID,
			meta.DatabaseID,
			meta.ProjectID,
		)),
	)

	// When SPANNER_EMULATOR_HOST is set the SDK handles auth automatically;
	// credentials are not required in that case.
	if os.Getenv("SPANNER_EMULATOR_HOST") != "" {
		meta.gcpAuthorization = &gcp.AuthorizationMetadata{}
		return meta, nil
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	return meta, nil
}

func newSpannerClient(ctx context.Context, meta *spannerMetadata) (*spanner.Client, error) {
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		meta.ProjectID, meta.InstanceID, meta.DatabaseID)

	var opts []option.ClientOption

	switch {
	case meta.gcpAuthorization.PodIdentityProviderEnabled:
		// Workload Identity / ADC — the SDK picks up credentials from the
		// metadata server automatically; no explicit option is needed.
	case meta.gcpAuthorization.GoogleApplicationCredentials != "":
		opts = append(opts, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(meta.gcpAuthorization.GoogleApplicationCredentials)))
	case meta.gcpAuthorization.GoogleApplicationCredentialsFile != "":
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, meta.gcpAuthorization.GoogleApplicationCredentialsFile))
	default:
		// No explicit credentials and no pod identity — the SDK will attempt
		// Application Default Credentials (ADC) from the environment
		// (GOOGLE_APPLICATION_CREDENTIALS, gcloud default, metadata server).
		// GetGCPAuthorization already rejected this path in production; we reach
		// here only when SPANNER_EMULATOR_HOST bypasses auth validation.
	}

	return spanner.NewClient(ctx, db, opts...)
}

func (s *spannerScaler) Close(_ context.Context) error {
	if s.client != nil {
		s.client.Close()
	}
	return nil
}

// getQueryResult executes the configured SQL and returns the value of the first
// column of the first row as int64.  A query that matches no rows is treated as
// a value of 0 (not an error) so that users can write queries like
// "SELECT COUNT(*) FROM jobs WHERE status = 'pending'" — when the table is
// empty the result is a single row with value 0, but arbitrary single-value
// SELECTs that may return no rows are also handled gracefully.
func (s *spannerScaler) getQueryResult(ctx context.Context) (int64, error) {
	iter := s.client.Single().Query(ctx, spanner.Statement{SQL: s.metadata.Query})
	defer iter.Stop()

	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		// No rows returned. COUNT(*) never produces an empty result set, so
		// reaching here means the query is a single-value SELECT that matched
		// nothing. Treat as 0 so KEDA scales to zero cleanly rather than
		// surfacing a spurious error.
		s.logger.V(1).Info("query returned no rows, treating value as 0",
			"query", s.metadata.Query)
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("querying Spanner: %w", err)
	}

	var count int64
	if err := row.Column(0, &count); err != nil {
		return 0, fmt.Errorf("reading first column as INT64: %w", err)
	}
	return count, nil
}

func (s *spannerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	count, err := s.getQueryResult(ctx)
	if err != nil {
		s.logger.Error(err, "error querying Spanner")
		return nil, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(count))

	return []external_metrics.ExternalMetricValue{metric}, float64(count) > s.metadata.ActivationValue, nil
}

func (s *spannerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{Name: s.metadata.metricName},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	return []v2.MetricSpec{{External: externalMetric, Type: externalMetricType}}
}

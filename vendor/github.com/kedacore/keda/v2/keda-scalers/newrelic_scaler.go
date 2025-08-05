package scalers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	scalerName = "new-relic"
)

type newrelicScaler struct {
	metricType v2.MetricTargetType
	metadata   newrelicMetadata
	nrClient   *newrelic.NewRelic
	logger     logr.Logger
}

type newrelicMetadata struct {
	Account             int     `keda:"name=account,             order=authParams;triggerMetadata"`
	Region              string  `keda:"name=region,              order=authParams;triggerMetadata, default=US"`
	QueryKey            string  `keda:"name=queryKey,            order=authParams;triggerMetadata"`
	NoDataError         bool    `keda:"name=noDataError,         order=triggerMetadata, default=false"`
	NRQL                string  `keda:"name=nrql,                order=triggerMetadata"`
	Threshold           float64 `keda:"name=threshold,           order=triggerMetadata"`
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	TriggerIndex        int
}

func NewNewRelicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, fmt.Sprintf("%s_scaler", scalerName))

	meta, err := parseNewRelicMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", scalerName, err)
	}

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey(meta.QueryKey),
		newrelic.ConfigRegion(meta.Region))
	if err != nil {
		return nil, fmt.Errorf("error initializing client: %w", err)
	}

	logger.Info(fmt.Sprintf("Initializing New Relic Scaler (account %d in region %s)", meta.Account, meta.Region))

	return &newrelicScaler{
		metricType: metricType,
		metadata:   meta,
		nrClient:   nrClient,
		logger:     logger,
	}, nil
}

func parseNewRelicMetadata(config *scalersconfig.ScalerConfig) (newrelicMetadata, error) {
	meta := newrelicMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing newrelic metadata: %w", err)
	}

	if config.AsMetricSource {
		meta.Threshold = 0
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func (s *newrelicScaler) Close(context.Context) error {
	return nil
}

func (s *newrelicScaler) executeNewRelicQuery(ctx context.Context) (float64, error) {
	nrdbQuery := nrdb.NRQL(s.metadata.NRQL)
	resp, err := s.nrClient.Nrdb.QueryWithContext(ctx, s.metadata.Account, nrdbQuery)
	if err != nil {
		return 0, fmt.Errorf("error running NRQL %s: %w", s.metadata.NRQL, err)
	}

	if len(resp.Results) == 0 {
		if s.metadata.NoDataError {
			return 0, fmt.Errorf("query returned no results: %s", s.metadata.NRQL)
		}
		return 0, nil
	}
	// Only use the first result from the query, as the query should not be multi row
	for _, v := range resp.Results[0] {
		if val, ok := v.(float64); ok {
			return val, nil
		}
	}

	if s.metadata.NoDataError {
		return 0, fmt.Errorf("query returned no numeric results: %s", s.metadata.NRQL)
	}
	return 0, nil
}

func (s *newrelicScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeNewRelicQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing NRQL query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

func (s *newrelicScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(scalerName)
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

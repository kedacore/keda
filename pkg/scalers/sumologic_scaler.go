package scalers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/sumologic"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	multiMetricsQueryPrefix = "query."
)

type sumologicScaler struct {
	client     *sumologic.Client
	metricType v2.MetricTargetType
	metadata   *sumologicMetadata
	logger     logr.Logger
}

type sumologicMetadata struct {
	AccessID            string            `keda:"name=accessID,            order=authParams"`
	AccessKey           string            `keda:"name=accessKey,           order=authParams"`
	Host                string            `keda:"name=host,                order=triggerMetadata"`
	UnsafeSsl           bool              `keda:"name=unsafeSsl,           order=triggerMetadata, optional"`
	QueryType           string            `keda:"name=queryType,           order=triggerMetadata, enum=logs;metrics"`
	Query               string            `keda:"name=query,               order=triggerMetadata, optional"`
	Queries             map[string]string `keda:"name=query.*,             order=triggerMetadata, optional"`                                // Only for metrics queries
	ResultQueryRowID    string            `keda:"name=resultQueryRowID,    order=triggerMetadata, optional"`                                // Only for metrics queries
	Quantization        time.Duration     `keda:"name=quantization,        order=triggerMetadata, optional"`                                // Only for metrics queries
	Rollup              string            `keda:"name=rollup,              order=triggerMetadata, enum=Avg;Sum;Count;Min;Max, default=Avg"` // Only for metrics queries
	ResultField         string            `keda:"name=resultField,         order=triggerMetadata, optional"`                                // Only for logs queries
	Timerange           time.Duration     `keda:"name=timerange,           order=triggerMetadata"`
	Timezone            string            `keda:"name=timezone,            order=triggerMetadata, default=UTC"`
	ActivationThreshold float64           `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	QueryAggregator     string            `keda:"name=queryAggregator,     order=triggerMetadata, enum=Latest;Avg;Sum;Count;Min;Max, default=Avg"`
	Threshold           float64           `keda:"name=threshold,           order=triggerMetadata"`
	TriggerIndex        int
}

func NewSumologicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "sumologic_scaler")
	meta, err := parseSumoMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	client, err := sumologic.NewClient(&sumologic.Config{
		Host:      meta.Host,
		AccessID:  meta.AccessID,
		AccessKey: meta.AccessKey,
		UnsafeSsl: meta.UnsafeSsl,
	}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sumologic client: %w", err)
	}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	return &sumologicScaler{
		client:     client,
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseSumoMetadata(config *scalersconfig.ScalerConfig) (*sumologicMetadata, error) {
	meta := sumologicMetadata{}
	if err := config.TypedConfig(&meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}
	meta.TriggerIndex = config.TriggerIndex

	queries, err := parseMultiMetricsQueries(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	meta.Queries = queries

	if meta.QueryType == "logs" {
		if meta.Query == "" {
			return nil, errors.New("missing required metadata: query")
		}
		if len(meta.Queries) != 0 {
			return nil, errors.New("invalid metadata, query.<RowId> not supported for logs queryType")
		}
		if meta.ResultField == "" {
			return nil, errors.New("missing required metadata: resultField (required for logs queryType)")
		}
	}

	if meta.QueryType == "metrics" {
		if meta.Query == "" && len(meta.Queries) == 0 {
			return nil, errors.New("missing metadata: either of query or query.<RowId> must be defined for metrics queryType")
		}
		if meta.Query != "" && len(meta.Queries) != 0 {
			return nil, errors.New("invalid metadata: only one of query or query.<RowId> must be defined for metrics queryType")
		}
		if len(meta.Queries) > 0 {
			if meta.ResultQueryRowID == "" {
				return nil, errors.New("missing required metadata: resultQueryRowID for multi-metrics query")
			}
			if _, ok := meta.Queries[meta.ResultQueryRowID]; !ok {
				return nil, fmt.Errorf("resultQueryRowID '%s' not found in queries", meta.ResultQueryRowID)
			}
		}

		if meta.Quantization == 0 {
			return nil, errors.New("missing required metadata: quantization for metrics queryType")
		}
	}

	return &meta, nil
}

func parseMultiMetricsQueries(triggerMetadata map[string]string) (map[string]string, error) {
	queries := make(map[string]string)
	for key, value := range triggerMetadata {
		if strings.HasPrefix(key, multiMetricsQueryPrefix) {
			rowID := strings.TrimPrefix(key, multiMetricsQueryPrefix)
			if rowID == "" {
				return nil, fmt.Errorf("malformed metadata, unable to parse rowID from key: %s", key)
			}
			if value == "" {
				return nil, fmt.Errorf("malformed metadata, invalid value for key: %s", key)
			}
			queries[rowID] = value
		}
	}
	return queries, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("sumologic-%s", s.metadata.QueryType))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	return []v2.MetricSpec{{
		External: externalMetric,
		Type:     externalMetricType,
	}}
}

// No need to close connections manually, but we can close idle HTTP connections
func (s *sumologicScaler) Close(_ context.Context) error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *sumologicScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.executeQuery()
	if err != nil {
		s.logger.Error(err, "error getting metrics from sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from sumologic: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.ActivationThreshold

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func (s *sumologicScaler) executeQuery() (float64, error) {
	return s.client.GetQueryResult(
		s.metadata.QueryType,
		s.metadata.Query,
		s.metadata.Queries,
		s.metadata.ResultQueryRowID,
		s.metadata.Quantization,
		s.metadata.Rollup,
		s.metadata.ResultField,
		s.metadata.Timerange,
		s.metadata.Timezone,
		s.metadata.QueryAggregator,
	)
}

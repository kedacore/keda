package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	defaultTimezone         = "UTC"
	multiMetricsQueryPrefix = "query."
)

type sumologicScaler struct {
	client     *sumologic.Client
	metricType v2.MetricTargetType
	metadata   *sumologicMetadata
	logger     logr.Logger
}

type sumologicMetadata struct {
	accessID            string            `keda:"name=access_id,           order=authParams"`
	accessKey           string            `keda:"name=access_key,          order=authParams"`
	host                string            `keda:"name=host,                order=triggerMetadata"`
	unsafeSsl           bool              `keda:"name=unsafeSsl,           order=triggerMetadata, optional"`
	queryType           string            `keda:"name=queryType,           order=triggerMetadata"`
	query               string            `keda:"name=query,               order=triggerMetadata"`
	queries             map[string]string `keda:"name=query.*,             order=triggerMetadata"`           // Only for metrics queries
	resultQueryRowID    string            `keda:"name=resultQueryRowID,    order=triggerMetadata"`           // Only for metrics queries
	quantization        time.Duration     `keda:"name=quantization,        order=triggerMetadata"`           // Only for metrics queries
	rollup              string            `keda:"name=rollup,              order=triggerMetadata, optional"` // Only for metrics queries
	resultField         string            `keda:"name=resultField,         order=triggerMetadata"`           // Only for logs queries
	timerange           time.Duration     `keda:"name=timerange,           order=triggerMetadata"`
	timezone            string            `keda:"name=timezone,            order=triggerMetadata, optional"`
	activationThreshold float64           `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	queryAggregator     string            `keda:"name=queryAggregator,     order=triggerMetadata, optional"`
	threshold           float64           `keda:"name=threshold,           order=triggerMetadata"`
	triggerIndex        int
}

func NewSumologicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "sumologic_scaler")
	meta, err := parseSumoMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	client, err := sumologic.NewClient(&sumologic.Config{
		Host:      meta.host,
		AccessID:  meta.accessID,
		AccessKey: meta.accessKey,
		UnsafeSsl: meta.unsafeSsl,
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
	meta.triggerIndex = config.TriggerIndex

	if err := validateAuthParams(config.AuthParams); err != nil {
		return nil, err
	}
	meta.accessID = config.AuthParams["accessID"]
	meta.accessKey = config.AuthParams["accessKey"]

	if err := validateMetadata(config.TriggerMetadata); err != nil {
		return nil, err
	}
	meta.host = config.TriggerMetadata["host"]
	meta.queryType = config.TriggerMetadata["queryType"]

	query := config.TriggerMetadata["query"]
	queries, err := parseMultiMetricsQueries(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}

	if meta.queryType == "logs" {
		if query == "" {
			return nil, errors.New("missing required metadata: query")
		}
		if len(queries) != 0 {
			return nil, errors.New("invalid metadata, query.<RowId> not supported for logs queryType")
		}
		meta.query = query

		if resultField, exists := config.TriggerMetadata["resultField"]; !exists || resultField == "" {
			return nil, fmt.Errorf("resultField is required when queryType is 'logs'")
		}
		meta.resultField = config.TriggerMetadata["resultField"]
	}

	if meta.queryType == "metrics" {
		switch {
		case query == "" && len(queries) == 0:
			return nil, errors.New("missing metadata, please define either of query or query.<RowId> for metrics queryType")
		case query != "" && len(queries) != 0:
			return nil, errors.New("incorrect metadata, please only define either query or query.<RowId> for metrics queryType, not both")
		case query != "":
			meta.query = query
		default:
			meta.queries = queries
			if config.TriggerMetadata["resultQueryRowID"] == "" {
				return nil, errors.New("missing required metadata: resultQueryRowID")
			}
			meta.resultQueryRowID = config.TriggerMetadata["resultQueryRowID"]
		}

		if config.TriggerMetadata["quantization"] == "" {
			return nil, errors.New("missing required metadata: quantization for metrics queryType")
		}
		quantization, err := strconv.Atoi(config.TriggerMetadata["quantization"])
		if err != nil {
			return nil, fmt.Errorf("invalid metadata, quantization: %w", err)
		}
		meta.quantization = time.Duration(quantization) * time.Second

		if rollup, exists := config.TriggerMetadata["rollup"]; exists {
			if err := sumologic.IsValidRollupType(rollup); err != nil {
				return nil, err
			}
			meta.rollup = rollup
		} else {
			meta.rollup = sumologic.DefaultRollup
		}
	}

	if config.TriggerMetadata["timerange"] == "" {
		return nil, errors.New("missing required metadata: timerange")
	}
	timerange, err := strconv.Atoi(config.TriggerMetadata["timerange"])
	if err != nil {
		return nil, fmt.Errorf("invalid timerange: %w", err)
	}
	meta.timerange = time.Duration(timerange)

	if config.TriggerMetadata["timezone"] == "" {
		meta.timezone = defaultTimezone // Default to UTC if not provided
	} else {
		meta.timezone = config.TriggerMetadata["timezone"]
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThreshold parsing error: %w", err)
		}
		meta.activationThreshold = activationThreshold
	}

	if queryAggregator, ok := config.TriggerMetadata["queryAggregator"]; ok && queryAggregator != "" {
		if err := sumologic.IsValidQueryAggregation(queryAggregator); err != nil {
			return nil, err
		}
		meta.queryAggregator = queryAggregator
	} else {
		meta.queryAggregator = sumologic.DefaultQueryAggregator
	}

	if val, ok := config.TriggerMetadata["threshold"]; ok {
		threshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("threshold parsing error: %w", err)
		}
		meta.threshold = threshold
	}

	return &meta, nil
}

func validateAuthParams(authParams map[string]string) error {
	if authParams["accessID"] == "" {
		return errors.New("missing required auth params: accessID")
	}
	if authParams["accessKey"] == "" {
		return errors.New("missing required auth params: accessKey")
	}
	return nil
}

func validateMetadata(metadata map[string]string) error {
	if metadata["host"] == "" {
		return errors.New("missing required metadata: host")
	}
	if metadata["queryType"] == "" {
		return errors.New("missing required metadata: queryType")
	}
	if metadata["queryType"] != "logs" && metadata["queryType"] != "metrics" {
		return fmt.Errorf("invalid queryType: %s, must be 'logs' or 'metrics'", metadata["queryType"])
	}
	return nil
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
	metricName := kedautil.NormalizeString(fmt.Sprintf("sumologic-%s", s.metadata.queryType))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
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
	var metric external_metrics.ExternalMetricValue
	var num float64
	var err error

	num, err = s.client.GetQueryResult(
		s.metadata.queryType,
		s.metadata.query,
		s.metadata.queries,
		s.metadata.resultQueryRowID,
		s.metadata.quantization,
		s.metadata.rollup,
		s.metadata.resultField,
		s.metadata.timerange,
		s.metadata.timezone,
		s.metadata.queryAggregator,
	)
	if err != nil {
		s.logger.Error(err, "error getting metrics from sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.activationThreshold

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

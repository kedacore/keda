package scalers

import (
	"context"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/sumologic"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type sumologicScaler struct {
	client   *sumologic.Client
	metadata *sumoMetadata
}

type sumologicMetadata struct {
	AccessID             string `keda:"name=access_id,        order=authParams"`
	AccessKey            string `keda:"name=access_key,        order=authParams"`
	Host                 string `keda:"name=host,            order=triggerMetadata"`
	UnsafeSsl            bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	Query                string
	QueryType            string
	Dimension            string
	Quantization         time.Duration // Only for metrics queries
	Timerange            time.Duration
	Timezone             string
	activationQueryValue float64
	queryAggegrator      string
	vType                v2.MetricTargetType
}

const maxString = "max"
const avgString = "average"
const logsQueryType = "logs"
const metricsQueryType = "metrics"

func NewSumoScaler(config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
	meta, err := parseSumoMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	client, err := NewClient(&Config{
		Host:      meta.Host,
		AccessID:  meta.AccessID,
		AccessKey: meta.AccessKey,
	}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sumologic client: %w", err)
	}

	return &sumoScaler{
		metadata: meta,
		client:   client,
	}, nil
}

func parseSumoMetadata(config *scalersconfig.ScalerConfig) (*sumoMetadata, error) {
	meta := sumoMetadata{}

	if config.TriggerMetadata["host"] == "" {
		return nil, errors.New("missing required metadata: host")
	}
	meta.Host = config.TriggerMetadata["host"]

	if config.TriggerMetadata["accessID"] == "" {
		return nil, errors.New("missing required metadata: accessID")
	}
	meta.AccessID = config.TriggerMetadata["accessID"]

	if config.TriggerMetadata["accessKey"] == "" {
		return nil, errors.New("missing required metadata: accessKey")
	}
	meta.AccessKey = config.TriggerMetadata["accessKey"]

	if config.TriggerMetadata["query"] == "" {
		return nil, errors.New("missing required metadata: query")
	}
	meta.Query = config.TriggerMetadata["query"]

	if config.TriggerMetadata["queryType"] == "" {
		return nil, errors.New("missing required metadata: type (must be 'logs' or 'metrics')")
	}
	meta.Type = config.TriggerMetadata["queryType"]
	if meta.Type != "logs" && meta.Type != "metrics" {
		return nil, fmt.Errorf("invalid type: %s, must be '%s' or '%s'", meta.Type, logsQueryType, metricsQueryType)
	}

	if config.TriggerMetadata["timerange"] == "" {
		return nil, errors.New("missing required metadata: timerange")
	}
	timerange, err := strconv.Atoi(config.TriggerMetadata["timerange"])
	if err != nil {
		return nil, fmt.Errorf("invalid timerange: %w", err)
	}
	meta.Timerange = time.Duration(timerange) * time.Minute

	if config.TriggerMetadata["dimension"] == "" {
		return nil, errors.New("missing required metadata: dimension")
	}
	meta.Dimension = config.TriggerMetadata["dimension"]

	if config.TriggerMetadata["timezone"] == "" {
		meta.Timezone = "UTC" // Default to UTC if not provided
	} else {
		meta.Timezone = config.TriggerMetadata["timezone"]
	}

	if meta.Type == metricsQueryType {
		if config.TriggerMetadata["quantization"] == "" {
			return nil, errors.New("missing required metadata: quantization (only for metrics queries)")
		}
		quantization, err := strconv.Atoi(config.TriggerMetadata["quantization"])
		if err != nil {
			return nil, fmt.Errorf("invalid quantization: %w", err)
		}
		meta.Quantization = time.Duration(quantization) * time.Minute
	}

	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return nil, fmt.Errorf("type has to be 'global' or 'average'")
		}
	} else {
		// Default to using config.MetricType
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return nil, fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	if val, ok := config.TriggerMetadata["metricUnavailableValue"]; ok {
		fillValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("metricUnavailableValue parsing error %w", err)
		}
		meta.fillValue = fillValue
		meta.useFiller = true
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case avgString, maxString:
			meta.queryAggegrator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of '%s, %s'", queryAggregator, avgString, maxString)
		}
	} else {
		meta.queryAggegrator = ""
	}

	return &meta, nil
}

func (s *sumoScaler) GetMetric(ctx context.Context) (float64, error) {
	var result *float64
	var err error

	if s.metadata.QueryType == logsQueryType {
		result, err = s.client.GetLogSearchResult(
			s.metadata.Query,
			s.metadata.Timerange,
			s.metadata.Dimension,
			s.metadata.Timezone,
		)
	} else {
		result, err = s.client.GetMetricsSearchResult(
			s.metadata.Query,
			s.metadata.Quantization,
			s.metadata.Timerange,
			s.metadata.Dimension,
			s.metadata.Timezone,
		)
	}

	if err != nil {
		return 0, err
	}

	return *result, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: fmt.Sprintf("sumologic-%s", s.metadata.Type),
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.targetValue),
	}
	return []v2.MetricSpec{{
		External: externalMetric,
		Type:     externalMetricType,
	}}
}

// No need to close connections manually, but we can close idle HTTP connections
func (s *sumologicScaler) Close(context.Context) error {
	if s.client != nil && s.client.httpClient != nil {
		s.client.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *sumoScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metric external_metrics.ExternalMetricValue
	var num float64
	var err error

	num, err = s.GetMetric(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metrics from Sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.activationQueryValue

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

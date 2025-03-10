package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/sumologic"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type sumologicScaler struct {
	client   *sumologic.Client
	metadata *sumologicMetadata
	logger   logr.Logger
}

type sumologicMetadata struct {
	AccessID             string `keda:"name=access_id,       order=authParams"`
	AccessKey            string `keda:"name=access_key,      order=authParams"`
	Host                 string `keda:"name=host,            order=authParams"`
	UnsafeSsl            bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	Query                string
	QueryType            string
	Dimension            string
	Quantization         time.Duration // Only for metrics queries
	Timerange            time.Duration
	Timezone             string
	activationQueryValue float64
	queryAggegrator      string
	fillValue            float64
	targetValue          float64
	vType                v2.MetricTargetType
}

const max = "max"
const avg = "average"
const logsQueryType = "logs"
const metricsQueryType = "metrics"

func NewSumoScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "sumologic_scaler")
	meta, err := parseSumoMetadata(config, logger)
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

	return &sumologicScaler{
		client:   client,
		metadata: meta,
		logger:   logger,
	}, nil
}

func parseSumoMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*sumologicMetadata, error) {
	meta := sumologicMetadata{}

	if config.AuthParams["host"] == "" {
		return nil, errors.New("missing required metadata: host")
	}
	meta.Host = config.TriggerMetadata["host"]

	if config.AuthParams["accessID"] == "" {
		return nil, errors.New("missing required metadata: accessID")
	}
	meta.AccessID = config.TriggerMetadata["accessID"]

	if config.AuthParams["accessKey"] == "" {
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
	meta.QueryType = config.TriggerMetadata["queryType"]
	if meta.QueryType != "logs" && meta.QueryType != "metrics" {
		return nil, fmt.Errorf("invalid type: %s, must be '%s' or '%s'", meta.QueryType, logsQueryType, metricsQueryType)
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

	if meta.QueryType == metricsQueryType {
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
		case avg:
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
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case avg, max:
			meta.queryAggegrator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of '%s, %s'", queryAggregator, avg, max)
		}
	} else {
		meta.queryAggegrator = ""
	}

	return &meta, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: fmt.Sprintf("sumologic-%s", s.metadata.QueryType),
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.targetValue),
	}
	return []v2.MetricSpec{{
		External: externalMetric,
		Type:     externalMetricType,
	}}
}

// No need to close connections manually, but we can close idle HTTP connections
func (s *sumologicScaler) Close(ctx context.Context) error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *sumologicScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metric external_metrics.ExternalMetricValue
	var num float64
	var err error

	num, err = s.client.GetQueryResult(s.metadata.QueryType, s.metadata.Query, s.metadata.Quantization, s.metadata.Timerange, s.metadata.Dimension, s.metadata.Timezone)
	if err != nil {
		s.logger.Error(err, "error getting metrics from Sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.activationQueryValue

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

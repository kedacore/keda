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
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type sumologicScaler struct {
	client     *sumologic.Client
	metricType v2.MetricTargetType
	metadata   *sumologicMetadata
	logger     logr.Logger
}

type sumologicMetadata struct {
	AccessID             string `keda:"name=access_id,       order=authParams"`
	AccessKey            string `keda:"name=access_key,      order=authParams"`
	Host                 string `keda:"name=host,            order=triggerMetadata"`
	UnsafeSsl            bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	Query                string
	QueryType            string
	Quantization         time.Duration // Only for metrics queries
	Timerange            time.Duration
	Timezone             string
	ActivationQueryValue float64
	QueryAggregator      string
	TargetValue          float64
	TriggerIndex         int
}

const max = "max"
const avg = "average"

func NewSumologicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
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

func parseSumoMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*sumologicMetadata, error) {
	meta := sumologicMetadata{}

	meta.TriggerIndex = config.TriggerIndex

	if config.TriggerMetadata["host"] == "" {
		return nil, errors.New("missing required metadata: host")
	}
	meta.Host = config.TriggerMetadata["host"]

	if config.AuthParams["accessID"] == "" {
		return nil, errors.New("missing required metadata: accessID")
	}
	meta.AccessID = config.AuthParams["accessID"]

	if config.AuthParams["accessKey"] == "" {
		return nil, errors.New("missing required metadata: accessKey")
	}
	meta.AccessKey = config.AuthParams["accessKey"]

	if config.TriggerMetadata["query"] == "" {
		return nil, errors.New("missing required metadata: query")
	}
	meta.Query = config.TriggerMetadata["query"]

	if config.TriggerMetadata["queryType"] == "" {
		return nil, errors.New("missing required metadata: type (must be 'logs' or 'metrics')")
	}
	meta.QueryType = config.TriggerMetadata["queryType"]
	if meta.QueryType != "logs" && meta.QueryType != "metrics" {
		return nil, fmt.Errorf("invalid type: %s, must be 'logs' or 'metrics'", meta.QueryType)
	}

	if config.TriggerMetadata["timerange"] == "" {
		return nil, errors.New("missing required metadata: timerange")
	}
	timerange, err := strconv.Atoi(config.TriggerMetadata["timerange"])
	if err != nil {
		return nil, fmt.Errorf("invalid timerange: %w", err)
	}
	meta.Timerange = time.Duration(timerange)

	if config.TriggerMetadata["timezone"] == "" {
		meta.Timezone = "UTC" // Default to UTC if not provided
	} else {
		meta.Timezone = config.TriggerMetadata["timezone"]
	}

	if meta.QueryType == "metrics" {
		if config.TriggerMetadata["quantization"] == "" {
			return nil, errors.New("missing required metadata: quantization (only for metrics queries)")
		}
		quantization, err := strconv.Atoi(config.TriggerMetadata["quantization"])
		if err != nil {
			return nil, fmt.Errorf("invalid quantization: %w", err)
		}
		meta.Quantization = time.Duration(quantization) * time.Minute
	}

	meta.ActivationQueryValue = 0
	if val, ok := config.TriggerMetadata["ActivationQueryValue"]; ok {
		ActivationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.ActivationQueryValue = ActivationQueryValue
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case avg, max:
			meta.QueryAggregator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of '%s, %s'", queryAggregator, avg, max)
		}
	} else {
		meta.QueryAggregator = ""
	}

	return &meta, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(scalerName)
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
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

	num, err = s.client.GetQueryResult(s.metadata.QueryType, s.metadata.Query, s.metadata.Quantization, s.metadata.Timerange, s.metadata.QueryAggregator, s.metadata.Timezone)
	if err != nil {
		s.logger.Error(err, "error getting metrics from Sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.ActivationQueryValue

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

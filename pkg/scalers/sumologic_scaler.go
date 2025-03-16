package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	accessID            string        `keda:"name=access_id,       order=authParams"`
	accessKey           string        `keda:"name=access_key,      order=authParams"`
	host                string        `keda:"name=host,            order=triggerMetadata"`
	unsafeSsl           bool          `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	query               string        `keda:"name=query,           order=triggerMetadata"`
	queryType           string        `keda:"name=queryType,       order=triggerMetadata"`
	resultField         string        `keda:"name=resultField,     order=triggerMetadata"`           // Only for logs queries
	rollup              string        `keda:"name=rollup,          order=triggerMetadata, optional"` // Only for metrics queries
	quantization        time.Duration `keda:"name=quantization,     order=triggerMetadata"`          // Only for metrics queries
	timerange           time.Duration `keda:"name=timerange,       order=triggerMetadata"`
	timezone            string        `keda:"name=timezone,        order=triggerMetadata, optional"`
	activationThreshold float64       `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	queryAggregator     string        `keda:"name=queryAggregator, order=triggerMetadata, optional"`
	threshold           float64       `keda:"name=threshold,       order=triggerMetadata"`
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

	if config.TriggerMetadata["host"] == "" {
		return nil, errors.New("missing required metadata: host")
	}
	meta.host = config.TriggerMetadata["host"]

	if config.AuthParams["accessID"] == "" {
		return nil, errors.New("missing required metadata: accessID")
	}
	meta.accessID = config.AuthParams["accessID"]

	if config.AuthParams["accessKey"] == "" {
		return nil, errors.New("missing required metadata: accessKey")
	}
	meta.accessKey = config.AuthParams["accessKey"]

	if config.TriggerMetadata["query"] == "" {
		return nil, errors.New("missing required metadata: query")
	}
	meta.query = config.TriggerMetadata["query"]

	if config.TriggerMetadata["queryType"] == "" {
		return nil, errors.New("missing required metadata: type (must be 'logs' or 'metrics')")
	}
	meta.queryType = config.TriggerMetadata["queryType"]

	if meta.queryType != "logs" && meta.queryType != "metrics" {
		return nil, fmt.Errorf("invalid type: %s, must be 'logs' or 'metrics'", meta.queryType)
	}

	if meta.queryType == "logs" {
		if resultField, exists := config.TriggerMetadata["resultField"]; !exists || resultField == "" {
			return nil, fmt.Errorf("resultField is required when queryType is 'logs'")
		}
		meta.resultField = config.TriggerMetadata["resultField"]
	}

	if meta.queryType == "metrics" {
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
		meta.timezone = "UTC" // Default to UTC if not provided
	} else {
		meta.timezone = config.TriggerMetadata["timezone"]
	}

	if meta.queryType == "metrics" {
		if config.TriggerMetadata["quantization"] == "" {
			return nil, errors.New("missing required metadata: quantization (only for metrics queries)")
		}
		quantization, err := strconv.Atoi(config.TriggerMetadata["quantization"])
		if err != nil {
			return nil, fmt.Errorf("invalid quantization: %w", err)
		}
		meta.quantization = time.Duration(quantization) * time.Second
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThreshold parsing error %w", err)
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
			return nil, fmt.Errorf("threshold parsing error %w", err)
		}
		meta.threshold = threshold
	}

	return &meta, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
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

	num, err = s.client.GetQueryResult(s.metadata.queryType, s.metadata.query, s.metadata.quantization, s.metadata.timerange, s.metadata.queryAggregator, s.metadata.timezone, s.metadata.resultField, s.metadata.rollup)
	if err != nil {
		s.logger.Error(err, "error getting metrics from sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.activationThreshold

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

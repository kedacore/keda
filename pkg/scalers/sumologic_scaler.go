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
	accessID            string `keda:"name=access_id,       order=authParams"`
	accessKey           string `keda:"name=access_key,      order=authParams"`
	host                string `keda:"name=host,            order=triggerMetadata"`
	unsafeSsl           bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	query               string
	queryType           string
	quantization        time.Duration // Only for metrics queries
	timerange           time.Duration
	timezone            string
	activationThreshold float64
	queryAggregator     string
	targetThreshold     float64
	triggerIndex        int
}

const (
	defaultQueryAggregator = "average"
)

func NewSumologicScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "sumologic_scaler")
	meta, err := parseSumoMetadata(config, logger)
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

func parseSumoMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*sumologicMetadata, error) {
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
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationThreshold = activationThreshold
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		meta.queryAggregator = queryAggregator
	} else {
		meta.queryAggregator = defaultQueryAggregator
	}

	if val, ok := config.TriggerMetadata["targetThreshold"]; ok {
		targetThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetThreshold parsing error %w", err)
		}
		meta.targetThreshold = targetThreshold
	}

	return &meta, nil
}

func (s *sumologicScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(scalerName)
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetThreshold),
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

	num, err = s.client.GetQueryResult(s.metadata.queryType, s.metadata.query, s.metadata.quantization, s.metadata.timerange, s.metadata.queryAggregator, s.metadata.timezone)
	if err != nil {
		s.logger.Error(err, "error getting metrics from sumologic")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from sumologic: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	isActive := num > s.metadata.activationThreshold

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

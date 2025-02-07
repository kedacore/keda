package scalers

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type splunkObservabilityMetadata struct {
	TriggerIndex int

	AccessToken           string  `keda:"name=accessToken,          order=authParams"`
	Realm                 string  `keda:"name=realm,                order=authParams"`
	Query                 string  `keda:"name=query,                order=triggerMetadata"`
	Duration              int     `keda:"name=duration,             order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue,   	   order=triggerMetadata"`
	QueryAggregator       string  `keda:"name=queryAggregator,      order=triggerMetadata"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue, order=triggerMetadata"`
}

type splunkObservabilityScaler struct {
	metadata  *splunkObservabilityMetadata
	apiClient *signalflow.Client
	logger    logr.Logger
}

func parseSplunkObservabilityMetadata(config *scalersconfig.ScalerConfig) (*splunkObservabilityMetadata, error) {
	meta := &splunkObservabilityMetadata{}
	meta.TriggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing splunk observability metadata: %w", err)
	}

	return meta, nil
}

func newSplunkO11yConnection(meta *splunkObservabilityMetadata, logger logr.Logger) (*signalflow.Client, error) {
	if meta.Realm == "" || meta.AccessToken == "" {
		return nil, fmt.Errorf("error: Could not find splunk access token or realm")
	}

	apiClient, err := signalflow.NewClient(
		signalflow.StreamURLForRealm(meta.Realm),
		signalflow.AccessToken(meta.AccessToken),
		signalflow.OnError(func(err error) {
			logger.Error(err, "error in SignalFlow client")
		}))
	if err != nil {
		return nil, fmt.Errorf("error creating SignalFlow client: %w", err)
	}

	return apiClient, nil
}

func NewSplunkObservabilityScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "splunk_observability_scaler")

	meta, err := parseSplunkObservabilityMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Splunk metadata: %w", err)
	}

	apiClient, err := newSplunkO11yConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing Splunk Observability Cloud connection: %w", err)
	}

	return &splunkObservabilityScaler{
		metadata:  meta,
		apiClient: apiClient,
		logger:    logger,
	}, nil
}

func (s *splunkObservabilityScaler) getQueryResult() (float64, error) {
	comp, err := s.apiClient.Execute(context.Background(), &signalflow.ExecuteRequest{
		Program: s.metadata.Query,
	})
	if err != nil {
		return -1, fmt.Errorf("could not execute signalflow query: %w", err)
	}

	s.logger.V(1).Info("Started MTS stream.")

	time.Sleep(time.Duration(s.metadata.Duration * int(time.Second)))
	if err := comp.Stop(context.Background()); err != nil {
		return -1, fmt.Errorf("error creating SignalFlow client: %w", err)
	}

	s.logger.V(1).Info("Closed MTS stream.")

	max := math.Inf(-1)
	min := math.Inf(1)
	valueSum := 0.0
	valueCount := 0
	s.logger.V(1).Info("Now iterating over results.")
	for msg := range comp.Data() {
		if len(msg.Payloads) == 0 {
			s.logger.V(1).Info("No data retrieved.")
			continue
		}
		for _, pl := range msg.Payloads {
			value, ok := pl.Value().(float64)
			if !ok {
				return -1, fmt.Errorf("could not convert Splunk Observability metric value to float64")
			}
			s.logger.V(1).Info(fmt.Sprintf("Encountering value %.4f\n", value))
			max = math.Max(max, value)
			min = math.Min(min, value)
			valueSum += value
			valueCount++
		}
	}

	if valueCount > 1 && s.metadata.QueryAggregator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}

	switch s.metadata.QueryAggregator {
	case "max":
		s.logger.V(1).Info(fmt.Sprintf("Returning max value: %.4f\n", max))
		return max, nil
	case "min":
		s.logger.V(1).Info(fmt.Sprintf("Returning min value: %.4f\n", min))
		return min, nil
	case "avg":
		avg := valueSum / float64(valueCount)
		s.logger.V(1).Info(fmt.Sprintf("Returning avg value: %.4f\n", avg))
		return avg, nil
	default:
		return max, nil
	}
}

func (s *splunkObservabilityScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult()

	if err != nil {
		s.logger.Error(err, "error getting metrics from Splunk Observability Cloud.")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Splunk Observability Cloud: %w", err)
	}
	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetValue, nil
}

func (s *splunkObservabilityScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString("signalfx")
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(v2.ValueMetricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *splunkObservabilityScaler) Close(context.Context) error {
	return nil
}


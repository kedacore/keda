package scalers

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// splunkO11yStreamMargin bounds the stream read beyond the configured Duration.
const splunkO11yStreamMargin = 10 * time.Second

// splunkO11yDrainTimeout is a short best-effort budget for draining after Stop.
const splunkO11yDrainTimeout = 2 * time.Second

type splunkObservabilityMetadata struct {
	TriggerIndex int

	AccessToken           string  `keda:"name=accessToken,           order=authParams"`
	Realm                 string  `keda:"name=realm,                 order=authParams"`
	Query                 string  `keda:"name=query,                 order=triggerMetadata"`
	Duration              int     `keda:"name=duration,              order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue,   	     order=triggerMetadata"`
	QueryAggregator       string  `keda:"name=queryAggregator,       order=triggerMetadata"`
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

// stopAndDrain stops the computation and keeps reading Data() until it closes or a
// short grace period elapses, so the SignalFlow client's goroutines are not left
// blocked on sends to an unconsumed channel. If process is non-nil, drained messages
// are passed to it; a process error ends the drain and is returned.
func (s *splunkObservabilityScaler) stopAndDrain(comp *signalflow.Computation, process func(*messages.DataMessage) error) error {
	stopCtx, cancel := context.WithTimeout(context.Background(), splunkO11yDrainTimeout)
	defer cancel()

	if err := comp.Stop(stopCtx); err != nil {
		s.logger.V(1).Info("Failed to stop SignalFlow computation", "error", err)
	}

	dataCh := comp.Data()
	for {
		// Give the deadline priority so a backend that keeps sending cannot extend the grace budget.
		select {
		case <-stopCtx.Done():
			s.logger.V(1).Info("Gave up draining SignalFlow data channel after stop")
			return nil
		default:
		}

		select {
		case msg, ok := <-dataCh:
			if !ok {
				return nil
			}
			if process != nil {
				if err := process(msg); err != nil {
					return err
				}
			}
		case <-stopCtx.Done():
			s.logger.V(1).Info("Gave up draining SignalFlow data channel after stop")
			return nil
		}
	}
}

func (s *splunkObservabilityScaler) getQueryResult(ctx context.Context) (float64, error) {
	comp, err := s.apiClient.Execute(ctx, &signalflow.ExecuteRequest{
		Program: s.metadata.Query,
	})
	if err != nil {
		return -1, fmt.Errorf("could not execute signalflow query: %w", err)
	}

	s.logger.V(1).Info("Started MTS stream.")

	streamDuration := time.Duration(s.metadata.Duration) * time.Second
	// Hard deadline beyond the Duration window so a non-responsive backend cannot block forever.
	streamCtx, cancel := context.WithTimeout(ctx, streamDuration+splunkO11yStreamMargin)
	defer cancel()

	stopTimer := time.NewTimer(streamDuration)
	defer stopTimer.Stop()

	maxValue := math.Inf(-1)
	minValue := math.Inf(1)
	valueSum := 0.0
	valueCount := 0

	process := func(msg *messages.DataMessage) error {
		if len(msg.Payloads) == 0 {
			s.logger.V(1).Info("No data retrieved.")
			return nil
		}
		for _, pl := range msg.Payloads {
			value, ok := pl.Value().(float64)
			if !ok {
				return fmt.Errorf("could not convert Splunk Observability metric value to float64")
			}
			s.logger.V(1).Info(fmt.Sprintf("Encountering value %.4f\n", value))
			maxValue = math.Max(maxValue, value)
			minValue = math.Min(minValue, value)
			valueSum += value
			valueCount++
		}
		return nil
	}

	s.logger.V(1).Info("Now iterating over results.")

	dataCh := comp.Data()

	// timedOut handles the hard-deadline path: stop, drain, and return the timeout error.
	timedOut := func() (float64, error) {
		s.logger.V(1).Info("Context done before stream completed; stopping computation.")
		go func() { _ = s.stopAndDrain(comp, nil) }()
		return -1, fmt.Errorf("splunk observability query ended before stream completed: %w", streamCtx.Err())
	}

loop:
	for {
		// Give the hard deadline priority: select has no fairness, so a continuously
		// ready dataCh could otherwise starve the streamCtx.Done() case.
		select {
		case <-streamCtx.Done():
			return timedOut()
		default:
		}

		select {
		case <-streamCtx.Done():
			return timedOut()
		case <-stopTimer.C:
			s.logger.V(1).Info("Stopping MTS stream after duration.")
			// Stop, but keep processing any remaining buffered messages for this window.
			if err := s.stopAndDrain(comp, process); err != nil {
				return -1, err
			}
			break loop
		case msg, ok := <-dataCh:
			if !ok {
				break loop
			}
			if err := process(msg); err != nil {
				_ = s.stopAndDrain(comp, nil)
				return -1, err
			}
		}
	}

	if valueCount == 0 {
		return 0, fmt.Errorf("query returned no data points")
	}

	if valueCount > 1 && s.metadata.QueryAggregator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}

	switch s.metadata.QueryAggregator {
	case "max":
		s.logger.V(1).Info(fmt.Sprintf("Returning max value: %.4f\n", maxValue))
		return maxValue, nil
	case "min":
		s.logger.V(1).Info(fmt.Sprintf("Returning min value: %.4f\n", minValue))
		return minValue, nil
	case "avg":
		avg := valueSum / float64(valueCount)
		s.logger.V(1).Info(fmt.Sprintf("Returning avg value: %.4f\n", avg))
		return avg, nil
	default:
		return 0, fmt.Errorf("invalid queryAggregator: %q", s.metadata.QueryAggregator)
	}
}

func (s *splunkObservabilityScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)

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

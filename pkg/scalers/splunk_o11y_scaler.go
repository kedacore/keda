package scalers

import (
	"context"
	"fmt"
	"math"

	//"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalfx-go/signalflow/v2"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type splunkO11yScaler struct {
	metadata  *splunkO11yMetadata
	apiClient *signalflow.Client
	logger    logr.Logger
}

type splunkO11yMetadata struct {
	query                string
	queryValue           float64
	queryAggegrator      string
	activationQueryValue float64
	metricName           string
	vType                v2.MetricTargetType
	accessToken          string
	realm                string
}

func NewSplunkO11yScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "splunk_o11y_scaler")

	meta, err := parseSplunkO11yMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Splunk metadata: %w", err)
	}

	apiClient, err := newSplunkO11yConnection(ctx, meta, config)
	if err != nil {
		return nil, fmt.Errorf("error establishing Splunk Observability Cloud connection: %w", err)
	}

	return &splunkO11yScaler{
		metadata:  meta,
		apiClient: apiClient,
		logger:    logger,
	}, nil
}

func parseSplunkO11yMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*splunkO11yMetadata, error) {
	meta := splunkO11yMetadata{}

	// query
	if query, ok := config.TriggerMetadata["query"]; ok {
		meta.query = query
	} else {
		return nil, fmt.Errorf("no query given")
	}

	// metric name
	if metricName, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("signalfx-%s", metricName)))
	} else {
		return nil, fmt.Errorf("no metric name given")
	}

	// queryValue
	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.queryValue = queryValue
	} else {
		if config.AsMetricSource {
			meta.queryValue = 0
		} else {
			return nil, fmt.Errorf("no queryValue given")
		}
	}

	// activationQueryValue
	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	// queryAggregator
	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case "max", "min", "avg":
			meta.queryAggegrator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of 'max', 'min', or 'avg'.", queryAggregator)
		}
	} else {
		meta.queryAggegrator = ""
	}

	// accessToken
	/*
		accessToken := os.Getenv("SPLUNK_ACCESS_TOKEN")
		if accessToken != "" {
			meta.accessToken = accessToken
		} else {
			return nil, fmt.Errorf("No Splunk Observability Cloud Access Token found.")
		}
	*/
	if accessToken, ok := config.TriggerMetadata["accessToken"]; ok {
		meta.accessToken = accessToken
	} else {
		return nil, fmt.Errorf("no accessToken given")
	}

	// test trigger auth
	/*
		if val, ok := config.AuthParams["splunkAccessToken"]; ok {
			fmt.Sprintf("splunk_o11y_scaler found authtrigger token : %s", val)
		} else {
			return nil, fmt.Errorf("no trigger auth token :(")
		}
	*/

	// realm
	/*
		realm := os.Getenv("SPLUNK_REALM")
		if realm != "" {
			meta.realm = realm
		} else {
			return nil, fmt.Errorf("No Splunk Observability Cloud Realm found.")
		}
	*/
	if realm, ok := config.TriggerMetadata["realm"]; ok {
		meta.realm = realm
	} else {
		return nil, fmt.Errorf("no realm given")
	}

	// Debug TODO check
	meta.vType = v2.ValueMetricType

	return &meta, nil
}

func newSplunkO11yConnection(ctx context.Context, meta *splunkO11yMetadata, config *scalersconfig.ScalerConfig) (*signalflow.Client, error) {
	accessToken := meta.accessToken
	realm := meta.realm

	if realm == "" || accessToken == "" {
		return nil, fmt.Errorf("error. could not find splunk access token or ream.")
	}

	apiClient, err := signalflow.NewClient(
		signalflow.StreamURLForRealm(realm),
		signalflow.AccessToken(accessToken))
	if err != nil {
		return nil, fmt.Errorf("error creating SignalFlow client: %w", err)
	}

	return apiClient, nil
}

func logMessage(logger logr.Logger, msg string, value float64) {
	if value != -1 {
		msg = fmt.Sprintf("splunk_o11y_scaler: %s -> %v", msg, value)
	} else {
		msg = fmt.Sprintf("splunk_o11y_scaler: %s", msg)
	}
	logger.Info(msg)
}

func (s *splunkO11yScaler) getQueryResult(ctx context.Context) (float64, error) {
	var duration time.Duration = 1000000000 // one second in nano seconds
	// var duration time.Duration = 10000000000 // ten seconds in nano seconds

	comp, err := s.apiClient.Execute(context.Background(), &signalflow.ExecuteRequest{
		Program: s.metadata.query,
	})
	if err != nil {
		return -1, fmt.Errorf("error: could not execute signalflow query: %w", err)
	}

	go func() {
		time.Sleep(duration)
		if err := comp.Stop(context.Background()); err != nil {
			s.logger.Info("Failed to stop computation")
		}
	}()

	logMessage(s.logger, "Received Splunk Observability metrics", -1)

	max := math.Inf(-1)
	min := math.Inf(1)
	valueSum := 0.0
	valueCount := 0
	s.logger.Info("getQueryResult -> Now Iterating")
	for msg := range comp.Data() {
		if len(msg.Payloads) == 0 {
			continue
		}
		for _, pl := range msg.Payloads {
			value, ok := pl.Value().(float64)
			if !ok {
				return -1, fmt.Errorf("error: could not convert Splunk Observability metric value to float64")
			}
			logMessage(s.logger, "Encountering value ", value)
			if value > max {
				max = value
			}
			if value < min {
				min = value
			}
			valueSum += value
			valueCount++
		}
	}

	if valueCount > 1 && s.metadata.queryAggegrator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}

	switch s.metadata.queryAggegrator {
	case "max":
		logMessage(s.logger, "Returning max value ", max)
		return max, nil
	case "min":
		logMessage(s.logger, "Returning min value ", min)
		return min, nil
	case "avg":
		avg := valueSum / float64(valueCount)
		logMessage(s.logger, "Returning avg value ", avg)
		return avg, nil
	default:
		return max, nil
	}
}

func (s *splunkO11yScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	s.logger.Info(fmt.Sprintf("splunk_o11y_scaler found authtrigger token : %s", s.metadata.accessToken))
	num, err := s.getQueryResult(ctx)

	if err != nil {
		s.logger.Error(err, "error getting metrics from Splunk Observability Cloud.")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Splunk Observability Cloud: %w", err)
	}
	metric := GenerateMetricInMili(metricName, num)

	logMessage(s.logger, "num", num)
	logMessage(s.logger, "s.metadata.activationQueryValue", s.metadata.activationQueryValue)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationQueryValue, nil
}

func (s *splunkO11yScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *splunkO11yScaler) Close(context.Context) error {
	return nil
}

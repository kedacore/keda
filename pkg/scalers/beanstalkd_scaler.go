package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	beanstalk "github.com/beanstalkd/go-beanstalk"
	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	beanstalkdJobsMetricName                   = "jobs"
	beanstalkdValueConfigName                  = "value"
	beanstalkdActivationValueTriggerConfigName = "activationValue"
	beanstalkdMetricType                       = "External"
	beanstalkdNetworkProtocol                  = "tcp"
)

type beanstalkdScaler struct {
	metricType v2.MetricTargetType
	metadata   *beanstalkdMetadata
	connection *beanstalk.Conn
	tube       *beanstalk.Tube
	logger     logr.Logger
}

type beanstalkdMetadata struct {
	server          string
	tube            string
	value           float64
	activationValue float64
	includeDelayed  bool
	timeout         time.Duration
	triggerIndex    int
}

// TubeStats represents a set of tube statistics.
type tubeStats struct {
	TotalJobs    int64 `mapstructure:"total-jobs"`
	JobsReady    int64 `mapstructure:"current-jobs-ready"`
	JobsReserved int64 `mapstructure:"current-jobs-reserved"`
	JobsUrgent   int64 `mapstructure:"current-jobs-urgent"`
	JobsBuried   int64 `mapstructure:"current-jobs-buried"`
	JobsDelayed  int64 `mapstructure:"current-jobs-delayed"`
}

func NewBeanstalkdScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	s := &beanstalkdScaler{}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	s.metricType = metricType

	s.logger = InitializeLogger(config, "beanstalkd_scaler")

	meta, err := parseBeanstalkdMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing beanstalkd metadata: %w", err)
	}
	s.metadata = meta

	conn, err := beanstalk.DialTimeout(beanstalkdNetworkProtocol, s.metadata.server, s.metadata.timeout)
	if err != nil {
		return nil, fmt.Errorf("error connecting to beanstalkd: %w", err)
	}

	s.connection = conn

	s.tube = beanstalk.NewTube(s.connection, meta.tube)

	return s, nil
}

func parseBeanstalkdMetadata(config *scalersconfig.ScalerConfig) (*beanstalkdMetadata, error) {
	meta := beanstalkdMetadata{}

	if err := parseServerValue(config, &meta); err != nil {
		return nil, err
	}

	if val, ok := config.TriggerMetadata["tube"]; ok {
		meta.tube = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	if err := parseTimeout(config, &meta); err != nil {
		return nil, err
	}

	meta.includeDelayed = false
	if val, ok := config.TriggerMetadata["includeDelayed"]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse includeDelayed value. Must be either true or false")
		}
		meta.includeDelayed = boolVal
	}

	value, valuePresent := config.TriggerMetadata[beanstalkdValueConfigName]
	activationValue, activationValuePresent := config.TriggerMetadata[beanstalkdActivationValueTriggerConfigName]

	if activationValuePresent {
		activation, err := strconv.ParseFloat(activationValue, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse %s: %w", beanstalkdActivationValueTriggerConfigName, err)
		}
		meta.activationValue = activation
	}

	if !valuePresent {
		return nil, fmt.Errorf("%s must be specified", beanstalkdValueConfigName)
	}
	triggerValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse %s: %w", beanstalkdValueConfigName, err)
	}
	meta.value = triggerValue

	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

func parseServerValue(config *scalersconfig.ScalerConfig, meta *beanstalkdMetadata) error {
	switch {
	case config.AuthParams["server"] != "":
		meta.server = config.AuthParams["server"]
	case config.TriggerMetadata["server"] != "":
		meta.server = config.TriggerMetadata["server"]
	default:
		return fmt.Errorf("no server setting given")
	}
	return nil
}

func parseTimeout(config *scalersconfig.ScalerConfig, meta *beanstalkdMetadata) error {
	if val, ok := config.TriggerMetadata["timeout"]; ok {
		timeoutMS, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("unable to parse timeout: %w", err)
		}
		if timeoutMS <= 0 {
			return fmt.Errorf("timeout must be greater than 0: %w", err)
		}
		meta.timeout = time.Duration(timeoutMS) * time.Millisecond
	} else {
		meta.timeout = config.GlobalHTTPTimeout
	}
	return nil
}

func (s *beanstalkdScaler) getTubeStats(ctx context.Context) (*tubeStats, error) {
	errCh := make(chan error)
	statsCh := make(chan *tubeStats)

	go func() {
		rawStats, err := s.tube.Stats()
		if err != nil {
			errCh <- fmt.Errorf("error retrieving stats from beanstalkd: %w", err)
		}

		var stats tubeStats
		err = mapstructure.WeakDecode(rawStats, &stats)
		if err != nil {
			errCh <- fmt.Errorf("error decoding stats from beanstalkd: %w", err)
		}

		statsCh <- &stats
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, beanstalk.ErrNotFound) {
			s.logger.Info("tube not found, setting stats to 0")
			return &tubeStats{
				TotalJobs:    0,
				JobsReady:    0,
				JobsDelayed:  0,
				JobsReserved: 0,
				JobsUrgent:   0,
				JobsBuried:   0,
			}, nil
		}
		return nil, err
	case tubeStats := <-statsCh:
		return tubeStats, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *beanstalkdScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	stats, err := s.getTubeStats(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error interacting with beanstalkd: %w", err)
	}

	totalJobs := stats.JobsReady + stats.JobsReserved

	if s.metadata.includeDelayed {
		totalJobs += stats.JobsDelayed
	}

	metric := GenerateMetricInMili(metricName, float64(totalJobs))
	isActive := float64(totalJobs) > s.metadata.activationValue

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func (s *beanstalkdScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, util.NormalizeString(fmt.Sprintf("beanstalkd-%s", url.QueryEscape(s.metadata.tube)))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.value),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: beanstalkdMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

func (s *beanstalkdScaler) Close(context.Context) error {
	if s.connection != nil {
		err := s.connection.Close()
		if err != nil {
			s.logger.Error(err, "Error closing beanstalkd connection")
			return err
		}
	}
	return nil
}

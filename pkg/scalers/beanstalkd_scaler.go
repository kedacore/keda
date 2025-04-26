package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

type BeanstalkdScaler struct {
	metricType v2.MetricTargetType
	metadata   *BeanstalkdMetadata
	connection *beanstalk.Conn
	tube       *beanstalk.Tube
	logger     logr.Logger
}

type BeanstalkdMetadata struct {
	Server          string        `keda:"name=server,          order=triggerMetadata"`
	Tube            string        `keda:"name=tube,            order=triggerMetadata"`
	Value           float64       `keda:"name=value,           order=triggerMetadata"`
	ActivationValue float64       `keda:"name=activationValue, order=triggerMetadata, optional"`
	IncludeDelayed  bool          `keda:"name=includeDelayed,  order=triggerMetadata, optional"`
	Timeout         time.Duration `keda:"name=timeout,         order=triggerMetadata, default=30"`
	TriggerIndex    int
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
	s := &BeanstalkdScaler{}

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

	conn, err := beanstalk.DialTimeout(beanstalkdNetworkProtocol, s.metadata.Server, s.metadata.Timeout)
	if err != nil {
		return nil, fmt.Errorf("error connecting to beanstalkd: %w", err)
	}

	s.connection = conn

	s.tube = beanstalk.NewTube(s.connection, meta.Tube)

	return s, nil
}

func parseBeanstalkdMetadata(config *scalersconfig.ScalerConfig) (*BeanstalkdMetadata, error) {
	meta := &BeanstalkdMetadata{}

	meta.TriggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing beanstalkd metadata: %w", err)
	}

	return meta, nil
}

func (s *BeanstalkdScaler) getTubeStats(ctx context.Context) (*tubeStats, error) {
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

func (s *BeanstalkdScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	stats, err := s.getTubeStats(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error interacting with beanstalkd: %w", err)
	}

	totalJobs := stats.JobsReady + stats.JobsReserved

	if s.metadata.IncludeDelayed {
		totalJobs += stats.JobsDelayed
	}

	metric := GenerateMetricInMili(metricName, float64(totalJobs))
	isActive := float64(totalJobs) > s.metadata.ActivationValue

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func (s *BeanstalkdScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, util.NormalizeString(fmt.Sprintf("beanstalkd-%s", url.QueryEscape(s.metadata.Tube)))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: beanstalkdMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

func (s *BeanstalkdScaler) Close(context.Context) error {
	if s.connection != nil {
		err := s.connection.Close()
		if err != nil {
			s.logger.Error(err, "Error closing beanstalkd connection")
			return err
		}
	}
	return nil
}

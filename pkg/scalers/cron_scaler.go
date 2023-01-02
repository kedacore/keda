package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultDesiredReplicas = 1
	cronMetricType         = "External"
)

type cronScaler struct {
	metricType v2.MetricTargetType
	metadata   *cronMetadata
	logger     logr.Logger
}

type cronMetadata struct {
	start           string
	end             string
	timezone        string
	desiredReplicas int64
	scalerIndex     int
}

// NewCronScaler creates a new cronScaler
func NewCronScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, parseErr := parseCronMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %w", parseErr)
	}

	return &cronScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     InitializeLogger(config, "cron_scaler"),
	}, nil
}

func getCronTime(location *time.Location, spec string) (int64, error) {
	c := cron.New(cron.WithLocation(location))
	_, err := c.AddFunc(spec, func() { _ = fmt.Sprintf("Cron initialized for location %s", location.String()) })
	if err != nil {
		return 0, err
	}

	c.Start()
	cronTime := c.Entries()[0].Next.Unix()
	c.Stop()

	return cronTime, nil
}

func parseCronMetadata(config *ScalerConfig) (*cronMetadata, error) {
	if len(config.TriggerMetadata) == 0 {
		return nil, fmt.Errorf("invalid Input Metadata. %s", config.TriggerMetadata)
	}

	meta := cronMetadata{}
	if val, ok := config.TriggerMetadata["timezone"]; ok && val != "" {
		meta.timezone = val
	} else {
		return nil, fmt.Errorf("no timezone specified. %s", config.TriggerMetadata)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if val, ok := config.TriggerMetadata["start"]; ok && val != "" {
		_, err := parser.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing start schedule: %w", err)
		}
		meta.start = val
	} else {
		return nil, fmt.Errorf("no start schedule specified. %s", config.TriggerMetadata)
	}
	if val, ok := config.TriggerMetadata["end"]; ok && val != "" {
		_, err := parser.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing end schedule: %w", err)
		}
		meta.end = val
	} else {
		return nil, fmt.Errorf("no end schedule specified. %s", config.TriggerMetadata)
	}
	if meta.start == meta.end {
		return nil, fmt.Errorf("error parsing schedule. %s: start and end can not have exactly same time input", config.TriggerMetadata)
	}
	if val, ok := config.TriggerMetadata["desiredReplicas"]; ok && val != "" {
		metadataDesiredReplicas, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing desiredReplicas metadata. %s", config.TriggerMetadata)
		}

		meta.desiredReplicas = int64(metadataDesiredReplicas)
	} else {
		return nil, fmt.Errorf("no DesiredReplicas specified. %s", config.TriggerMetadata)
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive checks if the startTime or endTime has reached
func (s *cronScaler) IsActive(ctx context.Context) (bool, error) {
	location, err := time.LoadLocation(s.metadata.timezone)
	if err != nil {
		return false, fmt.Errorf("unable to load timezone. Error: %w", err)
	}

	// Since we are considering the timestamp here and not the exact time, timezone does matter.
	currentTime := time.Now().Unix()

	nextStartTime, startTimecronErr := getCronTime(location, s.metadata.start)
	if startTimecronErr != nil {
		return false, fmt.Errorf("error initializing start cron: %w", startTimecronErr)
	}

	nextEndTime, endTimecronErr := getCronTime(location, s.metadata.end)
	if endTimecronErr != nil {
		return false, fmt.Errorf("error intializing end cron: %w", endTimecronErr)
	}

	switch {
	case nextStartTime < nextEndTime && currentTime < nextStartTime:
		return false, nil
	case currentTime <= nextEndTime:
		return true, nil
	default:
		return false, nil
	}
}

func (s *cronScaler) Close(context.Context) error {
	return nil
}

func parseCronTimeFormat(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "*", "x")
	s = strings.ReplaceAll(s, "/", "Sl")
	s = strings.ReplaceAll(s, "?", "Qm")
	return s
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cronScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var specReplicas int64 = 1
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("cron-%s-%s-%s", s.metadata.timezone, parseCronTimeFormat(s.metadata.start), parseCronTimeFormat(s.metadata.end)))),
		},
		Target: GetMetricTarget(s.metricType, specReplicas),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cronMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetrics finds the current value of the metric
func (s *cronScaler) GetMetrics(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, error) {
	var currentReplicas = int64(defaultDesiredReplicas)
	isActive, err := s.IsActive(ctx)
	if err != nil {
		s.logger.Error(err, "error")
		return []external_metrics.ExternalMetricValue{}, err
	}
	if isActive {
		currentReplicas = s.metadata.desiredReplicas
	}

	/*******************************************************************************/
	metric := GenerateMetricInMili(metricName, float64(currentReplicas))

	return []external_metrics.ExternalMetricValue{metric}, nil
}

// TODO merge isActive() and GetMetrics()
func (s *cronScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metrics, err := s.GetMetrics(ctx, metricName)
	if err != nil {
		s.logger.Error(err, "error getting metrics")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	isActive, err := s.IsActive(ctx)
	if err != nil {
		s.logger.Error(err, "error getting activity status")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	return metrics, isActive, nil
}

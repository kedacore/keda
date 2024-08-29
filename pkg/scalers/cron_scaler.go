package scalers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultDesiredReplicas = 1
	cronMetricType         = "External"
)

type cronScaler struct {
	metricType v2.MetricTargetType
	metadata   cronMetadata
	logger     logr.Logger
}

type cronMetadata struct {
	Start           string `keda:"name=start,           order=triggerMetadata"`
	End             string `keda:"name=end,             order=triggerMetadata"`
	Timezone        string `keda:"name=timezone,        order=triggerMetadata"`
	DesiredReplicas int64  `keda:"name=desiredReplicas, order=triggerMetadata"`
	TriggerIndex    int
}

func (m *cronMetadata) Validate() error {
	if m.Timezone == "" {
		return fmt.Errorf("no timezone specified")
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if m.Start == "" {
		return fmt.Errorf("no start schedule specified")
	}
	if _, err := parser.Parse(m.Start); err != nil {
		return fmt.Errorf("error parsing start schedule: %w", err)
	}

	if m.End == "" {
		return fmt.Errorf("no end schedule specified")
	}
	if _, err := parser.Parse(m.End); err != nil {
		return fmt.Errorf("error parsing end schedule: %w", err)
	}

	if m.Start == m.End {
		return fmt.Errorf("start and end can not have exactly same time input")
	}

	if m.DesiredReplicas == 0 {
		return fmt.Errorf("no desiredReplicas specified")
	}

	return nil
}

// NewCronScaler creates a new cronScaler
func NewCronScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseCronMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %w", err)
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

func parseCronMetadata(config *scalersconfig.ScalerConfig) (cronMetadata, error) {
	meta := cronMetadata{TriggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, err
	}
	return meta, nil
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
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("cron-%s-%s-%s", s.metadata.Timezone, parseCronTimeFormat(s.metadata.Start), parseCronTimeFormat(s.metadata.End)))),
		},
		Target: GetMetricTarget(s.metricType, specReplicas),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cronMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *cronScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var defaultDesiredReplicas = int64(defaultDesiredReplicas)

	location, err := time.LoadLocation(s.metadata.Timezone)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to load timezone. Error: %w", err)
	}

	// Since we are considering the timestamp here and not the exact time, timezone does matter.
	currentTime := time.Now().Unix()

	nextStartTime, startTimecronErr := getCronTime(location, s.metadata.Start)
	if startTimecronErr != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error initializing start cron: %w", startTimecronErr)
	}

	nextEndTime, endTimecronErr := getCronTime(location, s.metadata.End)
	if endTimecronErr != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error intializing end cron: %w", endTimecronErr)
	}

	switch {
	case nextStartTime < nextEndTime && currentTime < nextStartTime:
		metric := GenerateMetricInMili(metricName, float64(defaultDesiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, false, nil
	case currentTime <= nextEndTime:
		metric := GenerateMetricInMili(metricName, float64(s.metadata.DesiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, true, nil
	default:
		metric := GenerateMetricInMili(metricName, float64(defaultDesiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, false, nil
	}
}

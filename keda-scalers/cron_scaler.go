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

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	cronMetricType = "External"
)

type cronScaler struct {
	metricType    v2.MetricTargetType
	metadata      cronMetadata
	logger        logr.Logger
	startSchedule cron.Schedule
	endSchedule   cron.Schedule
}

type cronMetadata struct {
	Start           string `keda:"name=start,           order=triggerMetadata"`
	End             string `keda:"name=end,             order=triggerMetadata"`
	Timezone        string `keda:"name=timezone,        order=triggerMetadata"`
	DesiredReplicas int64  `keda:"name=desiredReplicas, order=triggerMetadata"`
	TriggerIndex    int
}

func (m *cronMetadata) Validate() error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(m.Start); err != nil {
		return fmt.Errorf("error parsing start schedule: %w", err)
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

func NewCronScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseCronMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %w", err)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	startSchedule, _ := parser.Parse(meta.Start)
	endSchedule, _ := parser.Parse(meta.End)

	return &cronScaler{
		metricType:    metricType,
		metadata:      meta,
		logger:        InitializeLogger(config, "cron_scaler"),
		startSchedule: startSchedule,
		endSchedule:   endSchedule,
	}, nil
}

func getCronTime(location *time.Location, schedule cron.Schedule) time.Time {
	// Use the pre-parsed cron schedule directly to get the next time
	return schedule.Next(time.Now().In(location))
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

func (s *cronScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	location, err := time.LoadLocation(s.metadata.Timezone)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to load timezone: %w", err)
	}

	currentTime := time.Now().In(location)

	// Use the pre-parsed schedules to get the next start and end times
	nextStartTime := getCronTime(location, s.startSchedule)
	nextEndTime := getCronTime(location, s.endSchedule)

	isWithinInterval := false

	if nextStartTime.Before(nextEndTime) {
		// Interval within the same day
		isWithinInterval = currentTime.After(nextStartTime) && currentTime.Before(nextEndTime)
	} else {
		// Interval spans midnight
		isWithinInterval = currentTime.After(nextStartTime) || currentTime.Before(nextEndTime)
	}

	metricValue := float64(1)
	if isWithinInterval {
		metricValue = float64(s.metadata.DesiredReplicas)
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, isWithinInterval, nil
}

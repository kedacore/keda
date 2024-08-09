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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	startSched      cron.Schedule
	endSched        cron.Schedule
	timezone        string
	desiredReplicas int64
	triggerIndex    int
}

// NewCronScaler creates a new cronScaler
func NewCronScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
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

func getCronTime(location *time.Location, sched cron.Schedule) int64 {
	return sched.Next(time.Now().In(location)).Unix()
}

func parseCronMetadata(config *scalersconfig.ScalerConfig) (*cronMetadata, error) {
	var err error
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
		meta.startSched, err = parser.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing start schedule: %w", err)
		}
		meta.start = val
	} else {
		return nil, fmt.Errorf("no start schedule specified. %s", config.TriggerMetadata)
	}
	if val, ok := config.TriggerMetadata["end"]; ok && val != "" {
		meta.endSched, err = parser.Parse(val)
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
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
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
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("cron-%s-%s-%s", s.metadata.timezone, parseCronTimeFormat(s.metadata.start), parseCronTimeFormat(s.metadata.end)))),
		},
		Target: GetMetricTarget(s.metricType, specReplicas),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cronMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *cronScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var defaultDesiredReplicas = int64(defaultDesiredReplicas)

	location, err := time.LoadLocation(s.metadata.timezone)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to load timezone. Error: %w", err)
	}

	// Since we are considering the timestamp here and not the exact time, timezone does matter.
	currentTime := time.Now().Unix()

	nextStartTime := getCronTime(location, s.metadata.startSched)

	nextEndTime := getCronTime(location, s.metadata.endSched)

	switch {
	case nextStartTime < nextEndTime && currentTime < nextStartTime:
		metric := GenerateMetricInMili(metricName, float64(defaultDesiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, false, nil
	case currentTime <= nextEndTime:
		metric := GenerateMetricInMili(metricName, float64(s.metadata.desiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, true, nil
	default:
		metric := GenerateMetricInMili(metricName, float64(defaultDesiredReplicas))
		return []external_metrics.ExternalMetricValue{metric}, false, nil
	}
}

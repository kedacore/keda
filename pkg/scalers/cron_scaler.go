package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultDesiredReplicas = 1
	cronMetricType         = "External"
)

type cronScaler struct {
	metadata *cronMetadata
}

type cronMetadata struct {
	start           string
	end             string
	timezone        string
	desiredReplicas int64
}

var cronLog = logf.Log.WithName("cron_scaler")

// NewCronScaler creates a new cronScaler
func NewCronScaler(config *ScalerConfig) (Scaler, error) {
	meta, parseErr := parseCronMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %s", parseErr)
	}

	return &cronScaler{
		metadata: meta,
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
	if val, ok := config.TriggerMetadata["start"]; ok && val != "" {
		meta.start = val
	} else {
		return nil, fmt.Errorf("no start schedule specified. %s", config.TriggerMetadata)
	}
	if val, ok := config.TriggerMetadata["end"]; ok && val != "" {
		meta.end = val
	} else {
		return nil, fmt.Errorf("no end schedule specified. %s", config.TriggerMetadata)
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

	return &meta, nil
}

// IsActive checks if the startTime or endTime has reached
func (s *cronScaler) IsActive(ctx context.Context) (bool, error) {
	location, err := time.LoadLocation(s.metadata.timezone)
	if err != nil {
		return false, fmt.Errorf("unable to load timezone. Error: %s", err)
	}

	nextStartTime, startTimecronErr := getCronTime(location, s.metadata.start)
	if startTimecronErr != nil {
		return false, fmt.Errorf("error initializing start cron: %s", startTimecronErr)
	}

	nextEndTime, endTimecronErr := getCronTime(location, s.metadata.end)
	if endTimecronErr != nil {
		return false, fmt.Errorf("error intializing end cron: %s", endTimecronErr)
	}

	// Since we are considering the timestamp here and not the exact time, timezone does matter.
	currentTime := time.Now().Unix()
	switch {
	case nextStartTime < nextEndTime && currentTime < nextStartTime:
		return false, nil
	case currentTime <= nextEndTime:
		return true, nil
	default:
		return false, nil
	}
}

func (s *cronScaler) Close() error {
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
func (s *cronScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	specReplicas := 1
	targetMetricValue := resource.NewQuantity(int64(specReplicas), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", "cron", s.metadata.timezone, parseCronTimeFormat(s.metadata.start), parseCronTimeFormat(s.metadata.end))),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: cronMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics finds the current value of the metric
func (s *cronScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	var currentReplicas = int64(defaultDesiredReplicas)
	isActive, err := s.IsActive(ctx)
	if err != nil {
		cronLog.Error(err, "error")
		return []external_metrics.ExternalMetricValue{}, err
	}
	if isActive {
		currentReplicas = s.metadata.desiredReplicas
	}

	/*******************************************************************************/
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(currentReplicas, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

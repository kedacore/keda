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

	kedautil "github.com/kedacore/keda/pkg/util"
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
func NewCronScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, parseErr := parseCronMetadata(metadata, resolvedEnv)
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

func parseCronMetadata(metadata, resolvedEnv map[string]string) (*cronMetadata, error) {
	if len(metadata) == 0 {
		return nil, fmt.Errorf("Invalid Input Metadata. %s", metadata)
	}

	meta := cronMetadata{}
	if val, ok := metadata["timezone"]; ok && val != "" {
		meta.timezone = val
	} else {
		return nil, fmt.Errorf("No timezone specified. %s", metadata)
	}
	if val, ok := metadata["start"]; ok && val != "" {
		meta.start = val
	} else {
		return nil, fmt.Errorf("No start schedule specified. %s", metadata)
	}
	if val, ok := metadata["end"]; ok && val != "" {
		meta.end = val
	} else {
		return nil, fmt.Errorf("No end schedule specified. %s", metadata)
	}
	if val, ok := metadata["desiredReplicas"]; ok && val != "" {
		metadataDesiredReplicas, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Error parsing desiredReplicas metadata. %s", metadata)
		}

		meta.desiredReplicas = int64(metadataDesiredReplicas)
	} else {
		return nil, fmt.Errorf("No DesiredReplicas specified. %s", metadata)
	}

	return &meta, nil
}

// IsActive checks if the startTime or endTime has reached
func (s *cronScaler) IsActive(ctx context.Context) (bool, error) {
	location, err := time.LoadLocation(s.metadata.timezone)
	if err != nil {
		return false, fmt.Errorf("Unable to load timezone. Error: %s", err)
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
	if nextStartTime < nextEndTime && currentTime < nextStartTime {
		return false, nil
	} else if currentTime <= nextEndTime {
		return true, nil
	} else {
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

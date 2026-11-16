package scalers

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
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
	cronMetricType = "External"
)

var (
	cronHashRegex   = regexp.MustCompile(`^H(?:\((\d+)-(\d+)\))?(?:/(\d+))?$`)
	cronFieldBounds = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	// Day of month is capped at 28 for a bare H so every month has the chosen day, matching Jenkins.
	cronHashBounds = [5][2]int{{0, 59}, {0, 23}, {1, 28}, {1, 12}, {0, 6}}
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
	if _, err := parseCronSchedule(m.Start, ""); err != nil {
		return fmt.Errorf("error parsing start schedule: %w", err)
	}

	if _, err := parseCronSchedule(m.End, ""); err != nil {
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

	seed := fmt.Sprintf("%s/%s/%d", config.ScalableObjectNamespace, config.ScalableObjectName, config.TriggerIndex)
	startExpr, _ := expandCronHash(meta.Start, seed)
	endExpr, _ := expandCronHash(meta.End, seed)
	if startExpr == endExpr {
		return nil, fmt.Errorf("start and end resolve to the same schedule %q", startExpr)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	startSchedule, _ := parser.Parse(startExpr)
	endSchedule, _ := parser.Parse(endExpr)

	return &cronScaler{
		metricType:    metricType,
		metadata:      meta,
		logger:        InitializeLogger(config, "cron_scaler"),
		startSchedule: startSchedule,
		endSchedule:   endSchedule,
	}, nil
}

func parseCronSchedule(expr, seed string) (cron.Schedule, error) {
	expanded, err := expandCronHash(expr, seed)
	if err != nil {
		return nil, err
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser.Parse(expanded)
}

// expandCronHash replaces Jenkins-style H tokens (H, H(a-b), H/n, H(a-b)/n) with values derived
// from the seed, so each trigger gets a stable but distinct time within the given range.
func expandCronHash(expr, seed string) (string, error) {
	fields := strings.Fields(expr)
	if len(fields) != len(cronHashBounds) {
		return expr, nil
	}

	for i, field := range fields {
		parts := strings.Split(field, ",")
		for j, part := range parts {
			match := cronHashRegex.FindStringSubmatch(part)
			if match == nil {
				continue
			}

			low, high := cronHashBounds[i][0], cronHashBounds[i][1]
			if match[1] != "" {
				low, _ = strconv.Atoi(match[1])
				high, _ = strconv.Atoi(match[2])
				if low > high || low < cronFieldBounds[i][0] || high > cronFieldBounds[i][1] {
					return "", fmt.Errorf("invalid H range %q", part)
				}
			}

			h := fnv.New64a()
			fmt.Fprintf(h, "%s/%d", seed, i)
			hash := h.Sum64()

			if match[3] == "" {
				parts[j] = strconv.Itoa(low + int(hash%uint64(high-low+1)))
				continue
			}

			step, _ := strconv.Atoi(match[3])
			if step == 0 || step > high-low+1 {
				return "", fmt.Errorf("invalid H step %q", part)
			}
			parts[j] = fmt.Sprintf("%d-%d/%d", low+int(hash%uint64(step)), high, step)
		}
		fields[i] = strings.Join(parts, ",")
	}

	return strings.Join(fields, " "), nil
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
	s = strings.ReplaceAll(s, ",", "Cm")
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

	metricValue := float64(0)
	if isWithinInterval {
		metricValue = float64(s.metadata.DesiredReplicas)
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, isWithinInterval, nil
}

package scalers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type cronMinReplicasScaler struct {
	metricType    v2.MetricTargetType
	metadata      cronMinReplicasMetadata
	logger        logr.Logger
	startSchedule cron.Schedule
	endSchedule   cron.Schedule
	kubeClient    client.Client
}

type cronMinReplicasMetadata struct {
	Start        string `keda:"name=start,       order=triggerMetadata"`
	End          string `keda:"name=end,         order=triggerMetadata"`
	Timezone     string `keda:"name=timezone,    order=triggerMetadata"`
	MinReplicas  int64  `keda:"name=minReplicas, order=triggerMetadata"`
	MaxReplicas  int64  `keda:"name=maxReplicas, order=triggerMetadata, optional"`
	TriggerIndex int

	scalableObjectName      string
	scalableObjectNamespace string
}

func (m *cronMinReplicasMetadata) Validate() error {
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

	if m.MinReplicas <= 0 {
		return fmt.Errorf("minReplicas must be greater than 0")
	}

	if m.MaxReplicas < 0 {
		return fmt.Errorf("maxReplicas must not be negative")
	}

	if m.MaxReplicas > 0 && m.MaxReplicas < m.MinReplicas {
		return fmt.Errorf("maxReplicas (%d) must be greater than or equal to minReplicas (%d)", m.MaxReplicas, m.MinReplicas)
	}

	return nil
}

func NewCronMinReplicasScaler(kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseCronMinReplicasMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cron-min-replicas metadata: %w", err)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	startSchedule, _ := parser.Parse(meta.Start)
	endSchedule, _ := parser.Parse(meta.End)

	return &cronMinReplicasScaler{
		metricType:    metricType,
		metadata:      meta,
		logger:        InitializeLogger(config, "cron_min_replicas_scaler"),
		startSchedule: startSchedule,
		endSchedule:   endSchedule,
		kubeClient:    kubeClient,
	}, nil
}

func parseCronMinReplicasMetadata(config *scalersconfig.ScalerConfig) (cronMinReplicasMetadata, error) {
	meta := cronMinReplicasMetadata{
		TriggerIndex:            config.TriggerIndex,
		scalableObjectName:      config.ScalableObjectName,
		scalableObjectNamespace: config.ScalableObjectNamespace,
	}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func (s *cronMinReplicasScaler) Close(context.Context) error {
	return nil
}

func parseCronMinReplicasTimeFormat(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "*", "x")
	s = strings.ReplaceAll(s, "/", "Sl")
	s = strings.ReplaceAll(s, "?", "Qm")
	s = strings.ReplaceAll(s, ",", "Cm")
	return s
}

func (s *cronMinReplicasScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var specReplicas int64 = 1
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("cron-min-replicas-%s-%s-%s", s.metadata.Timezone, parseCronMinReplicasTimeFormat(s.metadata.Start), parseCronMinReplicasTimeFormat(s.metadata.End)))),
		},
		Target: GetMetricTarget(s.metricType, specReplicas),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: cronMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *cronMinReplicasScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	location, err := time.LoadLocation(s.metadata.Timezone)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("unable to load timezone: %w", err)
	}

	currentTime := time.Now().In(location)

	nextStartTime := s.startSchedule.Next(currentTime)
	nextEndTime := s.endSchedule.Next(currentTime)

	isWithinInterval := false
	if nextStartTime.Before(nextEndTime) {
		isWithinInterval = currentTime.After(nextStartTime) && currentTime.Before(nextEndTime)
	} else {
		isWithinInterval = currentTime.After(nextStartTime) || currentTime.Before(nextEndTime)
	}

	if isWithinInterval && s.metadata.MaxReplicas > 0 {
		if err := s.patchMaxReplicas(ctx, int32(s.metadata.MaxReplicas)); err != nil {
			s.logger.Error(err, "failed to patch ScaledObject maxReplicaCount")
		}
	}

	metricValue := float64(0)
	if isWithinInterval {
		metricValue = float64(s.metadata.MinReplicas)
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, isWithinInterval, nil
}

func (s *cronMinReplicasScaler) patchMaxReplicas(ctx context.Context, desired int32) error {
	if s.kubeClient == nil {
		return nil
	}
	so := &kedav1alpha1.ScaledObject{}
	if err := s.kubeClient.Get(ctx, types.NamespacedName{Name: s.metadata.scalableObjectName, Namespace: s.metadata.scalableObjectNamespace}, so); err != nil {
		return fmt.Errorf("failed to get ScaledObject: %w", err)
	}

	if so.Spec.MaxReplicaCount != nil && *so.Spec.MaxReplicaCount == desired {
		return nil
	}

	patch := client.MergeFrom(so.DeepCopy())
	so.Spec.MaxReplicaCount = &desired
	return s.kubeClient.Patch(ctx, so, patch)
}

package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type monitorChannelInfo struct {
	Name         string                  `json:"name"`
	MsgCount     int64                   `json:"msgs"`
	LastSequence int64                   `json:"last_seq"`
	Subscriber   []monitorSubscriberInfo `json:"subscriptions"`
}

type monitorSubscriberInfo struct {
	ClientID     string `json:"client_id"`
	QueueName    string `json:"queue_name"`
	Inbox        string `json:"inbox"`
	AckInbox     string `json:"ack_inbox"`
	IsDurable    bool   `json:"is_durable"`
	IsOffline    bool   `json:"is_offline"`
	MaxInflight  int    `json:"max_inflight"`
	LastSent     int64  `json:"last_sent"`
	PendingCount int    `json:"pending_count"`
	IsStalled    bool   `json:"is_stalled"`
}

type stanScaler struct {
	channelInfo *monitorChannelInfo
	metadata    stanMetadata
}

type stanMetadata struct {
	natsServerMonitoringEndpoint string
	queueGroup                   string
	durableName                  string
	subject                      string
	lagThreshold                 int64
}

const (
	stanLagThresholdMetricName = "lagThreshold"
	stanMetricType             = "External"
	defaultStanLagThreshold    = 10
)

var stanLog = logf.Log.WithName("stan_scaler")

// NewStanScaler creates a new stanScaler
func NewStanScaler(resolvedSecrets, metadata map[string]string) (Scaler, error) {
	stanMetadata, err := parseStanMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %s", err)
	}

	return &stanScaler{
		channelInfo: &monitorChannelInfo{},
		metadata:    stanMetadata,
	}, nil
}

func parseStanMetadata(metadata map[string]string) (stanMetadata, error) {
	meta := stanMetadata{}

	if metadata["natsServerMonitoringEndpoint"] == "" {
		return meta, errors.New("no monitoring endpoint given")
	}
	meta.natsServerMonitoringEndpoint = metadata["natsServerMonitoringEndpoint"]

	if metadata["queueGroup"] == "" {
		return meta, errors.New("no queue group given")
	}
	meta.queueGroup = metadata["queueGroup"]

	if metadata["durableName"] == "" {
		return meta, errors.New("no durable name group given")
	}
	meta.durableName = metadata["durableName"]

	if metadata["subject"] == "" {
		return meta, errors.New("no subject given")
	}
	meta.subject = metadata["subject"]

	meta.lagThreshold = defaultStanLagThreshold

	if val, ok := metadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %s", lagThresholdMetricName, err)
		}
		meta.lagThreshold = t
	}

	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *stanScaler) IsActive(ctx context.Context) (bool, error) {
	resp, err := http.Get(s.getMonitoringEndpoint())
	if err != nil {
		stanLog.Error(err, "Unable to access the nats streaming broker monitoring endpoint", "natsServerMonitoringEndpoint", s.metadata.natsServerMonitoringEndpoint)
		return false, err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&s.channelInfo)

	return s.hasPendingMessage() || s.getMaxMsgLag() > 0, nil
}

func (s *stanScaler) getMonitoringEndpoint() string {
	return "http://" + s.metadata.natsServerMonitoringEndpoint + "/streaming/channelsz?" + "channel=" + s.metadata.subject + "&subs=1"
}

func (s *stanScaler) getTotalMessages() int64 {
	return s.channelInfo.MsgCount
}

func (s *stanScaler) getMaxMsgLag() int64 {
	var maxValue int64
	maxValue = 0
	for _, subs := range s.channelInfo.Subscriber {
		if subs.LastSent > maxValue && subs.QueueName == (s.metadata.durableName+":"+s.metadata.queueGroup) {
			maxValue = subs.LastSent
		}
	}

	return s.channelInfo.MsgCount - maxValue
}

func (s *stanScaler) hasPendingMessage() bool {
	var hasPending bool
	hasPending = false
	for _, subs := range s.channelInfo.Subscriber {
		if subs.PendingCount > 0 && subs.QueueName == (s.metadata.durableName+":"+s.metadata.queueGroup) {
			hasPending = true
		}
	}

	return hasPending
}

func (s *stanScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         lagThresholdMetricName,
				TargetAverageValue: resource.NewQuantity(s.metadata.lagThreshold, resource.DecimalSI),
			},
			Type: stanMetricType,
		},
	}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *stanScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	resp, err := http.Get(s.getMonitoringEndpoint())

	if err != nil {
		stanLog.Error(err, "Unable to access the nats streaming broker monitoring endpoint", "natsServerMonitoringEndpoint", s.metadata.natsServerMonitoringEndpoint)
		return []external_metrics.ExternalMetricValue{}, err
	}

	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&s.channelInfo)
	totalLag := s.getMaxMsgLag()
	stanLog.V(1).Info("Stan scaler: Providing metrics based on totalLag, threshold", "totalLag", totalLag, "lagThreshold", s.metadata.lagThreshold)
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(totalLag), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Nothing to close here.
func (s *stanScaler) Close() error {
	return nil
}

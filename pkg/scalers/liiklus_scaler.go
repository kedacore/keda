package scalers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	liiklus_service "github.com/kedacore/keda/v2/pkg/scalers/liiklus"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type liiklusScaler struct {
	metricType v2.MetricTargetType
	metadata   *liiklusMetadata
	connection *grpc.ClientConn
	client     liiklus_service.LiiklusServiceClient
	logger     logr.Logger
}

type liiklusMetadata struct {
	lagThreshold           int64
	activationLagThreshold int64
	address                string
	topic                  string
	group                  string
	groupVersion           uint32
	triggerIndex           int
}

const (
	defaultLiiklusLagThreshold           int64 = 10
	defaultLiiklusActivationLagThreshold int64 = 0
)

const (
	liiklusLagThresholdMetricName           = "lagThreshold"
	liiklusActivationLagThresholdMetricName = "activationLagThreshold"
	liiklusMetricType                       = "External"
)

var (
	// ErrLiiklusNoTopic is returned when "topic" in the config is empty.
	ErrLiiklusNoTopic = errors.New("no topic provided")

	// ErrLiiklusNoAddress is returned when "address" in the config is empty.
	ErrLiiklusNoAddress = errors.New("no liiklus API address provided")

	// ErrLiiklusNoGroup is returned when "group" in the config is empty.
	ErrLiiklusNoGroup = errors.New("no consumer group provided")
)

// NewLiiklusScaler creates a new liiklusScaler scaler
func NewLiiklusScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	lm, err := parseLiiklusMetadata(config)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(lm.address,
		grpc.WithDefaultServiceConfig(grpcConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c := liiklus_service.NewLiiklusServiceClient(conn)

	scaler := liiklusScaler{
		connection: conn,
		client:     c,
		metricType: metricType,
		metadata:   lm,
		logger:     InitializeLogger(config, "liiklus_scaler"),
	}
	return &scaler, nil
}

func (s *liiklusScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalLag, lags, err := s.getLag(ctx)
	if err != nil {
		return nil, false, err
	}

	if totalLag/uint64(s.metadata.lagThreshold) > uint64(len(lags)) {
		totalLag = uint64(s.metadata.lagThreshold) * uint64(len(lags))
	}

	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLag > uint64(s.metadata.activationLagThreshold), nil
}

func (s *liiklusScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("liiklus-%s", s.metadata.topic))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: liiklusMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *liiklusScaler) Close(context.Context) error {
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing liiklus connection")
		return err
	}

	return nil
}

// getLag returns the total lag, as well as per-partition lag for this scaler. That is, the difference between the
// latest offset available on this scaler topic, and the position of the consumer group this scaler is configured for.
func (s *liiklusScaler) getLag(ctx context.Context) (uint64, map[uint32]uint64, error) {
	var totalLag uint64
	ctx1, cancel1 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel1()
	gor, err := s.client.GetOffsets(ctx1, &liiklus_service.GetOffsetsRequest{
		Topic:        s.metadata.topic,
		Group:        s.metadata.group,
		GroupVersion: s.metadata.groupVersion,
	})
	if err != nil {
		return 0, nil, err
	}

	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	geor, err := s.client.GetEndOffsets(ctx2, &liiklus_service.GetEndOffsetsRequest{
		Topic: s.metadata.topic,
	})
	if err != nil {
		return 0, nil, err
	}

	lags := make(map[uint32]uint64, len(geor.Offsets))

	for part, o := range geor.GetOffsets() {
		diff := o - gor.Offsets[part]
		lags[part] = diff
		totalLag += diff
	}
	return totalLag, lags, nil
}

func parseLiiklusMetadata(config *scalersconfig.ScalerConfig) (*liiklusMetadata, error) {
	lagThreshold := defaultLiiklusLagThreshold
	activationLagThreshold := defaultLiiklusActivationLagThreshold

	if val, ok := config.TriggerMetadata[liiklusLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", liiklusLagThresholdMetricName, err)
		}
		lagThreshold = t
	}

	if val, ok := config.TriggerMetadata[liiklusActivationLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", liiklusActivationLagThresholdMetricName, err)
		}
		activationLagThreshold = t
	}

	groupVersion := uint32(0)
	if val, ok := config.TriggerMetadata["groupVersion"]; ok {
		t, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing groupVersion: %w", err)
		}
		groupVersion = uint32(t)
	}

	switch {
	case config.TriggerMetadata["topic"] == "":
		return nil, ErrLiiklusNoTopic
	case config.TriggerMetadata["address"] == "":
		return nil, ErrLiiklusNoAddress
	case config.TriggerMetadata["group"] == "":
		return nil, ErrLiiklusNoGroup
	}

	return &liiklusMetadata{
		topic:                  config.TriggerMetadata["topic"],
		address:                config.TriggerMetadata["address"],
		group:                  config.TriggerMetadata["group"],
		groupVersion:           groupVersion,
		lagThreshold:           lagThreshold,
		activationLagThreshold: activationLagThreshold,
		triggerIndex:           config.TriggerIndex,
	}, nil
}

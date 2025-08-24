package scalers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	liiklus_service "github.com/kedacore/keda/v2/keda-scalers/liiklus"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
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
	LagThreshold           int64  `keda:"name=lagThreshold,order=triggerMetadata,default=10"`
	ActivationLagThreshold int64  `keda:"name=activationLagThreshold,order=triggerMetadata,default=0"`
	Address                string `keda:"name=address,order=triggerMetadata"`
	Topic                  string `keda:"name=topic,order=triggerMetadata"`
	Group                  string `keda:"name=group,order=triggerMetadata"`
	GroupVersion           uint32 `keda:"name=groupVersion,order=triggerMetadata,default=0"`
	triggerIndex           int
}

const (
	liiklusMetricType = "External"
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

	conn, err := grpc.NewClient(lm.Address,
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

	if totalLag/uint64(s.metadata.LagThreshold) > uint64(len(lags)) {
		totalLag = uint64(s.metadata.LagThreshold) * uint64(len(lags))
	}

	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLag > uint64(s.metadata.ActivationLagThreshold), nil
}

func (s *liiklusScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("liiklus-%s", s.metadata.Topic))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.LagThreshold),
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
		Topic:        s.metadata.Topic,
		Group:        s.metadata.Group,
		GroupVersion: s.metadata.GroupVersion,
	})
	if err != nil {
		return 0, nil, err
	}

	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	geor, err := s.client.GetEndOffsets(ctx2, &liiklus_service.GetEndOffsetsRequest{
		Topic: s.metadata.Topic,
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
	meta := &liiklusMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing liiklus metadata: %w", err)
	}
	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

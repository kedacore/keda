package scalers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	liiklus_service "github.com/kedacore/keda/pkg/scalers/liiklus"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type liiklusScaler struct {
	metadata   *liiklusMetadata
	connection *grpc.ClientConn
	client     liiklus_service.LiiklusServiceClient
}

type liiklusMetadata struct {
	lagThreshold int64
	address      string
	topic        string
	group        string
	groupVersion uint32
}

const (
	defaultLiiklusLagThreshold    int64 = 10
	liiklusLagThresholdMetricName       = "lagThreshold"
	liiklusMetricType                   = "External"
)

func NewLiiklusScaler(resolvedEnv map[string]string, metadata map[string]string) (*liiklusScaler, error) {

	lm, err := parseLiiklusMetadata(metadata)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(lm.address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := liiklus_service.NewLiiklusServiceClient(conn)

	scaler := liiklusScaler{
		connection: conn,
		client:     c,
		metadata:   lm,
	}
	return &scaler, nil
}

func (s *liiklusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	if totalLag, lags, err := s.getLag(ctx); err != nil {
		return nil, err
	} else {
		if totalLag/uint64(s.metadata.lagThreshold) > uint64(len(lags)) {
			totalLag = uint64(s.metadata.lagThreshold) * uint64(len(lags))
		}

		return []external_metrics.ExternalMetricValue{
			{
				MetricName: metricName,
				Timestamp:  meta_v1.Now(),
				Value:      *resource.NewQuantity(int64(totalLag), resource.DecimalSI),
			},
		}, nil

	}

}

func (s *liiklusScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         liiklusLagThresholdMetricName,
				TargetAverageValue: resource.NewQuantity(s.metadata.lagThreshold, resource.DecimalSI),
			},
			Type: liiklusMetricType,
		},
	}
}

func (s *liiklusScaler) Close() error {
	err := s.connection.Close()
	if err != nil {
		return err
	}
	return nil
}

// IsActive returns true if there is any lag on any partition.
func (s *liiklusScaler) IsActive(ctx context.Context) (bool, error) {
	if lag, _, err := s.getLag(ctx); err != nil {
		return false, err
	} else {
		return lag > 0, nil
	}
}

// getLag returns the total lag, as well as per-partition lag for this scaler. That is, the difference between the
// latest offset available on this scaler topic, and the position of the consumer group this scaler is configured for.
func (s *liiklusScaler) getLag(ctx context.Context) (uint64, map[uint32]uint64, error) {
	var totalLag uint64 = 0
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

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
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

func parseLiiklusMetadata(metadata map[string]string) (*liiklusMetadata, error) {

	lagThreshold := defaultLiiklusLagThreshold

	if val, ok := metadata[liiklusLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", liiklusLagThresholdMetricName, err)
		}
		lagThreshold = int64(t)
	}

	groupVersion := uint32(0)
	if val, ok := metadata["groupVersion"]; ok {
		t, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing groupVersion: %s", err)
		}
		groupVersion = uint32(t)
	}

	if metadata["topic"] == "" {
		return nil, errors.New("no topic provided")
	} else if metadata["address"] == "" {
		return nil, errors.New("no liiklus API address provided")
	} else if metadata["group"] == "" {
		return nil, errors.New("no consumer group provided")
	}

	return &liiklusMetadata{
		topic:        metadata["topic"],
		address:      metadata["address"],
		group:        metadata["group"],
		groupVersion: groupVersion,
		lagThreshold: lagThreshold,
	}, nil
}

package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	listLengthMetricName    = "RedisListLength"
	defaultTargetListLength = 5
	defaultRedisAddress     = "redis-master.default.svc.cluster.local:6379"
	defaultRedisPassword    = ""
)

type redisScaler struct {
	metadata *redisMetadata
}

type redisMetadata struct {
	targetListLength int
	listName         string
	address          string
	password         string
}

// NewRedisScaler creates a new redisScaler
func NewRedisScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseRedisMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %s", err)
	}

	return &redisScaler{
		metadata: meta,
	}, nil
}

func parseRedisMetadata(metadata, resolvedEnv map[string]string) (*redisMetadata, error) {
	meta := redisMetadata{}
	meta.targetListLength = defaultTargetListLength

	if val, ok := metadata["listLength"]; ok {
		listLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("List length parsing error %s", err.Error())
		}
		meta.targetListLength = listLength
	}

	if val, ok := metadata["listName"]; ok {
		meta.listName = val
	} else {
		return nil, fmt.Errorf("no list name given")
	}

	address := defaultRedisAddress
	if val, ok := metadata["address"]; ok && val != "" {
		address = val
	}

	if val, ok := resolvedEnv[address]; ok {
		meta.address = val
	} else {
		return nil, fmt.Errorf("no address given. Address should be in the format of host:port")
	}

	meta.password = defaultRedisPassword
	if val, ok := metadata["password"]; ok && val != "" {
		if passd, ok := resolvedEnv[val]; ok {
			meta.password = passd
		}
	}

	return &meta, nil
}

// IsActive checks if there is any element in the Redis list
func (s *redisScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := getRedisListLength(
		ctx, s.metadata.address, s.metadata.password, s.metadata.listName)

	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return length > 0, nil
}

func (s *redisScaler) Close() error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetListLengthQty := resource.NewQuantity(int64(s.metadata.targetListLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: listLengthMetricName, TargetAverageValue: targetListLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

// GetMetrics connects to Redis and finds the length of the list
func (s *redisScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	listLen, err := getRedisListLength(ctx, s.metadata.address, s.metadata.password, s.metadata.listName)

	if err != nil {
		log.Errorf("error getting list length %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(listLen, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func getRedisListLength(ctx context.Context, address string, password string, listName string) (int64, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0,
	})

	cmd := client.LLen(listName)

	if cmd.Err() != nil {
		return -1, cmd.Err()
	}

	return cmd.Result()
}

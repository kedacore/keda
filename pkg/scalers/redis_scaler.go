package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/go-redis/redis"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	listLengthMetricName    = "RedisListLength"
	defaultTargetListLength = 5
	defaultRedisAddress     = "redis-master.default.svc.cluster.local:6379"
	defaultRedisPassword    = ""
	defaultDbIdx            = 0
	defaultEnableTLS        = false
)

type redisScaler struct {
	metadata *redisMetadata
}

type redisMetadata struct {
	targetListLength int
	listName         string
	address          string
	password         string
	databaseIndex    int
	enableTLS        bool
}

var redisLog = logf.Log.WithName("redis_scaler")

// NewRedisScaler creates a new redisScaler
func NewRedisScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseRedisMetadata(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %s", err)
	}

	return &redisScaler{
		metadata: meta,
	}, nil
}

func parseRedisMetadata(metadata, resolvedEnv, authParams map[string]string) (*redisMetadata, error) {
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
	if val, ok := authParams["password"]; ok {
		meta.password = val
	} else if val, ok := metadata["password"]; ok && val != "" {
		if passd, ok := resolvedEnv[val]; ok {
			meta.password = passd
		}
	}

	meta.databaseIndex = defaultDbIdx
	if val, ok := metadata["databaseIndex"]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("databaseIndex: parsing error %s", err.Error())
		}
		meta.databaseIndex = int(dbIndex)
	}

	meta.enableTLS = defaultEnableTLS
	if val, ok := metadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("enableTLS parsing error %s", err.Error())
		}
		meta.enableTLS = tls
	}

	return &meta, nil
}

// IsActive checks if there is any element in the Redis list
func (s *redisScaler) IsActive(ctx context.Context) (bool, error) {

	length, err := getRedisListLength(
		ctx, s.metadata.address, s.metadata.password, s.metadata.listName, s.metadata.databaseIndex, s.metadata.enableTLS)

	if err != nil {
		redisLog.Error(err, "error")
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
	listLen, err := getRedisListLength(ctx, s.metadata.address, s.metadata.password, s.metadata.listName, s.metadata.databaseIndex, s.metadata.enableTLS)

	if err != nil {
		redisLog.Error(err, "error getting list length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(listLen, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func getRedisListLength(ctx context.Context, address string, password string, listName string, dbIndex int, enableTLS bool) (int64, error) {
	options := &redis.Options{
		Addr:     address,
		Password: password,
		DB:       dbIndex,
	}

	if enableTLS == true {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: enableTLS,
		}
	}

	client := redis.NewClient(options)

	cmd := client.LLen(listName)

	if cmd.Err() != nil {
		return -1, cmd.Err()
	}
	return cmd.Result()
}

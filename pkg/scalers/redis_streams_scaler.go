package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/go-redis/redis"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// defaults
	defaultTargetPendingEntriesCount = 5
	defaultDBIndex                   = 0

	// metadata names
	pendingEntriesCountMetadata = "pendingEntriesCount"
	streamNameMetadata          = "stream"
	consumerGroupNameMetadata   = "consumerGroup"
	passwordMetadata            = "password"
	databaseIndexMetadata       = "databaseIndex"
	enableTLSMetadata           = "enableTLS"
)

type redisStreamsScaler struct {
	metadata *redisStreamsMetadata
	conn     *redis.Client
}

type redisStreamsMetadata struct {
	targetPendingEntriesCount int
	streamName                string
	consumerGroupName         string
	databaseIndex             int
	connectionInfo            redisConnectionInfo
}

var redisStreamsLog = logf.Log.WithName("redis_streams_scaler")

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseRedisStreamsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis streams metadata: %s", err)
	}

	c, err := getRedisConnection(meta)
	if err != nil {
		return nil, fmt.Errorf("redis connection failed: %s", err)
	}

	return &redisStreamsScaler{
		metadata: meta,
		conn:     c,
	}, nil
}

func getRedisConnection(metadata *redisStreamsMetadata) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     metadata.connectionInfo.address,
		Password: metadata.connectionInfo.password,
		DB:       metadata.databaseIndex,
	}

	if metadata.connectionInfo.enableTLS {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// this does not guarantee successful connection
	c := redis.NewClient(options)

	// confirm if connected
	err := c.Ping().Err()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func parseRedisStreamsMetadata(config *ScalerConfig) (*redisStreamsMetadata, error) {
	connInfo, err := parseRedisAddress(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisStreamsMetadata{
		connectionInfo: connInfo,
	}
	meta.targetPendingEntriesCount = defaultTargetPendingEntriesCount

	if val, ok := config.TriggerMetadata[pendingEntriesCountMetadata]; ok {
		pendingEntriesCount, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing pending entries count %v", err)
		}
		meta.targetPendingEntriesCount = pendingEntriesCount
	} else {
		return nil, fmt.Errorf("missing pending entries count")
	}

	if val, ok := config.TriggerMetadata[streamNameMetadata]; ok {
		meta.streamName = val
	} else {
		return nil, fmt.Errorf("missing redis stream name")
	}

	if val, ok := config.TriggerMetadata[consumerGroupNameMetadata]; ok {
		meta.consumerGroupName = val
	} else {
		return nil, fmt.Errorf("missing redis stream consumer group name")
	}

	meta.databaseIndex = defaultDBIndex
	if val, ok := config.TriggerMetadata[databaseIndexMetadata]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis database index %v", err)
		}
		meta.databaseIndex = int(dbIndex)
	}

	return &meta, nil
}

// IsActive checks if there are pending entries in the 'Pending Entries List' for consumer group of a stream
func (s *redisStreamsScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.getPendingEntriesCount()

	if err != nil {
		redisStreamsLog.Error(err, "error")
		return false, err
	}

	return count > 0, nil
}

func (s *redisStreamsScaler) Close() error {
	return s.conn.Close()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisStreamsScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetPendingEntriesCount := resource.NewQuantity(int64(s.metadata.targetPendingEntriesCount), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "redis-streams", s.metadata.streamName, s.metadata.consumerGroupName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetPendingEntriesCount,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics fetches the number of pending entries for a consumer group in a stream
func (s *redisStreamsScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	pendingEntriesCount, err := s.getPendingEntriesCount()

	if err != nil {
		redisStreamsLog.Error(err, "error fetching pending entries count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(pendingEntriesCount, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *redisStreamsScaler) getPendingEntriesCount() (int64, error) {
	pendingEntries, err := s.conn.XPending(s.metadata.streamName, s.metadata.consumerGroupName).Result()
	if err != nil {
		return -1, err
	}
	return pendingEntries.Count, nil
}

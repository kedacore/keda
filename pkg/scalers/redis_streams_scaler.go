package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

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
	usernameMetadata            = "username"
	passwordMetadata            = "password"
	databaseIndexMetadata       = "databaseIndex"
	enableTLSMetadata           = "enableTLS"
)

type redisStreamsScaler struct {
	metricType               v2.MetricTargetType
	metadata                 *redisStreamsMetadata
	closeFn                  func() error
	getPendingEntriesCountFn func(ctx context.Context) (int64, error)
	logger                   logr.Logger
}

type redisStreamsMetadata struct {
	targetPendingEntriesCount int64
	streamName                string
	consumerGroupName         string
	databaseIndex             int
	connectionInfo            redisConnectionInfo
	scalerIndex               int
}

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(ctx context.Context, isClustered, isSentinel bool, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "redis_streams_scaler")

	if isClustered {
		meta, err := parseRedisStreamsMetadata(config, parseRedisClusterAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis streams metadata: %s", err)
		}
		return createClusteredRedisStreamsScaler(ctx, meta, metricType, logger)
	} else if isSentinel {
		meta, err := parseRedisStreamsMetadata(config, parseRedisSentinelAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis streams metadata: %s", err)
		}
		return createSentinelRedisStreamsScaler(ctx, meta, metricType, logger)
	}
	meta, err := parseRedisStreamsMetadata(config, parseRedisAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis streams metadata: %s", err)
	}
	return createRedisStreamsScaler(ctx, meta, metricType, logger)
}

func createClusteredRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClusterClient(ctx, meta.connectionInfo)
	if err != nil {
		return nil, fmt.Errorf("connection to redis cluster failed: %s", err)
	}

	closeFn := func() error {
		if err := client.Close(); err != nil {
			logger.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	pendingEntriesCountFn := func(ctx context.Context) (int64, error) {
		pendingEntries, err := client.XPending(ctx, meta.streamName, meta.consumerGroupName).Result()
		if err != nil {
			return -1, err
		}
		return pendingEntries.Count, nil
	}

	return &redisStreamsScaler{
		metricType:               metricType,
		metadata:                 meta,
		closeFn:                  closeFn,
		getPendingEntriesCountFn: pendingEntriesCountFn,
		logger:                   logger,
	}, nil
}

func createSentinelRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisSentinelClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis sentinel failed: %s", err)
	}

	return createScaler(client, meta, metricType, logger)
}

func createRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis failed: %s", err)
	}

	return createScaler(client, meta, metricType, logger)
}

func createScaler(client *redis.Client, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	closeFn := func() error {
		if err := client.Close(); err != nil {
			logger.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	pendingEntriesCountFn := func(ctx context.Context) (int64, error) {
		pendingEntries, err := client.XPending(ctx, meta.streamName, meta.consumerGroupName).Result()
		if err != nil {
			return -1, err
		}
		return pendingEntries.Count, nil
	}

	return &redisStreamsScaler{
		metricType:               metricType,
		metadata:                 meta,
		closeFn:                  closeFn,
		getPendingEntriesCountFn: pendingEntriesCountFn,
		logger:                   logger,
	}, nil
}

func parseRedisStreamsMetadata(config *ScalerConfig, parseFn redisAddressParser) (*redisStreamsMetadata, error) {
	connInfo, err := parseFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisStreamsMetadata{
		connectionInfo: connInfo,
	}
	meta.targetPendingEntriesCount = defaultTargetPendingEntriesCount

	if val, ok := config.TriggerMetadata[pendingEntriesCountMetadata]; ok {
		pendingEntriesCount, err := strconv.ParseInt(val, 10, 64)
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
		dbIndex, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis database index %v", err)
		}
		meta.databaseIndex = int(dbIndex)
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive checks if there are pending entries in the 'Pending Entries List' for consumer group of a stream
func (s *redisStreamsScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.getPendingEntriesCountFn(ctx)

	if err != nil {
		s.logger.Error(err, "error")
		return false, err
	}

	return count > 0, nil
}

func (s *redisStreamsScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisStreamsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("redis-streams-%s", s.metadata.streamName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetPendingEntriesCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetrics fetches the number of pending entries for a consumer group in a stream
func (s *redisStreamsScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	pendingEntriesCount, err := s.getPendingEntriesCountFn(ctx)

	if err != nil {
		s.logger.Error(err, "error fetching pending entries count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(pendingEntriesCount))
	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

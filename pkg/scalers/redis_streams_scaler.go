package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type scaleFactor int8

const (
	xPendingFactor scaleFactor = iota + 1
	xLengthFactor
	lagFactor
)

const (
	// defaults
	defaultDBIndex            = 0
	defaultTargetEntries      = 5
	defaultTargetLag          = 5
	defaultActivationLagCount = 0

	// metadata names
	lagMetadata                      = "lagCount"
	pendingEntriesCountMetadata      = "pendingEntriesCount"
	streamLengthMetadata             = "streamLength"
	streamNameMetadata               = "stream"
	consumerGroupNameMetadata        = "consumerGroup"
	usernameMetadata                 = "username"
	passwordMetadata                 = "password"
	databaseIndexMetadata            = "databaseIndex"
	enableTLSMetadata                = "enableTLS"
	activationValueTriggerConfigName = "activationLagCount"
)

type redisStreamsScaler struct {
	metricType        v2.MetricTargetType
	metadata          *redisStreamsMetadata
	closeFn           func() error
	getEntriesCountFn func(ctx context.Context) (int64, error)
	logger            logr.Logger
}

type redisStreamsMetadata struct {
	scaleFactor               scaleFactor
	targetPendingEntriesCount int64
	targetStreamLength        int64
	targetLag                 int64
	streamName                string
	consumerGroupName         string
	databaseIndex             int
	connectionInfo            redisConnectionInfo
	triggerIndex              int
	activationLagCount        int64
}

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(ctx context.Context, isClustered, isSentinel bool, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "redis_streams_scaler")

	if isClustered {
		meta, err := parseRedisStreamsMetadata(config, parseRedisClusterAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis streams metadata: %w", err)
		}
		return createClusteredRedisStreamsScaler(ctx, meta, metricType, logger)
	} else if isSentinel {
		meta, err := parseRedisStreamsMetadata(config, parseRedisSentinelAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis streams metadata: %w", err)
		}
		return createSentinelRedisStreamsScaler(ctx, meta, metricType, logger)
	}
	meta, err := parseRedisStreamsMetadata(config, parseRedisAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis streams metadata: %w", err)
	}
	return createRedisStreamsScaler(ctx, meta, metricType, logger)
}

func createClusteredRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClusterClient(ctx, meta.connectionInfo)

	if err != nil {
		return nil, fmt.Errorf("connection to redis cluster failed: %w", err)
	}

	closeFn := func() error {
		if err := client.Close(); err != nil {
			logger.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	entriesCountFn, err := createEntriesCountFn(client, meta)

	return &redisStreamsScaler{
		metricType:        metricType,
		metadata:          meta,
		closeFn:           closeFn,
		getEntriesCountFn: entriesCountFn,
		logger:            logger,
	}, err
}

func createSentinelRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisSentinelClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis sentinel failed: %w", err)
	}

	return createScaler(client, meta, metricType, logger)
}

func createRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis failed: %w", err)
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

	entriesCountFn, err := createEntriesCountFn(client, meta)

	return &redisStreamsScaler{
		metricType:        metricType,
		metadata:          meta,
		closeFn:           closeFn,
		getEntriesCountFn: entriesCountFn,
		logger:            logger,
	}, err
}

func createEntriesCountFn(client redis.Cmdable, meta *redisStreamsMetadata) (entriesCountFn func(ctx context.Context) (int64, error), err error) {
	switch meta.scaleFactor {
	case xPendingFactor:
		entriesCountFn = func(ctx context.Context) (int64, error) {
			pendingEntries, err := client.XPending(ctx, meta.streamName, meta.consumerGroupName).Result()
			if err != nil {
				return -1, err
			}
			return pendingEntries.Count, nil
		}
	case xLengthFactor:
		entriesCountFn = func(ctx context.Context) (int64, error) {
			entriesLength, err := client.XLen(ctx, meta.streamName).Result()
			if err != nil {
				return -1, err
			}
			return entriesLength, nil
		}
	case lagFactor:
		entriesCountFn = func(ctx context.Context) (int64, error) {
			// Make sure that redis is version 7+, which is required for xinfo lag
			info, err := client.Info(ctx).Result()
			if err != nil {
				err := errors.New("could not find Redis version")
				return -1, err
			}
			infoLines := strings.Split(info, "\n")
			versionFound := false
			for i := 0; i < len(infoLines); i++ {
				line := infoLines[i]
				lineSplit := strings.Split(line, ":")
				if len(lineSplit) > 1 {
					fieldName := lineSplit[0]
					fieldValue := lineSplit[1]
					if fieldName == "redis_version" {
						versionFound = true
						versionNumString := strings.Split(fieldValue, ".")[0]
						versionNum, err := strconv.ParseInt(versionNumString, 10, 64)
						if err != nil {
							err := errors.New("redis version could not be converted to number")
							return -1, err
						}
						if versionNum < int64(7) {
							err := errors.New("redis version 7+ required for lag")
							return -1, err
						}
						break
					}
				}
			}
			if !versionFound {
				err := errors.New("could not find Redis version number")
				return -1, err
			}
			groups, err := client.XInfoGroups(ctx, meta.streamName).Result()

			// If XINFO GROUPS can't find the stream key, it hasn't been created
			// yet. In that case, we return a lag of 0.
			if fmt.Sprint(err) == "ERR no such key" {
				return 0, nil
			}

			// If the stream has been created, then we find the consumer group
			// associated with this scaler and return its lag.
			numGroups := len(groups)
			for i := 0; i < numGroups; i++ {
				group := groups[i]
				if group.Name == meta.consumerGroupName {
					return group.Lag, nil
				}
			}

			// There is an edge case where the Redis producer has set up the
			// stream [meta.streamName], but the consumer group [meta.consumerGroupName]
			// for that stream isn't registered with Redis. In other words, the
			// producer has created messages for the stream, but the consumer group
			// hasn't yet registered itself on Redis because scaling starts with 0
			// consumers. In this case, it's necessary to use XLEN to return what
			// the lag would have been if the consumer group had been created since
			// it's not possible to obtain the lag for a nonexistent consumer
			// group. From here, the consumer group gets instantiated, and scaling
			// again occurs according to XINFO GROUP lag.
			entriesLength, err := client.XLen(ctx, meta.streamName).Result()
			if err != nil {
				return -1, err
			}
			return entriesLength, nil
		}
	default:
		err = fmt.Errorf("unrecognized scale factor %v", meta.scaleFactor)
	}
	return
}

var (
	// ErrRedisMissingStreamName is returned when "stream" is missing.
	ErrRedisMissingStreamName = errors.New("missing redis stream name")
)

func parseRedisStreamsMetadata(config *scalersconfig.ScalerConfig, parseFn redisAddressParser) (*redisStreamsMetadata, error) {
	connInfo, err := parseFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisStreamsMetadata{
		connectionInfo: connInfo,
	}

	err = parseTLSConfigIntoConnectionInfo(config, &meta.connectionInfo)
	if err != nil {
		return nil, err
	}

	if val, ok := config.TriggerMetadata[streamNameMetadata]; ok {
		meta.streamName = val
	} else {
		return nil, ErrRedisMissingStreamName
	}

	meta.activationLagCount = defaultActivationLagCount

	if val, ok := config.TriggerMetadata[consumerGroupNameMetadata]; ok {
		meta.consumerGroupName = val
		if val, ok := config.TriggerMetadata[lagMetadata]; ok {
			meta.scaleFactor = lagFactor
			lag, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing lag: %w", err)
			}
			meta.targetLag = lag

			if val, ok := config.TriggerMetadata[activationValueTriggerConfigName]; ok {
				activationVal, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, errors.New("error while parsing activation lag value")
				}
				meta.activationLagCount = activationVal
			} else {
				err := errors.New("activationLagCount required for Redis lag")
				return nil, err
			}
		} else {
			meta.scaleFactor = xPendingFactor
			meta.targetPendingEntriesCount = defaultTargetEntries
			if val, ok := config.TriggerMetadata[pendingEntriesCountMetadata]; ok {
				pendingEntriesCount, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing pending entries count: %w", err)
				}
				meta.targetPendingEntriesCount = pendingEntriesCount
			}
		}
	} else {
		meta.scaleFactor = xLengthFactor
		meta.targetStreamLength = defaultTargetEntries
		if val, ok := config.TriggerMetadata[streamLengthMetadata]; ok {
			streamLength, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing stream length: %w", err)
			}
			meta.targetStreamLength = streamLength
		}
	}

	meta.databaseIndex = defaultDBIndex
	if val, ok := config.TriggerMetadata[databaseIndexMetadata]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis database index %w", err)
		}
		meta.databaseIndex = int(dbIndex)
	}

	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func (s *redisStreamsScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisStreamsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricValue int64

	switch s.metadata.scaleFactor {
	case xPendingFactor:
		metricValue = s.metadata.targetPendingEntriesCount
	case xLengthFactor:
		metricValue = s.metadata.targetStreamLength
	case lagFactor:
		metricValue = s.metadata.targetLag
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("redis-streams-%s", s.metadata.streamName))),
		},
		Target: GetMetricTarget(s.metricType, metricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity fetches the metric value for a consumer group in a stream
func (s *redisStreamsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricCount, err := s.getEntriesCountFn(ctx)

	if err != nil {
		s.logger.Error(err, "error fetching metric count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(metricCount))
	return []external_metrics.ExternalMetricValue{metric}, metricCount > s.metadata.activationLagCount, nil
}

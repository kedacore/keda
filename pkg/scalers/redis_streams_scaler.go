package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"os/exec"
	"strings"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

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
	defaultDBIndex       = 0
	defaultTargetEntries = 5
	defaultTargetLag     = 5

	// metadata names
	lagMetadata                 = "lagCount"
	pendingEntriesCountMetadata = "pendingEntriesCount"
	streamLengthMetadata        = "streamLength"
	streamNameMetadata          = "stream"
	consumerGroupNameMetadata   = "consumerGroup"
	usernameMetadata            = "username"
	passwordMetadata            = "password"
	databaseIndexMetadata       = "databaseIndex"
	enableTLSMetadata           = "enableTLS"
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
	scalerIndex               int
}

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(ctx context.Context, isClustered, isSentinel bool, config *ScalerConfig) (Scaler, error) {
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
	    // Make sure that redis is version 7+, which is required for xinfo lag
		cmd := exec.Command("redis-cli", "--version")
		out, err := cmd.Output()
		if err != nil {
		  fmt.Println("could not run command: ", err)
		}
		filter_version := strings.Split(string(out[:]), " ") 
		version := filter_version[1]                                      // Extract version
		version_split := strings.Split(version, ".")                      // Extract first number of version string
		version_number, err := strconv.ParseInt(version_split[0], 10, 64)
		if err != nil {
		  fmt.Println("Could not extract redis version number: ", err)
		}
		// assert.GreaterOrEqual(t, int(version_number), 7, "Need Redis version 7 or higher.") // xInfo lag is compatible only with Redis 7+
		if int(version_number) < 7 {
			err := errors.New("Redis version 7+ required for lag")
			return nil, err
		}

		entriesCountFn = func(ctx context.Context) (int64, error) {
			groups, err := client.XInfoGroups(ctx, meta.streamName).Result()
			if err != nil {
				return -1, err
			}
			numGroups := len(groups)
			for i := 0; i < numGroups; i++ {
				group := groups[i]
				if group.Name == meta.consumerGroupName {
					return group.Lag, nil
				}
			}
			err = fmt.Errorf("Stream name does not exist.")
			return int64(-1), err
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

func parseRedisStreamsMetadata(config *ScalerConfig, parseFn redisAddressParser) (*redisStreamsMetadata, error) {
	connInfo, err := parseFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisStreamsMetadata{
		connectionInfo: connInfo,
	}

	meta.connectionInfo.enableTLS = defaultEnableTLS
	if val, ok := config.TriggerMetadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("enableTLS parsing error %w", err)
		}
		meta.connectionInfo.enableTLS = tls
	}

	meta.connectionInfo.unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		parsedVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.connectionInfo.unsafeSsl = parsedVal
	}

	if val, ok := config.TriggerMetadata[streamNameMetadata]; ok {
		meta.streamName = val
	} else {
		return nil, ErrRedisMissingStreamName
	}

	if val, ok := config.TriggerMetadata[consumerGroupNameMetadata]; ok {
		meta.consumerGroupName = val
		if val, ok := config.TriggerMetadata[lagMetadata]; ok {
			meta.scaleFactor = lagFactor
			meta.targetLag = defaultTargetLag
			lag, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing lag: %w", err)
			}
			meta.targetLag = lag
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

	meta.scalerIndex = config.ScalerIndex
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
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("redis-streams-%s", s.metadata.streamName))),
		},
		Target: GetMetricTarget(s.metricType, metricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity fetches the number of pending entries for a consumer group in a stream
func (s *redisStreamsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	pendingEntriesCount, err := s.getEntriesCountFn(ctx)

	if err != nil {
		s.logger.Error(err, "error fetching pending entries count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(pendingEntriesCount))

	return []external_metrics.ExternalMetricValue{metric}, pendingEntriesCount > 0, nil
}

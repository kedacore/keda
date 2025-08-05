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

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type scaleFactor int8

const (
	xPendingFactor scaleFactor = iota + 1
	xLengthFactor
	lagFactor
)

const (
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
	triggerIndex              int
	TargetPendingEntriesCount int64               `keda:"name=pendingEntriesCount, order=triggerMetadata, default=5"`
	TargetStreamLength        int64               `keda:"name=streamLength,        order=triggerMetadata, default=5"`
	TargetLag                 int64               `keda:"name=lagCount,            order=triggerMetadata, optional"`
	StreamName                string              `keda:"name=stream,              order=triggerMetadata"`
	ConsumerGroupName         string              `keda:"name=consumerGroup,       order=triggerMetadata, optional"`
	DatabaseIndex             int                 `keda:"name=databaseIndex,       order=triggerMetadata, optional"`
	ConnectionInfo            redisConnectionInfo `keda:"optional"`
	ActivationLagCount        int64               `keda:"name=activationLagCount,  order=triggerMetadata, default=0"`
	MetadataEnableTLS         string              `keda:"name=enableTLS,           order=triggerMetadata, optional"`
	AuthParamEnableTLS        string              `keda:"name=tls,                 order=authParams, optional"`
}

func (r *redisStreamsMetadata) Validate() error {
	err := validateRedisAddress(&r.ConnectionInfo)
	if err != nil {
		return err
	}

	err = r.ConnectionInfo.SetEnableTLS(r.MetadataEnableTLS, r.AuthParamEnableTLS)
	if err != nil {
		return err
	}
	r.MetadataEnableTLS, r.AuthParamEnableTLS = "", ""

	if r.StreamName == "" {
		return ErrRedisMissingStreamName
	}

	if r.ConsumerGroupName != "" {
		r.TargetStreamLength = 0
		if r.TargetLag != 0 {
			r.scaleFactor = lagFactor
			r.TargetPendingEntriesCount = 0
		} else {
			r.scaleFactor = xPendingFactor
		}
	} else {
		r.scaleFactor = xLengthFactor
		r.TargetPendingEntriesCount = 0
	}

	return nil
}

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(ctx context.Context, isClustered, isSentinel bool, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "redis_streams_scaler")

	meta, err := parseRedisStreamsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis streams metadata: %w", err)
	}

	if isClustered {
		return createClusteredRedisStreamsScaler(ctx, meta, metricType, logger)
	} else if isSentinel {
		return createSentinelRedisStreamsScaler(ctx, meta, metricType, logger)
	}
	return createRedisStreamsScaler(ctx, meta, metricType, logger)
}

func createClusteredRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClusterClient(ctx, meta.ConnectionInfo)

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
	client, err := getRedisSentinelClient(ctx, meta.ConnectionInfo, meta.DatabaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis sentinel failed: %w", err)
	}

	return createScaler(client, meta, metricType, logger)
}

func createRedisStreamsScaler(ctx context.Context, meta *redisStreamsMetadata, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.ConnectionInfo, meta.DatabaseIndex)
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
			pendingEntries, err := client.XPending(ctx, meta.StreamName, meta.ConsumerGroupName).Result()
			if err != nil {
				return -1, err
			}
			return pendingEntries.Count, nil
		}
	case xLengthFactor:
		entriesCountFn = func(ctx context.Context) (int64, error) {
			entriesLength, err := client.XLen(ctx, meta.StreamName).Result()
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
			groups, err := client.XInfoGroups(ctx, meta.StreamName).Result()

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
				if group.Name == meta.ConsumerGroupName {
					return group.Lag, nil
				}
			}

			// There is an edge case where the Redis producer has set up the
			// stream [meta.StreamName], but the consumer group [meta.ConsumerGroupName]
			// for that stream isn't registered with Redis. In other words, the
			// producer has created messages for the stream, but the consumer group
			// hasn't yet registered itself on Redis because scaling starts with 0
			// consumers. In this case, it's necessary to use XLEN to return what
			// the lag would have been if the consumer group had been created since
			// it's not possible to obtain the lag for a nonexistent consumer
			// group. From here, the consumer group gets instantiated, and scaling
			// again occurs according to XINFO GROUP lag.
			entriesLength, err := client.XLen(ctx, meta.StreamName).Result()
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

	// ErrRedisStreamParse is returned when missing parameters or parsing parameters error.
	ErrRedisStreamParse = errors.New("error parsing redis stream metadata")
)

func parseRedisStreamsMetadata(config *scalersconfig.ScalerConfig) (*redisStreamsMetadata, error) {
	meta := &redisStreamsMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing redis stream metadata: %w", err)
	}
	return meta, nil
}

func (s *redisStreamsScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisStreamsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricValue int64

	switch s.metadata.scaleFactor {
	case xPendingFactor:
		metricValue = s.metadata.TargetPendingEntriesCount
	case xLengthFactor:
		metricValue = s.metadata.TargetStreamLength
	case lagFactor:
		metricValue = s.metadata.TargetLag
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("redis-streams-%s", s.metadata.StreamName))),
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
	return []external_metrics.ExternalMetricValue{metric}, metricCount > s.metadata.ActivationLagCount, nil
}

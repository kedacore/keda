package scalers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultListLength           = 5
	defaultActivationListLength = 0
	defaultDBIdx                = 0
	defaultEnableTLS            = false
)

var (
	// ErrRedisNoListName is returned when "listName" is missing from the config.
	ErrRedisNoListName = errors.New("no list name given")

	// ErrRedisNoAddresses is returned when the "addresses" in the connection info is empty.
	ErrRedisNoAddresses = errors.New("no addresses or hosts given. address should be a comma separated list of host:port or set the host/port values")

	// ErrRedisUnequalHostsAndPorts is returned when the number of hosts and ports are unequal.
	ErrRedisUnequalHostsAndPorts = errors.New("not enough hosts or ports given. number of hosts should be equal to the number of ports")

	// ErrRedisParse is returned when "listName" is missing from the config.
	ErrRedisParse = errors.New("error parsing redis metadata")
)

type redisScaler struct {
	metricType      v2.MetricTargetType
	metadata        *redisMetadata
	closeFn         func() error
	getListLengthFn func(context.Context) (int64, error)
	logger          logr.Logger
}

type redisConnectionInfo struct {
	Addresses        []string `keda:"name=address;addresses, order=triggerMetadata;authParams;resolvedEnv"`
	Username         string   `keda:"name=username,          order=triggerMetadata;resolvedEnv;authParams"`
	Password         string   `keda:"name=password,          order=triggerMetadata;resolvedEnv;authParams"`
	SentinelUsername string   `keda:"name=sentinelUsername,  order=triggerMetadata;authParams;resolvedEnv"`
	SentinelPassword string   `keda:"name=sentinelPassword,  order=triggerMetadata;authParams;resolvedEnv"`
	SentinelMaster   string   `keda:"name=sentinelMaster,    order=triggerMetadata;authParams;resolvedEnv"`
	Hosts            []string `keda:"name=host;hosts,        order=triggerMetadata;resolvedEnv;authParams"`
	Ports            []string `keda:"name=port;ports,        order=triggerMetadata;resolvedEnv;authParams"`
	EnableTLS        bool
	UnsafeSsl        bool   `keda:"name=unsafeSsl,   order=triggerMetadata, default=false"`
	Cert             string `keda:"name=Cert;cert,   order=authParams"`
	Key              string `keda:"name=key,         order=authParams"`
	KeyPassword      string `keda:"name=keyPassword, order=authParams"`
	Ca               string `keda:"name=ca,          order=authParams"`
}

type redisMetadata struct {
	ListLength           int64               `keda:"name=listLength,           order=triggerMetadata, default=5"`
	ActivationListLength int64               `keda:"name=activationListLength, order=triggerMetadata, optional"`
	ListName             string              `keda:"name=listName,             order=triggerMetadata"`
	DatabaseIndex        int                 `keda:"name=databaseIndex,        order=triggerMetadata, optional"`
	MetadataEnableTLS    string              `keda:"name=enableTLS,            order=triggerMetadata, optional"`
	AuthParamEnableTLS   string              `keda:"name=tls,                  order=authParams, optional"`
	ConnectionInfo       redisConnectionInfo `keda:"optional"`
	triggerIndex         int
}

func (rci *redisConnectionInfo) SetEnableTLS(metadataEnableTLS string, authParamEnableTLS string) error {
	EnableTLS := defaultEnableTLS

	if metadataEnableTLS != "" && authParamEnableTLS != "" {
		return errors.New("unable to set `tls` in both ScaledObject and TriggerAuthentication together")
	}

	if metadataEnableTLS != "" {
		tls, err := strconv.ParseBool(metadataEnableTLS)
		if err != nil {
			return fmt.Errorf("EnableTLS parsing error %w", err)
		}
		EnableTLS = tls
	}

	// parse tls config defined in auth params
	if authParamEnableTLS != "" {
		switch authParamEnableTLS {
		case stringEnable:
			EnableTLS = true
		case stringDisable:
			EnableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", authParamEnableTLS)
		}
	}
	rci.EnableTLS = EnableTLS
	return nil
}

func (r *redisMetadata) Validate() error {
	err := validateRedisAddress(&r.ConnectionInfo)

	if err != nil {
		return err
	}

	err = r.ConnectionInfo.SetEnableTLS(r.MetadataEnableTLS, r.AuthParamEnableTLS)
	if err == nil {
		r.MetadataEnableTLS, r.AuthParamEnableTLS = "", ""
	}

	return err
}

// NewRedisScaler creates a new redisScaler
func NewRedisScaler(ctx context.Context, isClustered, isSentinel bool, config *scalersconfig.ScalerConfig) (Scaler, error) {
	luaScript := `
		local listName = KEYS[1]
		local listType = redis.call('type', listName).ok
		local cmd = {
			zset = 'zcard',
			set = 'scard',
			list = 'llen',
			hash = 'hlen',
			none = 'llen'
		}

		return redis.call(cmd[listType], listName)
	`

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "redis_scaler")

	meta, err := parseRedisMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %w", err)
	}

	if isClustered {
		return createClusteredRedisScaler(ctx, meta, luaScript, metricType, logger)
	} else if isSentinel {
		return createSentinelRedisScaler(ctx, meta, luaScript, metricType, logger)
	}
	return createRedisScaler(ctx, meta, luaScript, metricType, logger)
}

func createClusteredRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
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

	listLengthFn := func(ctx context.Context) (int64, error) {
		cmd := client.Eval(ctx, script, []string{meta.ListName})
		if cmd.Err() != nil {
			return -1, cmd.Err()
		}

		return cmd.Int64()
	}

	return &redisScaler{
		metricType:      metricType,
		metadata:        meta,
		closeFn:         closeFn,
		getListLengthFn: listLengthFn,
		logger:          logger,
	}, nil
}

func createSentinelRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisSentinelClient(ctx, meta.ConnectionInfo, meta.DatabaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis sentinel failed: %w", err)
	}

	return createRedisScalerWithClient(client, meta, script, metricType, logger), nil
}

func createRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.ConnectionInfo, meta.DatabaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis failed: %w", err)
	}

	return createRedisScalerWithClient(client, meta, script, metricType, logger), nil
}

func createRedisScalerWithClient(client *redis.Client, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) Scaler {
	closeFn := func() error {
		if err := client.Close(); err != nil {
			logger.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	listLengthFn := func(ctx context.Context) (int64, error) {
		cmd := client.Eval(ctx, script, []string{meta.ListName})
		if cmd.Err() != nil {
			return -1, cmd.Err()
		}

		return cmd.Int64()
	}

	return &redisScaler{
		metricType:      metricType,
		metadata:        meta,
		closeFn:         closeFn,
		getListLengthFn: listLengthFn,
		logger:          logger,
	}
}

func parseRedisMetadata(config *scalersconfig.ScalerConfig) (*redisMetadata, error) {
	meta := &redisMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func (s *redisScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := util.NormalizeString(fmt.Sprintf("redis-%s", s.metadata.ListName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.ListLength),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity connects to Redis and finds the length of the list
func (s *redisScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	listLen, err := s.getListLengthFn(ctx)

	if err != nil {
		s.logger.Error(err, "error getting list length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(listLen))

	return []external_metrics.ExternalMetricValue{metric}, listLen > s.metadata.ActivationListLength, nil
}

func validateRedisAddress(c *redisConnectionInfo) error {
	if len(c.Hosts) != 0 && len(c.Ports) != 0 {
		if len(c.Hosts) != len(c.Ports) {
			return ErrRedisUnequalHostsAndPorts
		}
		for i := range c.Hosts {
			c.Addresses = append(c.Addresses, net.JoinHostPort(c.Hosts[i], c.Ports[i]))
		}
	}
	// }

	if len(c.Addresses) == 0 || len(c.Addresses[0]) == 0 {
		return ErrRedisNoAddresses
	}
	return nil
}

func getRedisClusterClient(ctx context.Context, info redisConnectionInfo) (*redis.ClusterClient, error) {
	options := &redis.ClusterOptions{
		Addrs:    info.Addresses,
		Username: info.Username,
		Password: info.Password,
	}
	if info.EnableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.Cert, info.Key, info.KeyPassword, info.Ca, info.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		options.TLSConfig = tlsConfig
	}

	// confirm if connected
	c := redis.NewClusterClient(options)
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return c, nil
}

func getRedisSentinelClient(ctx context.Context, info redisConnectionInfo, dbIndex int) (*redis.Client, error) {
	options := &redis.FailoverOptions{
		Username:         info.Username,
		Password:         info.Password,
		DB:               dbIndex,
		SentinelAddrs:    info.Addresses,
		SentinelUsername: info.SentinelUsername,
		SentinelPassword: info.SentinelPassword,
		MasterName:       info.SentinelMaster,
	}
	if info.EnableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.Cert, info.Key, info.KeyPassword, info.Ca, info.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		options.TLSConfig = tlsConfig
	}

	// confirm if connected
	c := redis.NewFailoverClient(options)
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return c, nil
}

func getRedisClient(ctx context.Context, info redisConnectionInfo, dbIndex int) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     info.Addresses[0],
		Username: info.Username,
		Password: info.Password,
		DB:       dbIndex,
	}
	if info.EnableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.Cert, info.Key, info.KeyPassword, info.Ca, info.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		options.TLSConfig = tlsConfig
	}

	// confirm if connected
	c := redis.NewClient(options)
	err := c.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}
	return c, nil
}

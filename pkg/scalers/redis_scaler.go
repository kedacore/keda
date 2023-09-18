package scalers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

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
)

type redisAddressParser func(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error)

type redisScaler struct {
	metricType      v2.MetricTargetType
	metadata        *redisMetadata
	closeFn         func() error
	getListLengthFn func(context.Context) (int64, error)
	logger          logr.Logger
}

type redisConnectionInfo struct {
	addresses        []string
	username         string
	password         string
	sentinelUsername string
	sentinelPassword string
	sentinelMaster   string
	hosts            []string
	ports            []string
	enableTLS        bool
	unsafeSsl        bool
	cert             string
	key              string
	keyPassword      string
	ca               string
}

type redisMetadata struct {
	listLength           int64
	activationListLength int64
	listName             string
	databaseIndex        int
	connectionInfo       redisConnectionInfo
	scalerIndex          int
}

// NewRedisScaler creates a new redisScaler
func NewRedisScaler(ctx context.Context, isClustered, isSentinel bool, config *ScalerConfig) (Scaler, error) {
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

	if isClustered {
		meta, err := parseRedisMetadata(config, parseRedisClusterAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis metadata: %w", err)
		}
		return createClusteredRedisScaler(ctx, meta, luaScript, metricType, logger)
	} else if isSentinel {
		meta, err := parseRedisMetadata(config, parseRedisSentinelAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis metadata: %w", err)
		}
		return createSentinelRedisScaler(ctx, meta, luaScript, metricType, logger)
	}

	meta, err := parseRedisMetadata(config, parseRedisAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %w", err)
	}

	return createRedisScaler(ctx, meta, luaScript, metricType, logger)
}

func createClusteredRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
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

	listLengthFn := func(ctx context.Context) (int64, error) {
		cmd := client.Eval(ctx, script, []string{meta.listName})
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
	client, err := getRedisSentinelClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis sentinel failed: %w", err)
	}

	return createRedisScalerWithClient(client, meta, script, metricType, logger), nil
}

func createRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.connectionInfo, meta.databaseIndex)
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
		cmd := client.Eval(ctx, script, []string{meta.listName})
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

func parseTLSConfigIntoConnectionInfo(config *ScalerConfig, connInfo *redisConnectionInfo) error {
	enableTLS := defaultEnableTLS
	if val, ok := config.TriggerMetadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("enableTLS parsing error %w", err)
		}
		enableTLS = tls
	}

	connInfo.unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		parsedVal, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		connInfo.unsafeSsl = parsedVal
	}

	// parse tls config defined in auth params
	if val, ok := config.AuthParams["tls"]; ok {
		val = strings.TrimSpace(val)
		if enableTLS {
			return errors.New("unable to set `tls` in both ScaledObject and TriggerAuthentication together")
		}
		switch val {
		case stringEnable:
			enableTLS = true
		case stringDisable:
			enableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", val)
		}
	}
	if enableTLS {
		certGiven := config.AuthParams["cert"] != ""
		keyGiven := config.AuthParams["key"] != ""
		if certGiven && !keyGiven {
			return errors.New("key must be provided with cert")
		}
		if keyGiven && !certGiven {
			return errors.New("cert must be provided with key")
		}
		connInfo.ca = config.AuthParams["ca"]
		connInfo.cert = config.AuthParams["cert"]
		connInfo.key = config.AuthParams["key"]
		if value, found := config.AuthParams["keyPassword"]; found {
			connInfo.keyPassword = value
		} else {
			connInfo.keyPassword = ""
		}
	}
	connInfo.enableTLS = enableTLS
	return nil
}

func parseRedisMetadata(config *ScalerConfig, parserFn redisAddressParser) (*redisMetadata, error) {
	connInfo, err := parserFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisMetadata{
		connectionInfo: connInfo,
	}

	err = parseTLSConfigIntoConnectionInfo(config, &meta.connectionInfo)
	if err != nil {
		return nil, err
	}

	meta.listLength = defaultListLength
	if val, ok := config.TriggerMetadata["listLength"]; ok {
		listLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("list length parsing error: %w", err)
		}
		meta.listLength = listLength
	}

	meta.activationListLength = defaultActivationListLength
	if val, ok := config.TriggerMetadata["activationListLength"]; ok {
		activationListLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationListLength parsing error %w", err)
		}
		meta.activationListLength = activationListLength
	}

	if val, ok := config.TriggerMetadata["listName"]; ok {
		meta.listName = val
	} else {
		return nil, ErrRedisNoListName
	}

	meta.databaseIndex = defaultDBIdx
	if val, ok := config.TriggerMetadata["databaseIndex"]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("databaseIndex: parsing error %w", err)
		}
		meta.databaseIndex = int(dbIndex)
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *redisScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := util.NormalizeString(fmt.Sprintf("redis-%s", s.metadata.listName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.listLength),
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

	return []external_metrics.ExternalMetricValue{metric}, listLen > s.metadata.activationListLength, nil
}

func parseRedisAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info := redisConnectionInfo{}
	switch {
	case authParams["address"] != "":
		info.addresses = append(info.addresses, authParams["address"])
	case metadata["address"] != "":
		info.addresses = append(info.addresses, metadata["address"])
	case metadata["addressFromEnv"] != "":
		info.addresses = append(info.addresses, resolvedEnv[metadata["addressFromEnv"]])
	default:
		switch {
		case authParams["host"] != "":
			info.hosts = append(info.hosts, authParams["host"])
		case metadata["host"] != "":
			info.hosts = append(info.hosts, metadata["host"])
		case metadata["hostFromEnv"] != "":
			info.hosts = append(info.hosts, resolvedEnv[metadata["hostFromEnv"]])
		}

		switch {
		case authParams["port"] != "":
			info.ports = append(info.ports, authParams["port"])
		case metadata["port"] != "":
			info.ports = append(info.ports, metadata["port"])
		case metadata["portFromEnv"] != "":
			info.ports = append(info.ports, resolvedEnv[metadata["portFromEnv"]])
		}

		if len(info.hosts) != 0 && len(info.ports) != 0 {
			info.addresses = append(info.addresses, net.JoinHostPort(info.hosts[0], info.ports[0]))
		}
	}

	if len(info.addresses) == 0 || len(info.addresses[0]) == 0 {
		return info, fmt.Errorf("no address or host given. address should be in the format of host:port or you should set the host/port values")
	}

	switch {
	case authParams["username"] != "":
		info.username = authParams["username"]
	case metadata["username"] != "":
		info.username = metadata["username"]
	case metadata["usernameFromEnv"] != "":
		info.username = resolvedEnv[metadata["usernameFromEnv"]]
	}

	if authParams["password"] != "" {
		info.password = authParams["password"]
	} else if metadata["passwordFromEnv"] != "" {
		info.password = resolvedEnv[metadata["passwordFromEnv"]]
	}

	return info, nil
}

func parseRedisMultipleAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info := redisConnectionInfo{}
	switch {
	case authParams["addresses"] != "":
		info.addresses = splitAndTrim(authParams["addresses"])
	case metadata["addresses"] != "":
		info.addresses = splitAndTrim(metadata["addresses"])
	case metadata["addressesFromEnv"] != "":
		info.addresses = splitAndTrim(resolvedEnv[metadata["addressesFromEnv"]])
	default:
		switch {
		case authParams["hosts"] != "":
			info.hosts = splitAndTrim(authParams["hosts"])
		case metadata["hosts"] != "":
			info.hosts = splitAndTrim(metadata["hosts"])
		case metadata["hostsFromEnv"] != "":
			info.hosts = splitAndTrim(resolvedEnv[metadata["hostsFromEnv"]])
		}

		switch {
		case authParams["ports"] != "":
			info.ports = splitAndTrim(authParams["ports"])
		case metadata["ports"] != "":
			info.ports = splitAndTrim(metadata["ports"])
		case metadata["portsFromEnv"] != "":
			info.ports = splitAndTrim(resolvedEnv[metadata["portsFromEnv"]])
		}

		if len(info.hosts) != 0 && len(info.ports) != 0 {
			if len(info.hosts) != len(info.ports) {
				return info, ErrRedisUnequalHostsAndPorts
			}
			for i := range info.hosts {
				info.addresses = append(info.addresses, net.JoinHostPort(info.hosts[i], info.ports[i]))
			}
		}
	}

	if len(info.addresses) == 0 {
		return info, ErrRedisNoAddresses
	}

	return info, nil
}

func parseRedisClusterAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info, err := parseRedisMultipleAddress(metadata, resolvedEnv, authParams)
	if err != nil {
		return redisConnectionInfo{}, err
	}

	switch {
	case authParams["username"] != "":
		info.username = authParams["username"]
	case metadata["username"] != "":
		info.username = metadata["username"]
	case metadata["usernameFromEnv"] != "":
		info.username = resolvedEnv[metadata["usernameFromEnv"]]
	}

	if authParams["password"] != "" {
		info.password = authParams["password"]
	} else if metadata["passwordFromEnv"] != "" {
		info.password = resolvedEnv[metadata["passwordFromEnv"]]
	}

	return info, nil
}

func parseRedisSentinelAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info, err := parseRedisMultipleAddress(metadata, resolvedEnv, authParams)
	if err != nil {
		return redisConnectionInfo{}, err
	}

	switch {
	case authParams["username"] != "":
		info.username = authParams["username"]
	case metadata["username"] != "":
		info.username = metadata["username"]
	case metadata["usernameFromEnv"] != "":
		info.username = resolvedEnv[metadata["usernameFromEnv"]]
	}

	if authParams["password"] != "" {
		info.password = authParams["password"]
	} else if metadata["passwordFromEnv"] != "" {
		info.password = resolvedEnv[metadata["passwordFromEnv"]]
	}

	switch {
	case authParams["sentinelUsername"] != "":
		info.sentinelUsername = authParams["sentinelUsername"]
	case metadata["sentinelUsername"] != "":
		info.sentinelUsername = metadata["sentinelUsername"]
	case metadata["sentinelUsernameFromEnv"] != "":
		info.sentinelUsername = resolvedEnv[metadata["sentinelUsernameFromEnv"]]
	}

	if authParams["sentinelPassword"] != "" {
		info.sentinelPassword = authParams["sentinelPassword"]
	} else if metadata["sentinelPasswordFromEnv"] != "" {
		info.sentinelPassword = resolvedEnv[metadata["sentinelPasswordFromEnv"]]
	}

	switch {
	case authParams["sentinelMaster"] != "":
		info.sentinelMaster = authParams["sentinelMaster"]
	case metadata["sentinelMaster"] != "":
		info.sentinelMaster = metadata["sentinelMaster"]
	case metadata["sentinelMasterFromEnv"] != "":
		info.sentinelMaster = resolvedEnv[metadata["sentinelMasterFromEnv"]]
	}

	return info, nil
}

func getRedisClusterClient(ctx context.Context, info redisConnectionInfo) (*redis.ClusterClient, error) {
	options := &redis.ClusterOptions{
		Addrs:    info.addresses,
		Username: info.username,
		Password: info.password,
	}
	if info.enableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.cert, info.key, info.keyPassword, info.ca, info.unsafeSsl)
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
		Username:         info.username,
		Password:         info.password,
		DB:               dbIndex,
		SentinelAddrs:    info.addresses,
		SentinelUsername: info.sentinelUsername,
		SentinelPassword: info.sentinelPassword,
		MasterName:       info.sentinelMaster,
	}
	if info.enableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.cert, info.key, info.keyPassword, info.ca, info.unsafeSsl)
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
		Addr:     info.addresses[0],
		Username: info.username,
		Password: info.password,
		DB:       dbIndex,
	}
	if info.enableTLS {
		tlsConfig, err := util.NewTLSConfigWithPassword(info.cert, info.key, info.keyPassword, info.ca, info.unsafeSsl)
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

// Splits a string separated by comma and trims space from all the elements.
func splitAndTrim(s string) []string {
	x := strings.Split(s, ",")
	for i := range x {
		x[i] = strings.Trim(x[i], " ")
	}
	return x
}

package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultListLength           = 5
	defaultActivationListLength = 0
	defaultDBIdx                = 0
	defaultEnableTLS            = false
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
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "redis_scaler")

	if isClustered {
		meta, err := parseRedisMetadata(config, parseRedisClusterAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis metadata: %s", err)
		}
		return createClusteredRedisScaler(ctx, meta, luaScript, metricType, logger)
	} else if isSentinel {
		meta, err := parseRedisMetadata(config, parseRedisSentinelAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis metadata: %s", err)
		}
		return createSentinelRedisScaler(ctx, meta, luaScript, metricType, logger)
	}

	meta, err := parseRedisMetadata(config, parseRedisAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %s", err)
	}
	return createRedisScaler(ctx, meta, luaScript, metricType, logger)
}

func createClusteredRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
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
		return nil, fmt.Errorf("connection to redis sentinel failed: %s", err)
	}

	return createRedisScalerWithClient(client, meta, script, metricType, logger), nil
}

func createRedisScaler(ctx context.Context, meta *redisMetadata, script string, metricType v2.MetricTargetType, logger logr.Logger) (Scaler, error) {
	client, err := getRedisClient(ctx, meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis failed: %s", err)
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
	}
}

func parseRedisMetadata(config *ScalerConfig, parserFn redisAddressParser) (*redisMetadata, error) {
	connInfo, err := parserFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisMetadata{
		connectionInfo: connInfo,
	}

	meta.listLength = defaultListLength
	if val, ok := config.TriggerMetadata["listLength"]; ok {
		listLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("list length parsing error %s", err.Error())
		}
		meta.listLength = listLength
	}

	meta.activationListLength = defaultActivationListLength
	if val, ok := config.TriggerMetadata["activationListLength"]; ok {
		activationListLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationListLength parsing error %s", err.Error())
		}
		meta.activationListLength = activationListLength
	}

	if val, ok := config.TriggerMetadata["listName"]; ok {
		meta.listName = val
	} else {
		return nil, fmt.Errorf("no list name given")
	}

	meta.databaseIndex = defaultDBIdx
	if val, ok := config.TriggerMetadata["databaseIndex"]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("databaseIndex: parsing error %s", err.Error())
		}
		meta.databaseIndex = int(dbIndex)
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive checks if there is any element in the Redis list
func (s *redisScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.getListLengthFn(ctx)

	if err != nil {
		s.logger.Error(err, "error")
		return false, err
	}

	return length > s.metadata.activationListLength, nil
}

func (s *redisScaler) Close(context.Context) error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("redis-%s", s.metadata.listName))
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

// GetMetrics connects to Redis and finds the length of the list
func (s *redisScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	listLen, err := s.getListLengthFn(ctx)

	if err != nil {
		s.logger.Error(err, "error getting list length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(listLen))

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
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
			info.addresses = append(info.addresses, fmt.Sprintf("%s:%s", info.hosts[0], info.ports[0]))
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

	info.enableTLS = defaultEnableTLS
	if val, ok := metadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return info, fmt.Errorf("enableTLS parsing error %s", err.Error())
		}
		info.enableTLS = tls
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
				return info, fmt.Errorf("not enough hosts or ports given. number of hosts should be equal to the number of ports")
			}
			for i := range info.hosts {
				info.addresses = append(info.addresses, fmt.Sprintf("%s:%s", info.hosts[i], info.ports[i]))
			}
		}
	}

	if len(info.addresses) == 0 {
		return info, fmt.Errorf("no addresses or hosts given. address should be a comma separated list of host:port or set the host/port values")
	}

	return info, nil
}

func parseRedisClusterAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info, err := parseRedisMultipleAddress(metadata, resolvedEnv, authParams)
	if err != nil {
		return info, err
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

	info.enableTLS = defaultEnableTLS
	if val, ok := metadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return info, fmt.Errorf("enableTLS parsing error %s", err.Error())
		}
		info.enableTLS = tls
	}

	return info, nil
}

func parseRedisSentinelAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info, err := parseRedisMultipleAddress(metadata, resolvedEnv, authParams)
	if err != nil {
		return info, err
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

	info.enableTLS = defaultEnableTLS
	if val, ok := metadata["enableTLS"]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return info, fmt.Errorf("enableTLS parsing error %s", err.Error())
		}
		info.enableTLS = tls
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
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: info.enableTLS,
		}
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
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: info.enableTLS,
		}
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
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: info.enableTLS,
		}
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

package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"

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
	defaultTargetListLength = 5
	defaultDBIdx            = 0
	defaultEnableTLS        = false
)

type redisAddressParser func(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error)

type redisScaler struct {
	metadata        *redisMetadata
	closeFn         func() error
	getListLengthFn func() (int64, error)
}

type redisConnectionInfo struct {
	addresses []string
	password  string
	hosts     []string
	ports     []string
	enableTLS bool
}

type redisMetadata struct {
	targetListLength int
	listName         string
	databaseIndex    int
	connectionInfo   redisConnectionInfo
}

var redisLog = logf.Log.WithName("redis_scaler")

// NewRedisScaler creates a new redisScaler
func NewRedisScaler(isClustered bool, config *ScalerConfig) (Scaler, error) {
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
	if isClustered {
		meta, err := parseRedisMetadata(config, parseRedisClusterAddress)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis metadata: %s", err)
		}
		return createClusteredRedisScaler(meta, luaScript)
	}
	meta, err := parseRedisMetadata(config, parseRedisAddress)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %s", err)
	}
	return createRedisScaler(meta, luaScript)
}

func createClusteredRedisScaler(meta *redisMetadata, script string) (Scaler, error) {
	client, err := getRedisClusterClient(meta.connectionInfo)
	if err != nil {
		return nil, fmt.Errorf("connection to redis cluster failed: %s", err)
	}

	closeFn := func() error {
		if err := client.Close(); err != nil {
			redisLog.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	listLengthFn := func() (int64, error) {
		cmd := client.Eval(script, []string{meta.listName})
		if cmd.Err() != nil {
			return -1, cmd.Err()
		}

		return cmd.Int64()
	}

	return &redisScaler{
		metadata:        meta,
		closeFn:         closeFn,
		getListLengthFn: listLengthFn,
	}, nil
}

func createRedisScaler(meta *redisMetadata, script string) (Scaler, error) {
	client, err := getRedisClient(meta.connectionInfo, meta.databaseIndex)
	if err != nil {
		return nil, fmt.Errorf("connection to redis failed: %s", err)
	}

	closeFn := func() error {
		if err := client.Close(); err != nil {
			redisLog.Error(err, "error closing redis client")
			return err
		}
		return nil
	}

	listLengthFn := func() (int64, error) {
		cmd := client.Eval(script, []string{meta.listName})
		if cmd.Err() != nil {
			return -1, cmd.Err()
		}

		return cmd.Int64()
	}

	return &redisScaler{
		metadata:        meta,
		closeFn:         closeFn,
		getListLengthFn: listLengthFn,
	}, nil
}

func parseRedisMetadata(config *ScalerConfig, parserFn redisAddressParser) (*redisMetadata, error) {
	connInfo, err := parserFn(config.TriggerMetadata, config.ResolvedEnv, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta := redisMetadata{
		connectionInfo: connInfo,
	}
	meta.targetListLength = defaultTargetListLength

	if val, ok := config.TriggerMetadata["listLength"]; ok {
		listLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("list length parsing error %s", err.Error())
		}
		meta.targetListLength = listLength
	}

	if val, ok := config.TriggerMetadata["listName"]; ok {
		meta.listName = val
	} else {
		return nil, fmt.Errorf("no list name given")
	}

	meta.databaseIndex = defaultDBIdx
	if val, ok := config.TriggerMetadata["databaseIndex"]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("databaseIndex: parsing error %s", err.Error())
		}
		meta.databaseIndex = int(dbIndex)
	}

	return &meta, nil
}

// IsActive checks if there is any element in the Redis list
func (s *redisScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.getListLengthFn()

	if err != nil {
		redisLog.Error(err, "error")
		return false, err
	}

	return length > 0, nil
}

func (s *redisScaler) Close() error {
	return s.closeFn()
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *redisScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetListLengthQty := resource.NewQuantity(int64(s.metadata.targetListLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "redis", s.metadata.listName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetListLengthQty,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics connects to Redis and finds the length of the list
func (s *redisScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	listLen, err := s.getListLengthFn()

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

func parseRedisClusterAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
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

func getRedisClusterClient(info redisConnectionInfo) (*redis.ClusterClient, error) {
	options := &redis.ClusterOptions{
		Addrs:    info.addresses,
		Password: info.password,
	}
	if info.enableTLS {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: info.enableTLS,
		}
	}

	// confirm if connected
	c := redis.NewClusterClient(options)
	err := c.Ping().Err()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func getRedisClient(info redisConnectionInfo, dbIndex int) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     info.addresses[0],
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
	err := c.Ping().Err()
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

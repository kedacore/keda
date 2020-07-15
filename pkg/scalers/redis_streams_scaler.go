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
	pendingEntriesCountMetricName = "RedisStreamPendingEntriesCount"

	// defaults
	defaultTargetPendingEntriesCount = 5
	defaultAddress                   = "redis-master.default.svc.cluster.local:6379"
	defaultPassword                  = ""
	defaultDbIndex                   = 0
	defaultTLS                       = false
	defaultRedisHost                 = ""
	defaultRedisPort                 = ""

	// metadata names
	pendingEntriesCountMetadata = "pendingEntriesCount"
	streamNameMetadata          = "stream"
	consumerGroupNameMetadata   = "consumerGroup"
	addressMetadata             = "address"
	hostMetadata                = "host"
	portMetadata                = "port"
	passwordMetadata            = "password"
	databaseIndexMetadata       = "databaseIndex"
	enableTLSMetadata           = "enableTLS"

	// error
	missingRedisAddressOrHostPortInfo = "address or host missing. please provide redis address should in host:port format or set the host/port values"
)

type redisStreamsScaler struct {
	metadata *redisStreamsMetadata
	conn     *redis.Client
}

type redisStreamsMetadata struct {
	targetPendingEntriesCount int
	streamName                string
	consumerGroupName         string
	address                   string
	password                  string
	host                      string
	port                      string
	databaseIndex             int
	enableTLS                 bool
}

var redisStreamsLog = logf.Log.WithName("redis_streams_scaler")

// NewRedisStreamsScaler creates a new redisStreamsScaler
func NewRedisStreamsScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseRedisStreamsMetadata(metadata, resolvedEnv, authParams)
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
		Addr:     metadata.address,
		Password: metadata.password,
		DB:       metadata.databaseIndex,
	}

	if metadata.enableTLS == true {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// this does not guarentee successful connection
	c := redis.NewClient(options)

	// confirm if connected
	err := c.Ping().Err()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func parseRedisStreamsMetadata(metadata, resolvedEnv, authParams map[string]string) (*redisStreamsMetadata, error) {
	meta := redisStreamsMetadata{}
	meta.targetPendingEntriesCount = defaultTargetPendingEntriesCount

	if val, ok := metadata[pendingEntriesCountMetadata]; ok {
		pendingEntriesCount, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing pending entries count %v", err)
		}
		meta.targetPendingEntriesCount = pendingEntriesCount
	} else {
		return nil, fmt.Errorf("missing pending entries count")
	}

	if val, ok := metadata[streamNameMetadata]; ok {
		meta.streamName = val
	} else {
		return nil, fmt.Errorf("missing redis stream name")
	}

	if val, ok := metadata[consumerGroupNameMetadata]; ok {
		meta.consumerGroupName = val
	} else {
		return nil, fmt.Errorf("missing redis stream consumer group name")
	}

	address := defaultAddress
	host := defaultRedisHost
	port := defaultRedisPort
	if val, ok := metadata[addressMetadata]; ok && val != "" {
		address = val
	} else {
		if val, ok := metadata[hostMetadata]; ok && val != "" {
			host = val
		} else {
			return nil, fmt.Errorf(missingRedisAddressOrHostPortInfo)
		}
		if val, ok := metadata[portMetadata]; ok && val != "" {
			port = val
		} else {
			return nil, fmt.Errorf(missingRedisAddressOrHostPortInfo)
		}
	}

	if val, ok := resolvedEnv[address]; ok {
		meta.address = val
	} else {
		if val, ok := resolvedEnv[host]; ok {
			meta.host = val
		} else {
			return nil, fmt.Errorf(missingRedisAddressOrHostPortInfo)
		}

		if val, ok := resolvedEnv[port]; ok {
			meta.port = val
		} else {
			return nil, fmt.Errorf(missingRedisAddressOrHostPortInfo)
		}
		meta.address = fmt.Sprintf("%s:%s", meta.host, meta.port)
	}

	meta.password = defaultPassword
	if val, ok := authParams[passwordMetadata]; ok {
		meta.password = val
	} else if val, ok := metadata[passwordMetadata]; ok && val != "" {
		if passd, ok := resolvedEnv[val]; ok {
			meta.password = passd
		}
	}

	meta.databaseIndex = defaultDbIndex
	if val, ok := metadata[databaseIndexMetadata]; ok {
		dbIndex, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing redis database index %v", err)
		}
		meta.databaseIndex = int(dbIndex)
	}

	meta.enableTLS = defaultTLS
	if val, ok := metadata[enableTLSMetadata]; ok {
		tls, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing enableTLS %v", err)
		}
		meta.enableTLS = tls
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
func (s *redisStreamsScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {

	targetPendingEntriesCount := resource.NewQuantity(int64(s.metadata.targetPendingEntriesCount), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: pendingEntriesCountMetricName, TargetAverageValue: targetPendingEntriesCount}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
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

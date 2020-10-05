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

	kedautil "github.com/kedacore/keda/pkg/util"
)

const (
	defaultTargetListLength = 5
	defaultDbIdx            = 0
	defaultEnableTLS        = false
)

type redisScaler struct {
	metadata *redisMetadata
	client   *redis.Client
}

type redisConnectionInfo struct {
	address   string
	password  string
	host      string
	port      string
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
func NewRedisScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseRedisMetadata(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %s", err)
	}
	options := &redis.Options{
		Addr:     meta.connectionInfo.address,
		Password: meta.connectionInfo.password,
		DB:       meta.databaseIndex,
	}

	if meta.connectionInfo.enableTLS {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: meta.connectionInfo.enableTLS,
		}
	}

	return &redisScaler{
		metadata: meta,
		client:   redis.NewClient(options),
	}, nil
}

func parseRedisMetadata(metadata, resolvedEnv, authParams map[string]string) (*redisMetadata, error) {
	connInfo, err := parseRedisAddress(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, err
	}
	meta := redisMetadata{
		connectionInfo: connInfo,
	}
	meta.targetListLength = defaultTargetListLength

	if val, ok := metadata["listLength"]; ok {
		listLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("List length parsing error %s", err.Error())
		}
		meta.targetListLength = listLength
	}

	if val, ok := metadata["listName"]; ok {
		meta.listName = val
	} else {
		return nil, fmt.Errorf("no list name given")
	}

	meta.databaseIndex = defaultDbIdx
	if val, ok := metadata["databaseIndex"]; ok {
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
	length, err := getRedisListLength(ctx, s.client, s.metadata.listName)

	if err != nil {
		redisLog.Error(err, "error")
		return false, err
	}

	return length > 0, nil
}

func (s *redisScaler) Close() error {
	if s.client != nil {
		err := s.client.Close()
		if err != nil {
			redisLog.Error(err, "error closing redis client")
			return err
		}
	}

	return nil
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
	listLen, err := getRedisListLength(ctx, s.client, s.metadata.listName)

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

func getRedisListLength(ctx context.Context, client *redis.Client, listName string) (int64, error) {
	listType := client.Type(listName)
	if listType.Err() != nil {
		return -1, listType.Err()
	}

	var cmd *redis.IntCmd
	switch listType.Val() {
	case "list", "none":
		cmd = client.LLen(listName)
	case "set":
		cmd = client.SCard(listName)
	case "hash":
		cmd = client.HLen(listName)
	case "zset":
		cmd = client.ZCard(listName)
	default:
		cmd = nil
	}

	if cmd == nil {
		return -1, fmt.Errorf("list must be of type: list, none, set, hash or zset but was %s", listType.Val())
	}
	if cmd.Err() != nil {
		return -1, cmd.Err()
	}
	return cmd.Result()
}

func parseRedisAddress(metadata, resolvedEnv, authParams map[string]string) (redisConnectionInfo, error) {
	info := redisConnectionInfo{}
	if authParams["address"] != "" {
		info.address = authParams["address"]
	} else if metadata["address"] != "" {
		info.address = metadata["address"]
	} else if metadata["addressFromEnv"] != "" {
		info.address = resolvedEnv[metadata["addressFromEnv"]]
	} else {
		if authParams["host"] != "" {
			info.host = authParams["host"]
		} else if metadata["host"] != "" {
			info.host = metadata["host"]
		} else if metadata["hostFromEnv"] != "" {
			info.host = resolvedEnv[metadata["hostFromEnv"]]
		}

		if authParams["port"] != "" {
			info.port = authParams["port"]
		} else if metadata["port"] != "" {
			info.port = metadata["port"]
		} else if metadata["portFromEnv"] != "" {
			info.port = resolvedEnv[metadata["portFromEnv"]]
		}

		if len(info.host) != 0 && len(info.port) != 0 {
			info.address = fmt.Sprintf("%s:%s", info.host, info.port)
		}
	}

	if len(info.address) == 0 {
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

package scalers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gocql/gocql"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// cassandraScaler exposes a data pointer to CassandraMetadata and gocql.Session connection.
type cassandraScaler struct {
	metricType v2.MetricTargetType
	metadata   *CassandraMetadata
	session    *gocql.Session
	logger     logr.Logger
}

// CassandraMetadata defines metadata used by KEDA to query a Cassandra table.
type CassandraMetadata struct {
	username                   string
	password                   string
	clusterIPAddress           string
	port                       int
	consistency                gocql.Consistency
	protocolVersion            int
	keyspace                   string
	query                      string
	targetQueryValue           int64
	activationTargetQueryValue int64
	triggerIndex               int
}

// NewCassandraScaler creates a new Cassandra scaler.
func NewCassandraScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "cassandra_scaler")

	meta, err := parseCassandraMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cassandra metadata: %w", err)
	}

	session, err := newCassandraSession(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing cassandra session: %w", err)
	}

	return &cassandraScaler{
		metricType: metricType,
		metadata:   meta,
		session:    session,
		logger:     logger,
	}, nil
}

// parseCassandraMetadata parses the metadata and returns a CassandraMetadata or an error if the ScalerConfig is invalid.
func parseCassandraMetadata(config *ScalerConfig) (*CassandraMetadata, error) {
	meta := CassandraMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["targetQueryValue"]; ok {
		targetQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("targetQueryValue parsing error %w", err)
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		if config.AsMetricSource {
			meta.targetQueryValue = 0
		} else {
			return nil, fmt.Errorf("no targetQueryValue given")
		}
	}

	meta.activationTargetQueryValue = 0
	if val, ok := config.TriggerMetadata["activationTargetQueryValue"]; ok {
		activationTargetQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetQueryValue parsing error %w", err)
		}
		meta.activationTargetQueryValue = activationTargetQueryValue
	}

	if val, ok := config.TriggerMetadata["username"]; ok {
		meta.username = val
	} else {
		return nil, fmt.Errorf("no username given")
	}

	if val, ok := config.TriggerMetadata["port"]; ok {
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("port parsing error %w", err)
		}
		meta.port = port
	}

	if val, ok := config.TriggerMetadata["clusterIPAddress"]; ok {
		splitval := strings.Split(val, ":")
		port := splitval[len(splitval)-1]

		_, err := strconv.Atoi(port)
		switch {
		case err == nil:
			meta.clusterIPAddress = val
		case meta.port > 0:
			meta.clusterIPAddress = net.JoinHostPort(val, fmt.Sprintf("%d", meta.port))
		default:
			return nil, fmt.Errorf("no port given")
		}
	} else {
		return nil, fmt.Errorf("no cluster IP address given")
	}

	if val, ok := config.TriggerMetadata["protocolVersion"]; ok {
		protocolVersion, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("protocolVersion parsing error %w", err)
		}
		meta.protocolVersion = protocolVersion
	} else {
		meta.protocolVersion = 4
	}

	if val, ok := config.TriggerMetadata["consistency"]; ok {
		meta.consistency = gocql.ParseConsistency(val)
	} else {
		meta.consistency = gocql.One
	}

	if val, ok := config.TriggerMetadata["keyspace"]; ok {
		meta.keyspace = val
	} else {
		return nil, fmt.Errorf("no keyspace given")
	}
	if val, ok := config.AuthParams["password"]; ok {
		meta.password = val
	} else {
		return nil, fmt.Errorf("no password given")
	}

	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

// newCassandraSession returns a new Cassandra session for the provided CassandraMetadata.
func newCassandraSession(meta *CassandraMetadata, logger logr.Logger) (*gocql.Session, error) {
	cluster := gocql.NewCluster(meta.clusterIPAddress)
	cluster.ProtoVersion = meta.protocolVersion
	cluster.Consistency = meta.consistency
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: meta.username,
		Password: meta.password,
	}

	session, err := cluster.CreateSession()
	if err != nil {
		logger.Error(err, "found error creating session")
		return nil, err
	}

	return session, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler.
func (s *cassandraScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("cassandra-%s", s.metadata.keyspace))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns a value for a supported metric or an error if there is a problem getting the metric.
func (s *cassandraScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.GetQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting cassandra: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(num))

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetQueryValue, nil
}

// GetQueryResult returns the result of the scaler query.
func (s *cassandraScaler) GetQueryResult(ctx context.Context) (int64, error) {
	var value int64
	if err := s.session.Query(s.metadata.query).WithContext(ctx).Scan(&value); err != nil {
		if err != gocql.ErrNotFound {
			s.logger.Error(err, "query failed")
			return 0, err
		}
	}

	return value, nil
}

// Close closes the Cassandra session connection.
func (s *cassandraScaler) Close(_ context.Context) error {
	if s.session != nil {
		s.session.Close()
	}
	return nil
}

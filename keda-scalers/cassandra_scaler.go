package scalers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gocql/gocql"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type cassandraScaler struct {
	metricType v2.MetricTargetType
	metadata   cassandraMetadata
	session    *gocql.Session
	logger     logr.Logger
}

type cassandraMetadata struct {
	Username                   string `keda:"name=username,                   order=triggerMetadata"`
	Password                   string `keda:"name=password,                   order=authParams"`
	TLS                        string `keda:"name=tls,                        order=authParams, enum=enable;disable, default=disable"`
	Cert                       string `keda:"name=cert,                       order=authParams, optional"`
	Key                        string `keda:"name=key,                        order=authParams, optional"`
	CA                         string `keda:"name=ca,                         order=authParams, optional"`
	ClusterIPAddress           string `keda:"name=clusterIPAddress,           order=triggerMetadata"`
	Port                       int    `keda:"name=port,                       order=triggerMetadata, optional"`
	Consistency                string `keda:"name=consistency,                order=triggerMetadata, default=one"`
	ProtocolVersion            int    `keda:"name=protocolVersion,            order=triggerMetadata, default=4"`
	Keyspace                   string `keda:"name=keyspace,                   order=triggerMetadata"`
	Query                      string `keda:"name=query,                      order=triggerMetadata"`
	TargetQueryValue           int64  `keda:"name=targetQueryValue,           order=triggerMetadata"`
	ActivationTargetQueryValue int64  `keda:"name=activationTargetQueryValue, order=triggerMetadata, default=0"`
	TriggerIndex               int
}

const (
	tlsEnable = "enable"
)

func (m *cassandraMetadata) Validate() error {
	if m.TLS == tlsEnable && (m.Cert == "" || m.Key == "") {
		return errors.New("both cert and key are required when TLS is enabled")
	}

	// Handle port in ClusterIPAddress
	splitVal := strings.Split(m.ClusterIPAddress, ":")
	if len(splitVal) == 2 {
		if port, err := strconv.Atoi(splitVal[1]); err == nil {
			m.Port = port
			return nil
		}
	}

	if m.Port == 0 {
		return fmt.Errorf("no port given")
	}

	m.ClusterIPAddress = net.JoinHostPort(m.ClusterIPAddress, fmt.Sprintf("%d", m.Port))
	return nil
}

// NewCassandraScaler creates a new Cassandra scaler
func NewCassandraScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseCassandraMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cassandra metadata: %w", err)
	}

	session, err := newCassandraSession(meta, InitializeLogger(config, "cassandra_scaler"))
	if err != nil {
		return nil, fmt.Errorf("error establishing cassandra session: %w", err)
	}

	return &cassandraScaler{
		metricType: metricType,
		metadata:   meta,
		session:    session,
		logger:     InitializeLogger(config, "cassandra_scaler"),
	}, nil
}

func parseCassandraMetadata(config *scalersconfig.ScalerConfig) (cassandraMetadata, error) {
	meta := cassandraMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing cassandra metadata: %w", err)
	}

	if config.AsMetricSource {
		meta.TargetQueryValue = 0
	}

	err = parseCassandraTLS(&meta)
	if err != nil {
		return meta, err
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func createTempFile(prefix string, content string) (string, error) {
	tempCassandraDir := fmt.Sprintf("%s%c%s", os.TempDir(), os.PathSeparator, "cassandra")
	err := os.MkdirAll(tempCassandraDir, 0700)
	if err != nil {
		return "", fmt.Errorf(`error creating temporary directory: %s. Error: %w
		Note, when running in a container a writable /tmp/cassandra emptyDir must be mounted. Refer to documentation`, tempCassandraDir, err)
	}

	f, err := os.CreateTemp(tempCassandraDir, prefix+"-*.pem")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func parseCassandraTLS(meta *cassandraMetadata) error {
	if meta.TLS == tlsEnable {
		// Create temp files for certs
		certFilePath, err := createTempFile("cert", meta.Cert)
		if err != nil {
			return fmt.Errorf("error creating cert file: %w", err)
		}
		meta.Cert = certFilePath

		keyFilePath, err := createTempFile("key", meta.Key)
		if err != nil {
			return fmt.Errorf("error creating key file: %w", err)
		}
		meta.Key = keyFilePath

		// If CA cert is given, make also file
		if meta.CA != "" {
			caCertFilePath, err := createTempFile("caCert", meta.CA)
			if err != nil {
				return fmt.Errorf("error creating ca file: %w", err)
			}
			meta.CA = caCertFilePath
		}
	}
	return nil
}

// newCassandraSession returns a new Cassandra session for the provided CassandraMetadata
func newCassandraSession(meta cassandraMetadata, logger logr.Logger) (*gocql.Session, error) {
	cluster := gocql.NewCluster(meta.ClusterIPAddress)
	cluster.ProtoVersion = meta.ProtocolVersion
	cluster.Consistency = gocql.ParseConsistency(meta.Consistency)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: meta.Username,
		Password: meta.Password,
	}

	if meta.TLS == tlsEnable {
		cluster.SslOpts = &gocql.SslOptions{
			CertPath: meta.Cert,
			KeyPath:  meta.Key,
			CaPath:   meta.CA,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		logger.Error(err, "found error creating session")
		return nil, err
	}

	return session, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *cassandraScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("cassandra-%s", s.metadata.Keyspace))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetQueryValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns a value for a supported metric or an error if there is a problem getting the metric
func (s *cassandraScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.GetQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting cassandra: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(num))
	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetQueryValue, nil
}

// GetQueryResult returns the result of the scaler query
func (s *cassandraScaler) GetQueryResult(ctx context.Context) (int64, error) {
	var value int64
	if err := s.session.Query(s.metadata.Query).WithContext(ctx).Scan(&value); err != nil {
		if err != gocql.ErrNotFound {
			s.logger.Error(err, "query failed")
			return 0, err
		}
	}
	return value, nil
}

// Close closes the Cassandra session connection
func (s *cassandraScaler) Close(_ context.Context) error {
	// clean up any temporary files
	if s.metadata.Cert != "" {
		if err := os.Remove(s.metadata.Cert); err != nil {
			return err
		}
	}
	if s.metadata.Key != "" {
		if err := os.Remove(s.metadata.Key); err != nil {
			return err
		}
	}
	if s.metadata.CA != "" {
		if err := os.Remove(s.metadata.CA); err != nil {
			return err
		}
	}

	if s.session != nil {
		s.session.Close()
	}
	return nil
}

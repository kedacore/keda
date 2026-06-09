package scalers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultClickHouseDB       = "default"
	defaultClickHouseUsername = "default"
)

type clickhouseScaler struct {
	metricType v2.MetricTargetType
	metadata   *clickhouseMetadata
	connection *sql.DB
	logger     logr.Logger
}

type clickhouseMetadata struct {
	ConnectionString           string  `keda:"name=connectionString,          order=authParams;resolvedEnv, optional"`
	Host                       string  `keda:"name=host,                      order=triggerMetadata;authParams, optional"`
	Port                       string  `keda:"name=port,                      order=triggerMetadata;authParams, optional"`
	Username                   string  `keda:"name=username,                  order=triggerMetadata;authParams;resolvedEnv, optional"`
	Password                   string  `keda:"name=password,                  order=authParams;resolvedEnv, optional"`
	Database                   string  `keda:"name=database,                  order=triggerMetadata;authParams, optional"`
	Query                      string  `keda:"name=query,                     order=triggerMetadata"`
	TargetQueryValue           float64 `keda:"name=targetQueryValue,          order=triggerMetadata"`
	ActivationTargetQueryValue float64 `keda:"name=activationTargetQueryValue, order=triggerMetadata, default=0"`

	MaxIdleConns    int           `keda:"name=maxIdleConns,    order=triggerMetadata;authParams, optional"`
	MaxOpenConns    int           `keda:"name=maxOpenConns,    order=triggerMetadata;authParams, optional"`
	ConnMaxLifetime time.Duration `keda:"name=connMaxLifetime, order=triggerMetadata;authParams, optional"`

	TLS         bool   `keda:"name=tls,         order=authParams, default=false"`
	Cert        string `keda:"name=cert,        order=authParams, optional"`
	Key         string `keda:"name=key,         order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
	CA          string `keda:"name=ca,          order=authParams, optional"`
	UnsafeSsl   bool   `keda:"name=unsafeSsl,   order=authParams, default=false"`

	triggerIndex int
}

// NewClickHouseScaler creates a new ClickHouse scaler that scales based on SQL query results
func NewClickHouseScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "clickhouse_scaler")

	meta, err := parseClickHouseMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing clickhouse metadata: %w", err)
	}

	conn, err := newClickHouseConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing clickhouse connection: %w", err)
	}

	return &clickhouseScaler{
		metricType: metricType,
		metadata:   meta,
		connection: conn,
		logger:     logger,
	}, nil
}

func parseClickHouseMetadata(config *scalersconfig.ScalerConfig) (*clickhouseMetadata, error) {
	meta := &clickhouseMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing clickhouse metadata: %w", err)
	}
	meta.triggerIndex = config.TriggerIndex

	if !config.AsMetricSource && meta.TargetQueryValue == 0 {
		return nil, fmt.Errorf("targetQueryValue is required when not using scaler as metric source")
	}

	if meta.ConnectionString == "" {
		if meta.Host == "" {
			return nil, fmt.Errorf("either connectionString or host must be set")
		}
		if meta.Port == "" {
			meta.Port = "9000"
		}
		if meta.Database == "" {
			meta.Database = defaultClickHouseDB
		}
		if meta.Username == "" {
			meta.Username = defaultClickHouseUsername
		}
	}

	if meta.TLS {
		if (meta.Cert != "" && meta.Key == "") || (meta.Cert == "" && meta.Key != "") {
			return nil, fmt.Errorf("both cert and key must be provided when TLS is enabled")
		}
	}

	return meta, nil
}

func buildClickHouseDSN(meta *clickhouseMetadata) (string, error) {
	var u *url.URL
	if meta.ConnectionString != "" {
		var err error
		u, err = url.Parse(meta.ConnectionString)
		if err != nil {
			return "", fmt.Errorf("failed to parse connection string: %w", err)
		}
	} else {
		user := url.UserPassword(meta.Username, meta.Password)
		u = &url.URL{
			Scheme: "clickhouse",
			User:   user,
			Host:   net.JoinHostPort(meta.Host, meta.Port),
			Path:   "/" + meta.Database,
		}
	}
	if meta.TLS {
		q := u.Query()
		q.Set("secure", "true")
		if meta.UnsafeSsl {
			q.Set("skip_verify", "true")
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func newClickHouseConnection(meta *clickhouseMetadata, logger logr.Logger) (*sql.DB, error) {
	dsn, err := buildClickHouseDSN(meta)
	if err != nil {
		return nil, err
	}

	var db *sql.DB

	if meta.TLS && (meta.Cert != "" || meta.Key != "" || meta.CA != "") {
		db, err = connectClickHouseWithTLS(meta, dsn)
	} else {
		db, err = sql.Open("clickhouse", dsn)
	}

	if err != nil {
		logger.Error(err, "Found error when opening ClickHouse connection")
		return nil, err
	}
	if meta.MaxIdleConns > 0 {
		db.SetMaxIdleConns(meta.MaxIdleConns)
	}
	if meta.MaxOpenConns > 0 {
		db.SetMaxOpenConns(meta.MaxOpenConns)
	}
	if meta.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(meta.ConnMaxLifetime)
	}
	if err := db.Ping(); err != nil {
		logger.Error(err, "Found error when pinging ClickHouse database")
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func connectClickHouseWithTLS(meta *clickhouseMetadata, dsn string) (*sql.DB, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing ClickHouse DSN: %w", err)
	}
	tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.CA, meta.UnsafeSsl)
	if err != nil {
		return nil, fmt.Errorf("error creating TLS config: %w", err)
	}
	opts.TLS = tlsConfig
	return clickhouse.OpenDB(opts), nil
}

// Close disposes of ClickHouse connections
func (s *clickhouseScaler) Close(context.Context) error {
	if s.connection != nil {
		if err := s.connection.Close(); err != nil {
			s.logger.Error(err, "Error closing clickhouse connection")
			return err
		}
	}
	return nil
}

func (s *clickhouseScaler) getQueryResult(ctx context.Context) (float64, error) {
	var value float64
	err := s.connection.QueryRowContext(ctx, s.metadata.Query).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		s.logger.Error(err, "Could not query ClickHouse database")
		return 0, fmt.Errorf("could not query ClickHouse database: %w", err)
	}
	return value, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *clickhouseScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("clickhouse"))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetQueryValue),
	}
	return []v2.MetricSpec{{
		External: externalMetric,
		Type:     externalMetricType,
	}}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *clickhouseScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting ClickHouse: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)
	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetQueryValue, nil
}

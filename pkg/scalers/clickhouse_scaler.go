package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"

	_ "github.com/ClickHouse/clickhouse-go/v2" // ClickHouse driver for database/sql
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
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
		return nil, fmt.Errorf("targetQueryValue is required")
	}

	if meta.ConnectionString == "" {
		if meta.Host == "" {
			return nil, fmt.Errorf("either connectionString or host must be set")
		}
		if meta.Port == "" {
			meta.Port = "9000"
		}
		if meta.Database == "" {
			meta.Database = "default"
		}
		if meta.Username == "" {
			meta.Username = "default"
		}
	}

	return meta, nil
}

func buildClickHouseDSN(meta *clickhouseMetadata) string {
	if meta.ConnectionString != "" {
		return meta.ConnectionString
	}
	user := url.UserPassword(meta.Username, meta.Password)
	u := &url.URL{
		Scheme: "clickhouse",
		User:   user,
		Host:   net.JoinHostPort(meta.Host, meta.Port),
		Path:   "/" + meta.Database,
	}
	return u.String()
}

func newClickHouseConnection(meta *clickhouseMetadata, logger logr.Logger) (*sql.DB, error) {
	dsn := buildClickHouseDSN(meta)
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error when opening ClickHouse connection: %s", err))
		return nil, err
	}
	if err := db.Ping(); err != nil {
		logger.Error(err, fmt.Sprintf("Found error when pinging ClickHouse database: %s", err))
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// Close disposes of ClickHouse connections
func (s *clickhouseScaler) Close(context.Context) error {
	if err := s.connection.Close(); err != nil {
		s.logger.Error(err, "Error closing clickhouse connection")
		return err
	}
	return nil
}

func (s *clickhouseScaler) getQueryResult(ctx context.Context) (float64, error) {
	var value float64
	err := s.connection.QueryRowContext(ctx, s.metadata.Query).Scan(&value)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Could not query ClickHouse database: %s", err))
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

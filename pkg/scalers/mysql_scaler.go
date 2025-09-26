package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-sql-driver/mysql"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type mySQLScaler struct {
	metricType v2.MetricTargetType
	metadata   *mySQLMetadata
	connection *sql.DB
	logger     logr.Logger
}

type mySQLMetadata struct {
	ConnectionString     string  `keda:"name=connectionString,           order=authParams;resolvedEnv, optional"` // Database connection string
	Username             string  `keda:"name=username,                   order=triggerMetadata;authParams;resolvedEnv, optional"`
	Password             string  `keda:"name=password,                   order=authParams;resolvedEnv, optional"`
	Host                 string  `keda:"name=host,                       order=triggerMetadata;authParams, optional"`
	Port                 string  `keda:"name=port,                       order=triggerMetadata;authParams, optional"`
	DBName               string  `keda:"name=dbName,                     order=triggerMetadata;authParams, optional"`
	Query                string  `keda:"name=query,                      order=triggerMetadata"`
	QueryValue           float64 `keda:"name=queryValue,                 order=triggerMetadata"`
	ActivationQueryValue float64 `keda:"name=activationQueryValue,       order=triggerMetadata, default=0"`
	MetricName           string  `keda:"name=metricName,                 order=triggerMetadata, optional"`
	TriggerIndex         int
}

// NewMySQLScaler creates a new MySQL scaler
func NewMySQLScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "mysql_scaler")

	meta, err := parseMySQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing MySQL metadata: %w", err)
	}

	conn, err := newMySQLConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing MySQL connection: %w", err)
	}
	return &mySQLScaler{
		metricType: metricType,
		metadata:   meta,
		connection: conn,
		logger:     logger,
	}, nil
}

func parseMySQLMetadata(config *scalersconfig.ScalerConfig) (*mySQLMetadata, error) {
	meta := &mySQLMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing mysql metadata: %w", err)
	}
	meta.TriggerIndex = config.TriggerIndex

	if meta.ConnectionString != "" {
		meta.DBName = parseMySQLDbNameFromConnectionStr(meta.ConnectionString)
	}
	meta.MetricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("mysql-%s", meta.DBName)))

	return meta, nil
}

// metadataToConnectionStr builds new MySQL connection string
func metadataToConnectionStr(meta *mySQLMetadata) string {
	var connStr string

	if meta.ConnectionString != "" {
		connStr = meta.ConnectionString
	} else {
		// Build connection str
		config := mysql.NewConfig()
		config.Addr = net.JoinHostPort(meta.Host, meta.Port)
		config.DBName = meta.DBName
		config.Passwd = meta.Password
		config.User = meta.Username
		config.Net = "tcp"
		connStr = config.FormatDSN()
	}
	return connStr
}

// newMySQLConnection creates MySQL db connection
func newMySQLConnection(meta *mySQLMetadata, logger logr.Logger) (*sql.DB, error) {
	connStr := metadataToConnectionStr(meta)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error when opening connection: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error when pinging database: %s", err))
		return nil, err
	}
	return db, nil
}

// parseMySQLDbNameFromConnectionStr returns dbname from connection string
// in it is not able to parse it, it returns "dbname" string
func parseMySQLDbNameFromConnectionStr(connectionString string) string {
	splitted := strings.Split(connectionString, "/")

	if size := len(splitted); size > 0 {
		return splitted[size-1]
	}
	return "dbname"
}

// Close disposes of MySQL connections
func (s *mySQLScaler) Close(context.Context) error {
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing MySQL connection")
		return err
	}
	return nil
}

// getQueryResult returns result of the scaler query
func (s *mySQLScaler) getQueryResult(ctx context.Context) (float64, error) {
	var value float64
	err := s.connection.QueryRowContext(ctx, s.metadata.Query).Scan(&value)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Could not query MySQL database: %s", err))
		return 0, err
	}
	return value, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *mySQLScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.MetricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.QueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *mySQLScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting MySQL: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationQueryValue, nil
}

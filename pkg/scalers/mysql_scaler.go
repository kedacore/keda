package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-sql-driver/mysql"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var (
	// A map that holds MySQL connection pools, keyed by connection string,
	// max open connections, max idle connections, and max idle time
	connectionPools *kedautil.RefMap[mySQLConnectionPoolKey, *sql.DB]
)

func init() {
	// Initialize the global connectionPools map
	connectionPools = kedautil.NewRefMap[mySQLConnectionPoolKey, *sql.DB]()
}

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

	// Connection pool settings
	UseGlobalConnPools bool `keda:"name=useGlobalConnPools, order=triggerMetadata, optional"`
	MaxOpenConns       int  `keda:"name=maxOpenConns,       order=triggerMetadata, optional"`
	MaxIdleConns       int  `keda:"name=maxIdleConns,       order=triggerMetadata, optional"`
	ConnMaxIdleTime    int  `keda:"name=connMaxIdleTime,    order=triggerMetadata, optional"` // seconds
}

// mySQLConnectionPoolKey is used as a key to store MySQL connection pools in
// the global map
type mySQLConnectionPoolKey struct {
	connectionString string
	maxOpenConns     int
	maxIdleConns     int
	connMaxIdleTime  int
}

// newMySQLConnectionPoolKey creates a new mySQLConnectionPoolKey
func newMySQLConnectionPoolKey(meta *mySQLMetadata) mySQLConnectionPoolKey {
	return mySQLConnectionPoolKey{
		connectionString: metadataToConnectionStr(meta),
		maxOpenConns:     meta.MaxOpenConns,
		maxIdleConns:     meta.MaxIdleConns,
		connMaxIdleTime:  meta.ConnMaxIdleTime,
	}
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

	// Create MySQL connection, if useGlobalConnPools is set to true, it will use
	// the global connection pool for the given connection string, otherwise it
	// will create a new local connection pool for the given connection string
	var conn *sql.DB
	if meta.UseGlobalConnPools {
		conn, err = getConnectionPool(meta, logger)
	} else {
		conn, err = newMySQLConnection(meta, logger)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating MySQL connection: %w", err)
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

// getConnectionPool will check if the connection pool has already been
// created for the given connection string and return it. If it has not
// been created, it will create a new connection pool and store it in the
// connectionPools map.
func getConnectionPool(meta *mySQLMetadata, logger logr.Logger) (*sql.DB, error) {
	key := newMySQLConnectionPoolKey(meta)

	// Try to load an existing pool and increment its reference count if found
	if pool, ok := connectionPools.Load(key); ok {
		err := connectionPools.AddRef(key)
		if err != nil {
			logger.Error(err, "Error increasing connection pool reference count")
			return nil, err
		}

		return pool, nil
	}

	// If pool does not exist, create a new one and store it in RefMap
	newPool, err := newMySQLConnection(meta, logger)
	if err != nil {
		return nil, err
	}
	err = connectionPools.Store(key, newPool, func(db *sql.DB) error {
		logger.Info("Closing MySQL connection pool", "connectionString", key.connectionString)
		return db.Close()
	})
	if err != nil {
		logger.Error(err, "Error storing connection pool in RefMap")
		return nil, err
	}

	return newPool, nil
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

	setConnectionPoolConfiguration(meta, db)

	return db, nil
}

// setConnectionPoolConfiguration configures the MySQL connection pool settings
// based on the parameters provided in mySQLMetadata. If a setting is zero, it
// is left at its default value.
func setConnectionPoolConfiguration(meta *mySQLMetadata, db *sql.DB) {
	if meta.MaxOpenConns > 0 {
		db.SetMaxOpenConns(meta.MaxOpenConns)
	}

	if meta.MaxIdleConns > 0 {
		db.SetMaxIdleConns(meta.MaxIdleConns)
	}

	if meta.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(time.Duration(meta.ConnMaxIdleTime) * time.Second)
	}
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

// Close disposes of MySQL connections, closing either the global pool if used
// or the local connection pool
func (s *mySQLScaler) Close(ctx context.Context) error {
	if s.metadata.UseGlobalConnPools {
		if err := s.closeGlobalPool(ctx); err != nil {
			return fmt.Errorf("error closing MySQL connection: %w", err)
		}
	} else {
		if err := s.connection.Close(); err != nil {
			return fmt.Errorf("error closing MySQL connection: %w", err)
		}
	}

	return nil
}

// closeGlobalPool closes all MySQL connections in the global pool
func (s *mySQLScaler) closeGlobalPool(_ context.Context) error {
	key := newMySQLConnectionPoolKey(s.metadata)

	if err := connectionPools.RemoveRef(key); err != nil {
		s.logger.Error(err, "Error decreasing connection pool reference count")
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

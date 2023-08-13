package scalers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	// mssql driver required for this scaler
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var (
	// ErrMsSQLNoQuery is returned when "query" is missing from the config.
	ErrMsSQLNoQuery = errors.New("no query given")

	// ErrMsSQLNoTargetValue is returned when "targetValue" is missing from the config.
	ErrMsSQLNoTargetValue = errors.New("no targetValue given")
)

// mssqlScaler exposes a data pointer to mssqlMetadata and sql.DB connection
type mssqlScaler struct {
	metricType v2.MetricTargetType
	metadata   *mssqlMetadata
	connection *sql.DB
	logger     logr.Logger
}

// mssqlMetadata defines metadata used by KEDA to query a Microsoft SQL database
type mssqlMetadata struct {
	// The connection string used to connect to the MSSQL database.
	// Both URL syntax (sqlserver://host?database=dbName) and OLEDB syntax is supported.
	// +optional
	connectionString string
	// The username credential for connecting to the MSSQL instance, if not specified in the connection string.
	// +optional
	username string
	// The password credential for connecting to the MSSQL instance, if not specified in the connection string.
	// +optional
	password string
	// The hostname of the MSSQL instance endpoint, if not specified in the connection string.
	// +optional
	host string
	// The port number of the MSSQL instance endpoint, if not specified in the connection string.
	// +optional
	port int
	// The name of the database to query, if not specified in the connection string.
	// +optional
	database string
	// The T-SQL query to run against the target database - e.g. SELECT COUNT(*) FROM table.
	// +required
	query string
	// The threshold that is used as targetAverageValue in the Horizontal Pod Autoscaler.
	// +required
	targetValue float64
	// The threshold that is used in activation phase
	// +optional
	activationTargetValue float64
	// The name of the metric to use in the Horizontal Pod Autoscaler. This value will be prefixed with "mssql-".
	// +optional
	metricName string
	// The index of the scaler inside the ScaledObject
	// +internal
	scalerIndex int
}

// NewMSSQLScaler creates a new mssql scaler
func NewMSSQLScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "mssql_scaler")

	meta, err := parseMSSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing mssql metadata: %w", err)
	}

	conn, err := newMSSQLConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing mssql connection: %w", err)
	}

	return &mssqlScaler{
		metricType: metricType,
		metadata:   meta,
		connection: conn,
		logger:     logger,
	}, nil
}

// parseMSSQLMetadata takes a ScalerConfig and returns a mssqlMetadata or an error if the config is invalid
func parseMSSQLMetadata(config *ScalerConfig) (*mssqlMetadata, error) {
	meta := mssqlMetadata{}

	// Query
	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, ErrMsSQLNoQuery
	}

	// Target query value
	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %w", err)
		}
		meta.targetValue = targetValue
	} else {
		return nil, ErrMsSQLNoTargetValue
	}

	// Activation target value
	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %w", err)
		}
		meta.activationTargetValue = activationTargetValue
	}

	// Connection string, which can either be provided explicitly or via the helper fields
	switch {
	case config.AuthParams["connectionString"] != "":
		meta.connectionString = config.AuthParams["connectionString"]
	case config.TriggerMetadata["connectionStringFromEnv"] != "":
		meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	default:
		meta.connectionString = ""
		var err error

		host, err := GetFromAuthOrMeta(config, "host")
		if err != nil {
			return nil, err
		}
		meta.host = host

		var paramPort string
		paramPort, _ = GetFromAuthOrMeta(config, "port")
		if paramPort != "" {
			port, err := strconv.Atoi(paramPort)
			if err != nil {
				return nil, fmt.Errorf("port parsing error %w", err)
			}
			meta.port = port
		}

		meta.username, _ = GetFromAuthOrMeta(config, "username")

		// database is optional in SQL s
		meta.database, _ = GetFromAuthOrMeta(config, "database")

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}
	}
	switch {
	case meta.database != "":
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%s", meta.database))
	case meta.host != "":
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%s", meta.host))
	default:
		meta.metricName = "mssql"
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// newMSSQLConnection returns a new, opened SQL connection for the provided mssqlMetadata
func newMSSQLConnection(meta *mssqlMetadata, logger logr.Logger) (*sql.DB, error) {
	connStr := getMSSQLConnectionString(meta)

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error opening mssql: %s", err))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error pinging mssql: %s", err))
		return nil, err
	}

	return db, nil
}

// getMSSQLConnectionString returns a connection string from a mssqlMetadata
func getMSSQLConnectionString(meta *mssqlMetadata) string {
	var connStr string

	if meta.connectionString != "" {
		connStr = meta.connectionString
	} else {
		query := url.Values{}
		if meta.database != "" {
			query.Add("database", meta.database)
		}

		connectionURL := &url.URL{Scheme: "sqlserver", RawQuery: query.Encode()}
		if meta.username != "" {
			if meta.password != "" {
				connectionURL.User = url.UserPassword(meta.username, meta.password)
			} else {
				connectionURL.User = url.User(meta.username)
			}
		}

		if meta.port > 0 {
			connectionURL.Host = net.JoinHostPort(meta.host, fmt.Sprintf("%d", meta.port))
		} else {
			connectionURL.Host = meta.host
		}

		connStr = connectionURL.String()
	}

	return connStr
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *mssqlScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns a value for a supported metric or an error if there is a problem getting the metric
func (s *mssqlScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting mssql: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetValue, nil
}

// getQueryResult returns the result of the scaler query
func (s *mssqlScaler) getQueryResult(ctx context.Context) (float64, error) {
	var value float64
	err := s.connection.QueryRowContext(ctx, s.metadata.query).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		value = 0
	case err != nil:
		s.logger.Error(err, fmt.Sprintf("Could not query mssql database: %s", err))
		return 0, err
	}

	return value, nil
}

// Close closes the mssql database connections
func (s *mssqlScaler) Close(context.Context) error {
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing mssql connection")
		return err
	}

	return nil
}

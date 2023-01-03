package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-sql-driver/mysql"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type mySQLScaler struct {
	metricType v2.MetricTargetType
	metadata   *mySQLMetadata
	connection *sql.DB
	logger     logr.Logger
}

type mySQLMetadata struct {
	connectionString     string // Database connection string
	username             string
	password             string
	host                 string
	port                 string
	dbName               string
	query                string
	queryValue           float64
	activationQueryValue float64
	metricName           string
}

// NewMySQLScaler creates a new MySQL scaler
func NewMySQLScaler(config *ScalerConfig) (Scaler, error) {
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

func parseMySQLMetadata(config *ScalerConfig) (*mySQLMetadata, error) {
	meta := mySQLMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.queryValue = queryValue
	} else {
		return nil, fmt.Errorf("no queryValue given")
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationQueryValue parsing error %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

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

		port, err := GetFromAuthOrMeta(config, "port")
		if err != nil {
			return nil, err
		}
		meta.port = port

		username, err := GetFromAuthOrMeta(config, "username")
		if err != nil {
			return nil, err
		}
		meta.username = username

		dbName, err := GetFromAuthOrMeta(config, "dbName")
		if err != nil {
			return nil, err
		}
		meta.dbName = dbName

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}

		if len(meta.password) == 0 {
			return nil, fmt.Errorf("no password given")
		}
	}

	if meta.connectionString != "" {
		meta.dbName = parseMySQLDbNameFromConnectionStr(meta.connectionString)
	}
	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("mysql-%s", meta.dbName)))

	return &meta, nil
}

// metadataToConnectionStr builds new MySQL connection string
func metadataToConnectionStr(meta *mySQLMetadata) string {
	var connStr string

	if meta.connectionString != "" {
		connStr = meta.connectionString
	} else {
		// Build connection str
		config := mysql.NewConfig()
		config.Addr = net.JoinHostPort(meta.host, meta.port)
		config.DBName = meta.dbName
		config.Passwd = meta.password
		config.User = meta.username
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
	err := s.connection.QueryRowContext(ctx, s.metadata.query).Scan(&value)
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
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.queryValue),
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

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationQueryValue, nil
}

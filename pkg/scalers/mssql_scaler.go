package scalers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	// mssql driver required for this scaler
	_ "github.com/denisenkom/go-mssqldb"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// mssqlScaler exposes a data pointer to mssqlMetadata and sql.DB connection
type mssqlScaler struct {
	metadata   *mssqlMetadata
	connection *sql.DB
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
	targetValue int
	// The name of the metric to use in the Horizontal Pod Autoscaler. This value will be prefixed with "mssql-".
	// +optional
	metricName string
}

var mssqlLog = logf.Log.WithName("mssql_scaler")

// NewMSSQLScaler creates a new mssql scaler
func NewMSSQLScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseMSSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing mssql metadata: %s", err)
	}

	conn, err := newMSSQLConnection(meta)
	if err != nil {
		return nil, fmt.Errorf("error establishing mssql connection: %s", err)
	}

	return &mssqlScaler{
		metadata:   meta,
		connection: conn,
	}, nil
}

// parseMSSQLMetadata takes a ScalerConfig and returns a mssqlMetadata or an error if the config is invalid
func parseMSSQLMetadata(config *ScalerConfig) (*mssqlMetadata, error) {
	meta := mssqlMetadata{}

	// Query
	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	// Target query value
	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given")
	}

	// Connection string, which can either be provided explicitly or via the helper fields
	switch {
	case config.AuthParams["connectionString"] != "":
		meta.connectionString = config.AuthParams["connectionString"]
	case config.TriggerMetadata["connectionStringFromEnv"] != "":
		meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	default:
		meta.connectionString = ""
		if val, ok := config.TriggerMetadata["host"]; ok {
			meta.host = val
		} else {
			return nil, fmt.Errorf("no host given")
		}

		if val, ok := config.TriggerMetadata["port"]; ok {
			port, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("port parsing error %s", err.Error())
			}

			meta.port = port
		}

		if val, ok := config.TriggerMetadata["username"]; ok {
			meta.username = val
		}

		// database is optional in SQL s
		if val, ok := config.TriggerMetadata["database"]; ok {
			meta.database = val
		}

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}
	}

	// get the metricName, which can be explicit or from the (masked) connection string
	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%s", val))
	} else {
		switch {
		case meta.database != "":
			meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%s", meta.database))
		case meta.host != "":
			meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%s", meta.host))
		case meta.connectionString != "":
			// The mssql provider supports of a variety of connection string formats. Instead of trying to parse
			// the connection string and mask out sensitive data, play it safe and just hash the whole thing.
			connectionStringHash := sha256.Sum256([]byte(meta.connectionString))
			meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mssql-%x", connectionStringHash))
		default:
			meta.metricName = "mssql"
		}
	}

	return &meta, nil
}

// newMSSQLConnection returns a new, opened SQL connection for the provided mssqlMetadata
func newMSSQLConnection(meta *mssqlMetadata) (*sql.DB, error) {
	connStr := getMSSQLConnectionString(meta)

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		mssqlLog.Error(err, fmt.Sprintf("Found error opening mssql: %s", err))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		mssqlLog.Error(err, fmt.Sprintf("Found error pinging mssql: %s", err))
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
			connectionURL.Host = fmt.Sprintf("%s:%d", meta.host, meta.port)
		} else {
			connectionURL.Host = meta.host
		}

		connStr = connectionURL.String()
	}

	return connStr
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *mssqlScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueryValue := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetQueryValue,
		},
	}

	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns a value for a supported metric or an error if there is a problem getting the metric
func (s *mssqlScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getQueryResult()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting mssql: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(num), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// getQueryResult returns the result of the scaler query
func (s *mssqlScaler) getQueryResult() (int, error) {
	var value int
	err := s.connection.QueryRow(s.metadata.query).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		value = 0
	case err != nil:
		mssqlLog.Error(err, fmt.Sprintf("Could not query mssql database: %s", err))
		return 0, err
	}

	return value, nil
}

// IsActive returns true if there are pending events to be processed
func (s *mssqlScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getQueryResult()
	if err != nil {
		return false, fmt.Errorf("error inspecting mssql: %s", err)
	}

	return messages > 0, nil
}

// Close closes the mssql database connections
func (s *mssqlScaler) Close() error {
	err := s.connection.Close()
	if err != nil {
		mssqlLog.Error(err, "Error closing mssql connection")
		return err
	}

	return nil
}

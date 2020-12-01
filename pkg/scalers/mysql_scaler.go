package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type mySQLScaler struct {
	metadata   *mySQLMetadata
	connection *sql.DB
}

type mySQLMetadata struct {
	connectionString string // Database connection string
	username         string
	password         string
	host             string
	port             string
	dbName           string
	query            string
	queryValue       int
}

var mySQLLog = logf.Log.WithName("mysql_scaler")

// NewMySQLScaler creates a new MySQL scaler
func NewMySQLScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseMySQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing MySQL metadata: %s", err)
	}

	conn, err := newMySQLConnection(meta)
	if err != nil {
		return nil, fmt.Errorf("error establishing MySQL connection: %s", err)
	}
	return &mySQLScaler{
		metadata:   meta,
		connection: conn,
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
		queryValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %s", err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, fmt.Errorf("no queryValue given")
	}

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
			meta.port = val
		} else {
			return nil, fmt.Errorf("no port given")
		}

		if val, ok := config.TriggerMetadata["username"]; ok {
			meta.username = val
		} else {
			return nil, fmt.Errorf("no username given")
		}
		if val, ok := config.TriggerMetadata["dbName"]; ok {
			meta.dbName = val
		} else {
			return nil, fmt.Errorf("no dbName given")
		}

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}

		if len(meta.password) == 0 {
			return nil, fmt.Errorf("no password given")
		}
	}

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
		config.Addr = fmt.Sprintf("%s:%s", meta.host, meta.port)
		config.DBName = meta.dbName
		config.Passwd = meta.password
		config.User = meta.username
		config.Net = "tcp"
		connStr = config.FormatDSN()
	}
	return connStr
}

// newMySQLConnection creates MySQL db connection
func newMySQLConnection(meta *mySQLMetadata) (*sql.DB, error) {
	connStr := metadataToConnectionStr(meta)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		mySQLLog.Error(err, fmt.Sprintf("Found error when opening connection: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		mySQLLog.Error(err, fmt.Sprintf("Found error when pinging database: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of MySQL connections
func (s *mySQLScaler) Close() error {
	err := s.connection.Close()
	if err != nil {
		mySQLLog.Error(err, "Error closing MySQL connection")
		return err
	}
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *mySQLScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getQueryResult()
	if err != nil {
		mySQLLog.Error(err, fmt.Sprintf("Error inspecting MySQL: %s", err))
		return false, err
	}
	return messages > 0, nil
}

// getQueryResult returns result of the scaler query
func (s *mySQLScaler) getQueryResult() (int, error) {
	var value int
	err := s.connection.QueryRow(s.metadata.query).Scan(&value)
	if err != nil {
		mySQLLog.Error(err, fmt.Sprintf("Could not query MySQL database: %s", err))
		return 0, err
	}
	return value, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *mySQLScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueryValue := resource.NewQuantity(int64(s.metadata.queryValue), resource.DecimalSI)
	metricName := "mysql"
	if s.metadata.connectionString != "" {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", metricName, s.metadata.connectionString))
	} else {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", metricName, s.metadata.dbName))
	}
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: metricName,
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

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *mySQLScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getQueryResult()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting MySQL: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(num), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

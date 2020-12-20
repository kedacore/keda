package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	// PostreSQL drive required for this scaler
	_ "github.com/lib/pq"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type postgreSQLScaler struct {
	metadata   *postgreSQLMetadata
	connection *sql.DB
}

type postgreSQLMetadata struct {
	targetQueryValue int
	connection       string
	userName         string
	password         string
	host             string
	port             string
	query            string
	dbName           string
	sslmode          string
	metricName       string
}

var postgreSQLLog = logf.Log.WithName("postgreSQL_scaler")

// NewPostgreSQLScaler creates a new postgreSQL scaler
func NewPostgreSQLScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parsePostgreSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgreSQL metadata: %s", err)
	}

	conn, err := getConnection(meta)
	if err != nil {
		return nil, fmt.Errorf("error establishing postgreSQL connection: %s", err)
	}
	return &postgreSQLScaler{
		metadata:   meta,
		connection: conn,
	}, nil
}

func parsePostgreSQLMetadata(config *ScalerConfig) (*postgreSQLMetadata, error) {
	meta := postgreSQLMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["targetQueryValue"]; ok {
		targetQueryValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %s", err.Error())
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		return nil, fmt.Errorf("no targetQueryValue given")
	}

	switch {
	case config.AuthParams["connection"] != "":
		meta.connection = config.AuthParams["connection"]
	case config.TriggerMetadata["connectionFromEnv"] != "":
		meta.connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
	default:
		meta.connection = ""
		if val, ok := config.TriggerMetadata["host"]; ok {
			meta.host = val
		} else {
			return nil, fmt.Errorf("no  host given")
		}
		if val, ok := config.TriggerMetadata["port"]; ok {
			meta.port = val
		} else {
			return nil, fmt.Errorf("no  port given")
		}

		if val, ok := config.TriggerMetadata["userName"]; ok {
			meta.userName = val
		} else {
			return nil, fmt.Errorf("no  username given")
		}
		if val, ok := config.TriggerMetadata["dbName"]; ok {
			meta.dbName = val
		} else {
			return nil, fmt.Errorf("no dbname given")
		}
		if val, ok := config.TriggerMetadata["sslmode"]; ok {
			meta.sslmode = val
		} else {
			return nil, fmt.Errorf("no sslmode name given")
		}

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("postgresql-%s", val))
	} else {
		if meta.connection != "" {
			maskedConnectionString, err := kedautil.MaskPartOfURL(meta.connection, kedautil.Password)
			if err != nil {
				return nil, fmt.Errorf("url parsing error %s", err.Error())
			}

			meta.metricName = kedautil.NormalizeString(fmt.Sprintf("postgresql-%s", maskedConnectionString))
		} else {
			meta.metricName = kedautil.NormalizeString(fmt.Sprintf("postgresql-%s", meta.dbName))
		}
	}

	return &meta, nil
}

func getConnection(meta *postgreSQLMetadata) (*sql.DB, error) {
	var connStr string
	if meta.connection != "" {
		connStr = meta.connection
	} else {
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s password=%s",
			meta.host,
			meta.port,
			meta.userName,
			meta.dbName,
			meta.sslmode,
			meta.password,
		)
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		postgreSQLLog.Error(err, fmt.Sprintf("Found error opening postgreSQL: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		postgreSQLLog.Error(err, fmt.Sprintf("Found error pinging postgreSQL: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of postgres connections
func (s *postgreSQLScaler) Close() error {
	err := s.connection.Close()
	if err != nil {
		postgreSQLLog.Error(err, "Error closing postgreSQL connection")
		return err
	}
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *postgreSQLScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getActiveNumber()
	if err != nil {
		return false, fmt.Errorf("error inspecting postgreSQL: %s", err)
	}

	return messages > 0, nil
}

func (s *postgreSQLScaler) getActiveNumber() (int, error) {
	var id int
	err := s.connection.QueryRow(s.metadata.query).Scan(&id)
	if err != nil {
		postgreSQLLog.Error(err, fmt.Sprintf("could not query postgreSQL: %s", err))
		return 0, fmt.Errorf("could not query postgreSQL: %s", err)
	}
	return id, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *postgreSQLScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueryValue := resource.NewQuantity(int64(s.metadata.targetQueryValue), resource.DecimalSI)

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

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *postgreSQLScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getActiveNumber()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting postgreSQL: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(num), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

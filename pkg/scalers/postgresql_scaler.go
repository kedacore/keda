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

	kedautil "github.com/kedacore/keda/pkg/util"
)

const (
	pgMetricName = "num"
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
}

var postgreSQLLog = logf.Log.WithName("postgreSQL_scaler")

// NewPostgreSQLScaler creates a new postgreSQL scaler
func NewPostgreSQLScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parsePostgreSQLMetadata(resolvedEnv, metadata, authParams)
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

func parsePostgreSQLMetadata(resolvedEnv, metadata, authParams map[string]string) (*postgreSQLMetadata, error) {
	meta := postgreSQLMetadata{}

	if val, ok := metadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := metadata["targetQueryValue"]; ok {
		targetQueryValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %s", err.Error())
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		return nil, fmt.Errorf("no targetQueryValue given")
	}

	if authParams["connection"] != "" {
		meta.connection = authParams["connection"]
	} else if metadata["connectionFromEnv"] != "" {
		meta.connection = resolvedEnv[metadata["connectionFromEnv"]]
	} else {
		meta.connection = ""
		if val, ok := metadata["host"]; ok {
			meta.host = val
		} else {
			return nil, fmt.Errorf("no  host given")
		}
		if val, ok := metadata["port"]; ok {
			meta.port = val
		} else {
			return nil, fmt.Errorf("no  port given")
		}

		if val, ok := metadata["userName"]; ok {
			meta.userName = val
		} else {
			return nil, fmt.Errorf("no  username given")
		}
		if val, ok := metadata["dbName"]; ok {
			meta.dbName = val
		} else {
			return nil, fmt.Errorf("no dbname given")
		}
		if val, ok := metadata["sslmode"]; ok {
			meta.sslmode = val
		} else {
			return nil, fmt.Errorf("no sslmode name given")
		}

		if authParams["password"] != "" {
			meta.password = authParams["password"]
		} else if metadata["passwordFromEnv"] != "" {
			meta.password = resolvedEnv[metadata["passwordFromEnv"]]
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
	metricName := "postgresql"
	if s.metadata.connection != "" {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", metricName, s.metadata.connection))
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

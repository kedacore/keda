package scalers

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	pgMetricName            = "num"
	defaultPostgresPassword = ""
)

type postGRESScaler struct {
	metadata   *postGRESMetadata
	connection *sql.DB
}

type postGRESMetadata struct {
	connStr  string
	userName string
	password string
	host     string
	port     string
	query    string
	dbName   string
	sslmode  string
}

var postgresLog = logf.Log.WithName("postgres_scaler")

// NewPostgresScaler creates a new postgres scaler
func NewPostgresScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parsePostgresMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgres metadata: %s", err)
	}

	conn, err := getConnection(meta)
	if err != nil {
		return nil, fmt.Errorf("error establishing postgres connection: %s", err)
	}
	return &postGRESScaler{
		metadata:   meta,
		connection: conn,
	}, nil
}

func parsePostgresMetadata(resolvedEnv, metadata, authParams map[string]string) (*postGRESMetadata, error) {
	meta := postGRESMetadata{}

	if val, ok := metadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := authParams["connStr"]; ok {
		meta.connStr = val
	} else if val, ok := metadata["connStr"]; ok {
		hostSetting := val

		if val, ok := resolvedEnv[hostSetting]; ok {
			meta.connStr = val
		}
	} else {
		meta.connStr = ""
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
		meta.password = defaultPostgresPassword
		if val, ok := authParams["password"]; ok {
			meta.password = val
		} else if val, ok := metadata["password"]; ok && val != "" {
			if passd, ok := resolvedEnv[val]; ok {
				meta.password = passd
			}
		}
	}

	return &meta, nil
}

func getConnection(meta *postGRESMetadata) (*sql.DB, error) {
	var connStr string
	if meta.connStr != "" {
		connStr = meta.connStr
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
		postgresLog.Error(err, fmt.Sprintf("Found error opening: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		postgresLog.Error(err, fmt.Sprintf("Found error pinging: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of postgres connections
func (s *postGRESScaler) Close() error {
	err := s.connection.Close()
	if err != nil {
		postgresLog.Error(err, "Error closing postgres connection")
		return err
	}
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *postGRESScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getActiveNumber()
	if err != nil {
		return false, fmt.Errorf("error inspecting postgres: %s", err)
	}

	return messages > 0, nil
}

func (s *postGRESScaler) getActiveNumber() (int, error) {
	var id int
	err := s.connection.QueryRow(s.metadata.query).Scan(&id)
	if err != nil {
		postgresLog.Error(err, fmt.Sprintf("could not query PG: %s", err))
		return 0, fmt.Errorf("could not query PG: %s", err)
	}
	return id, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *postGRESScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetListLengthQty := resource.NewQuantity(1, resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{
		MetricName:         pgMetricName,
		TargetAverageValue: targetListLengthQty,
	}
	metricSpec := v2beta1.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta1.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *postGRESScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getActiveNumber()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting postgres: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: pgMetricName,
		Value:      *resource.NewQuantity(int64(num), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

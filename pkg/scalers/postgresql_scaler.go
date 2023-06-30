package scalers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	// PostreSQL drive required for this scaler
	_ "github.com/jackc/pgx/v5/stdlib"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type postgreSQLScaler struct {
	metricType v2.MetricTargetType
	metadata   *postgreSQLMetadata
	connection *sql.DB
	logger     logr.Logger
}

type postgreSQLMetadata struct {
	targetQueryValue           float64
	activationTargetQueryValue float64
	connection                 string
	query                      string
	metricName                 string
	scalerIndex                int
}

// NewPostgreSQLScaler creates a new postgreSQL scaler
func NewPostgreSQLScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "postgresql_scaler")

	meta, err := parsePostgreSQLMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing postgreSQL metadata: %w", err)
	}

	conn, err := getConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing postgreSQL connection: %w", err)
	}
	return &postgreSQLScaler{
		metricType: metricType,
		metadata:   meta,
		connection: conn,
		logger:     logger,
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
		targetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		return nil, fmt.Errorf("no targetQueryValue given")
	}

	meta.activationTargetQueryValue = 0
	if val, ok := config.TriggerMetadata["activationTargetQueryValue"]; ok {
		activationTargetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetQueryValue parsing error %w", err)
		}
		meta.activationTargetQueryValue = activationTargetQueryValue
	}

	switch {
	case config.AuthParams["connection"] != "":
		meta.connection = config.AuthParams["connection"]
	case config.TriggerMetadata["connectionFromEnv"] != "":
		meta.connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
	default:
		host, err := GetFromAuthOrMeta(config, "host")
		if err != nil {
			return nil, err
		}

		port, err := GetFromAuthOrMeta(config, "port")
		if err != nil {
			return nil, err
		}

		userName, err := GetFromAuthOrMeta(config, "userName")
		if err != nil {
			return nil, err
		}

		dbName, err := GetFromAuthOrMeta(config, "dbName")
		if err != nil {
			return nil, err
		}

		sslmode, err := GetFromAuthOrMeta(config, "sslmode")
		if err != nil {
			return nil, err
		}

		var password string
		if config.AuthParams["password"] != "" {
			password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}

		// Build connection str
		var params []string
		params = append(params, "host="+escapePostgreConnectionParameter(host))
		params = append(params, "port="+escapePostgreConnectionParameter(port))
		params = append(params, "user="+escapePostgreConnectionParameter(userName))
		params = append(params, "dbname="+escapePostgreConnectionParameter(dbName))
		params = append(params, "sslmode="+escapePostgreConnectionParameter(sslmode))
		params = append(params, "password="+escapePostgreConnectionParameter(password))
		meta.connection = strings.Join(params, " ")
	}

	// FIXME: DEPRECATED to be removed in v2.12
	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("postgresql-%s", val))
	} else {
		meta.metricName = kedautil.NormalizeString("postgresql")
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func getConnection(meta *postgreSQLMetadata, logger logr.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", meta.connection)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error opening postgreSQL: %s", err))
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error pinging postgreSQL: %s", err))
		return nil, err
	}
	return db, nil
}

// Close disposes of postgres connections
func (s *postgreSQLScaler) Close(context.Context) error {
	err := s.connection.Close()
	if err != nil {
		s.logger.Error(err, "Error closing postgreSQL connection")
		return err
	}
	return nil
}

func (s *postgreSQLScaler) getActiveNumber(ctx context.Context) (float64, error) {
	var id float64
	err := s.connection.QueryRowContext(ctx, s.metadata.query).Scan(&id)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("could not query postgreSQL: %s", err))
		return 0, fmt.Errorf("could not query postgreSQL: %w", err)
	}
	return id, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *postgreSQLScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetQueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *postgreSQLScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getActiveNumber(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting postgreSQL: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetQueryValue, nil
}

func escapePostgreConnectionParameter(str string) string {
	if !strings.Contains(str, " ") {
		return str
	}

	str = strings.ReplaceAll(str, "'", "\\'")
	return fmt.Sprintf("'%s'", str)
}

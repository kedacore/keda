package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	couchdb "github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type couchDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *couchDBMetadata
	client     *kivik.Client
	logger     logr.Logger
}

type couchDBQueryRequest struct {
	Selector map[string]interface{} `json:"selector"`
	Fields   []string               `json:"fields"`
}

type couchDBMetadata struct {
	connectionString     string
	host                 string
	port                 string
	username             string
	password             string
	dbName               string
	query                string
	queryValue           int64
	activationQueryValue int64
	metricName           string
	scalerIndex          int
}

type Res struct {
	ID       string `json:"_id"`
	Feet     int    `json:"feet"`
	Greeting string `json:"greeting"`
}

func (s *couchDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s couchDBScaler) Close(ctx context.Context) error {
	if s.client != nil {
		err := s.client.Close(ctx)
		if err != nil {
			s.logger.Error(err, fmt.Sprintf("failed to close couchdb connection, because of %v", err))
			return err
		}
	}
	return nil
}

func (s *couchDBScaler) getQueryResult(ctx context.Context) (int64, error) {
	db := s.client.DB(ctx, s.metadata.dbName)
	var request couchDBQueryRequest
	err := json.Unmarshal([]byte(s.metadata.query), &request)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Couldn't unmarshal query string because of %v", err))
		return 0, err
	}
	rows, err := db.Find(ctx, request, nil)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("failed to fetch rows because of %v", err))
		return 0, err
	}
	var count int64
	for rows.Next() {
		count++
		res := Res{}
		if err := rows.ScanDoc(&res); err != nil {
			s.logger.Error(err, fmt.Sprintf("failed to scan the doc because of %v", err))
			return 0, err
		}
	}
	return count, nil
}

func parseCouchDBMetadata(config *ScalerConfig) (*couchDBMetadata, string, error) {
	var connStr string
	var err error
	meta := couchDBMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, "", fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %w", val, err)
		}
		meta.queryValue = queryValue
	} else {
		return nil, "", fmt.Errorf("no queryValue given")
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %w", val, err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	dbName, err := GetFromAuthOrMeta(config, "dbName")
	if err != nil {
		return nil, "", err
	}
	meta.dbName = dbName

	switch {
	case config.AuthParams["connectionString"] != "":
		meta.connectionString = config.AuthParams["connectionString"]
	case config.TriggerMetadata["connectionStringFromEnv"] != "":
		meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	default:
		meta.connectionString = ""
		host, err := GetFromAuthOrMeta(config, "host")
		if err != nil {
			return nil, "", err
		}
		meta.host = host

		port, err := GetFromAuthOrMeta(config, "port")
		if err != nil {
			return nil, "", err
		}
		meta.port = port

		username, err := GetFromAuthOrMeta(config, "username")
		if err != nil {
			return nil, "", err
		}
		meta.username = username

		if config.AuthParams["password"] != "" {
			meta.password = config.AuthParams["password"]
		} else if config.TriggerMetadata["passwordFromEnv"] != "" {
			meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
		}
		if len(meta.password) == 0 {
			return nil, "", fmt.Errorf("no password given")
		}
	}

	if meta.connectionString != "" {
		connStr = meta.connectionString
	} else {
		// Build connection str
		addr := net.JoinHostPort(meta.host, meta.port)
		// nosemgrep: db-connection-string
		connStr = "http://" + addr
	}
	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("coucdb-%s", meta.dbName)))
	meta.scalerIndex = config.ScalerIndex
	return &meta, connStr, nil
}

func NewCouchDBScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, connStr, err := parseCouchDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parsing couchDB metadata, because of %w", err)
	}

	client, err := kivik.New("couch", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	err = client.Authenticate(ctx, couchdb.BasicAuth("admin", meta.password))
	if err != nil {
		return nil, err
	}

	isconnected, err := client.Ping(ctx)
	if !isconnected {
		return nil, fmt.Errorf("%w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ping couchDB, because of %w", err)
	}

	return &couchDBScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
		logger:     InitializeLogger(config, "couchdb_scaler"),
	}, nil
}

// GetMetricsAndActivity query from couchDB,and return to external metrics and activity
func (s *couchDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	result, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect couchdb, because of %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(result))

	return append([]external_metrics.ExternalMetricValue{}, metric), result > s.metadata.activationQueryValue, nil
}

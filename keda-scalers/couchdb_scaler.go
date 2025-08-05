package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	couchdb "github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type couchDBScaler struct {
	metricType v2.MetricTargetType
	metadata   couchDBMetadata
	client     *kivik.Client
	logger     logr.Logger
}

type couchDBMetadata struct {
	ConnectionString     string `keda:"name=connectionString,     order=authParams;triggerMetadata;resolvedEnv,optional"`
	Host                 string `keda:"name=host,                 order=authParams;triggerMetadata,optional"`
	Port                 string `keda:"name=port,                 order=authParams;triggerMetadata,optional"`
	Username             string `keda:"name=username,             order=authParams;triggerMetadata,optional"`
	Password             string `keda:"name=password,             order=authParams;triggerMetadata;resolvedEnv,optional"`
	DBName               string `keda:"name=dbName,               order=authParams;triggerMetadata,optional"`
	Query                string `keda:"name=query,                order=triggerMetadata,optional"`
	QueryValue           int64  `keda:"name=queryValue,           order=triggerMetadata,optional"`
	ActivationQueryValue int64  `keda:"name=activationQueryValue, order=triggerMetadata,default=0"`
	TriggerIndex         int
}

func (m *couchDBMetadata) Validate() error {
	if m.ConnectionString == "" {
		if m.Host == "" {
			return fmt.Errorf("no host given")
		}
		if m.Port == "" {
			return fmt.Errorf("no port given")
		}
		if m.Username == "" {
			return fmt.Errorf("no username given")
		}
		if m.Password == "" {
			return fmt.Errorf("no password given")
		}
		if m.DBName == "" {
			return fmt.Errorf("no dbName given")
		}
	}
	return nil
}

type couchDBQueryRequest struct {
	Selector map[string]interface{} `json:"selector"`
	Fields   []string               `json:"fields"`
}

type Res struct {
	ID       string `json:"_id"`
	Feet     int    `json:"feet"`
	Greeting string `json:"greeting"`
}

func NewCouchDBScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseCouchDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing couchdb metadata: %w", err)
	}

	connStr := meta.ConnectionString
	if connStr == "" {
		addr := net.JoinHostPort(meta.Host, meta.Port)
		connStr = "http://" + addr
	}

	client, err := kivik.New("couch", connStr)
	if err != nil {
		return nil, fmt.Errorf("error creating couchdb client: %w", err)
	}

	err = client.Authenticate(ctx, couchdb.BasicAuth("admin", meta.Password))
	if err != nil {
		return nil, fmt.Errorf("error authenticating with couchdb: %w", err)
	}

	isConnected, err := client.Ping(ctx)
	if !isConnected || err != nil {
		return nil, fmt.Errorf("failed to ping couchdb: %w", err)
	}

	return &couchDBScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
		logger:     InitializeLogger(config, "couchdb_scaler"),
	}, nil
}

func parseCouchDBMetadata(config *scalersconfig.ScalerConfig) (couchDBMetadata, error) {
	meta := couchDBMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing couchdb metadata: %w", err)
	}

	if meta.QueryValue == 0 && !config.AsMetricSource {
		return meta, fmt.Errorf("no queryValue given")
	}

	if config.AsMetricSource {
		meta.QueryValue = 0
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func (s *couchDBScaler) Close(ctx context.Context) error {
	if s.client != nil {
		if err := s.client.Close(ctx); err != nil {
			s.logger.Error(err, "failed to close couchdb connection")
			return err
		}
	}
	return nil
}

func (s *couchDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("coucdb-%s", s.metadata.DBName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.QueryValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *couchDBScaler) getQueryResult(ctx context.Context) (int64, error) {
	db := s.client.DB(ctx, s.metadata.DBName)

	var request couchDBQueryRequest
	if err := json.Unmarshal([]byte(s.metadata.Query), &request); err != nil {
		return 0, fmt.Errorf("error unmarshaling query: %w", err)
	}

	rows, err := db.Find(ctx, request, nil)
	if err != nil {
		return 0, fmt.Errorf("error executing query: %w", err)
	}

	var count int64
	for rows.Next() {
		count++
		var res Res
		if err := rows.ScanDoc(&res); err != nil {
			return 0, fmt.Errorf("error scanning document: %w", err)
		}
	}

	return count, nil
}

func (s *couchDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	result, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect couchdb: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(result))
	return []external_metrics.ExternalMetricValue{metric}, result > s.metadata.ActivationQueryValue, nil
}

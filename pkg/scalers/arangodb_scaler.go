package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/arangodb/go-driver/jwt"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

type arangoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *arangoDBMetadata
	client     driver.Client
	logger     logr.Logger
}

type dbResult struct {
	Value float64 `json:"value"`
}

// arangoDBMetadata specify arangoDB scaler params.
type arangoDBMetadata struct {
	// Specify arangoDB server endpoint URL or comma separated URL endpoints of all the coordinators.
	// +required
	endpoints string
	// Authentication parameters for connecting to the database
	// +required
	arangoDBAuth *authentication.AuthMeta
	// Specify the unique arangoDB server ID. Only required if bearer JWT is being used.
	// +optional
	serverID string

	// The name of the database to be queried.
	// +required
	dbName string
	// The name of the collection to be queried.
	// +required
	collection string
	// The arangoDB query to be executed.
	// +required
	query string
	// A threshold that is used as targetAverageValue in HPA.
	// +required
	queryValue float64
	// A threshold that is used to check if scaler is active.
	// +optional
	activationQueryValue float64
	// Specify whether to verify the server's certificate chain and host name.
	// +optional
	unsafeSsl bool
	// Specify the max size of the active connection pool.
	// +optional
	connectionLimit int64

	// The index of the scaler inside the ScaledObject
	// +internal
	triggerIndex int
}

// NewArangoDBScaler creates a new arangodbScaler
func NewArangoDBScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseArangoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing arangoDB metadata: %w", err)
	}

	client, err := getNewArangoDBClient(meta)
	if err != nil {
		return nil, err
	}

	return &arangoDBScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
		logger:     InitializeLogger(config, "arangodb_scaler"),
	}, nil
}

func getNewArangoDBClient(meta *arangoDBMetadata) (driver.Client, error) {
	var auth driver.Authentication

	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: strings.Split(meta.endpoints, ","),
		TLSConfig: util.CreateTLSClientConfig(meta.unsafeSsl),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http connection, %w", err)
	}

	if meta.arangoDBAuth.EnableBasicAuth {
		auth = driver.BasicAuthentication(meta.arangoDBAuth.Username, meta.arangoDBAuth.Password)
	} else if meta.arangoDBAuth.EnableBearerAuth {
		hdr, err := jwt.CreateArangodJwtAuthorizationHeader(meta.arangoDBAuth.BearerToken, meta.serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bearer token authorization header, %w", err)
		}
		auth = driver.RawAuthentication(hdr)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: auth,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new client, %w", err)
	}

	return client, nil
}

func parseArangoDBMetadata(config *scalersconfig.ScalerConfig) (*arangoDBMetadata, error) {
	// setting default metadata
	meta := arangoDBMetadata{}

	// parse metaData from ScaledJob config
	endpoints, err := GetFromAuthOrMeta(config, "endpoints")
	if err != nil {
		return nil, err
	}
	meta.endpoints = endpoints

	if val, ok := config.TriggerMetadata["collection"]; ok {
		meta.collection = val
	} else {
		return nil, fmt.Errorf("no collection given")
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert queryValue to int, %w", err)
		}
		meta.queryValue = queryValue
	} else {
		if config.AsMetricSource {
			meta.queryValue = 0
		} else {
			return nil, fmt.Errorf("no queryValue given")
		}
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert activationQueryValue to int, %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	dbName, err := GetFromAuthOrMeta(config, "dbName")
	if err != nil {
		return nil, err
	}
	meta.dbName = dbName

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok && val != "" {
		unsafeSslValue, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unsafeSsl, %w", err)
		}
		meta.unsafeSsl = unsafeSslValue
	}

	if val, ok := config.TriggerMetadata["connectionLimit"]; ok {
		connectionLimit, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert connectionLimit to int, %w", err)
		}
		meta.connectionLimit = connectionLimit
	}

	// parse auth configs from ScalerConfig
	arangoDBAuth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta.arangoDBAuth = arangoDBAuth

	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

// Close disposes of arangoDB connections
func (s *arangoDBScaler) Close(_ context.Context) error {
	return nil
}

func (s *arangoDBScaler) getQueryResult(ctx context.Context) (float64, error) {
	dbExists, err := s.client.DatabaseExists(ctx, s.metadata.dbName)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s database exists, %w", s.metadata.dbName, err)
	}

	if !dbExists {
		return -1, fmt.Errorf("%s database not found", s.metadata.dbName)
	}

	db, err := s.client.Database(ctx, s.metadata.dbName)
	if err != nil {
		return -1, fmt.Errorf("failed to connect to %s db, %w", s.metadata.dbName, err)
	}

	collectionExists, err := db.CollectionExists(ctx, s.metadata.collection)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s collection exists, %w", s.metadata.collection, err)
	}

	if !collectionExists {
		return -1, fmt.Errorf("%s collection not found in %s database", s.metadata.collection, s.metadata.dbName)
	}

	ctx = driver.WithQueryCount(ctx)

	cursor, err := db.Query(ctx, s.metadata.query, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to execute the query, %w", err)
	}

	defer cursor.Close()

	if cursor.Count() != 1 {
		return -1, fmt.Errorf("more than one values received, please check the query, %w", err)
	}

	var result dbResult
	if _, err = cursor.ReadDocument(ctx, &result); err != nil {
		return -1, fmt.Errorf("query result is not in the specified format, pleast check the query, %w", err)
	}

	return result.Value, nil
}

// GetMetricsAndActivity query from arangoDB, and return to external metrics and activity
func (s *arangoDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect arangoDB, %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return append([]external_metrics.ExternalMetricValue{}, metric), num > s.metadata.activationQueryValue, nil
}

// GetMetricSpecForScaling get the query value for scaling
func (s *arangoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "arangodb"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

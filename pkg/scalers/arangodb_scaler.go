package scalers

import (
	"context"
	"crypto/tls"
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
)

type arangoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *arangoDBMetadata
	client     driver.Client
	logger     logr.Logger
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
	queryValue int64
	// A threshold that is used to check if scaler is active.
	// +optional
	activationQueryValue int64
	// Specify whether to verify the server's certificate chain and host name.
	// +optional
	unsafeSsl bool
	// Specify the max size of the active connection pool.
	// +optional
	connectionLimit int64

	// The index of the scaler inside the ScaledObject
	// +internal
	scalerIndex int
}

func NewArangoDBScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseArangoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing arangoDB metadata: %s", err)
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
		TLSConfig: &tls.Config{
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: meta.unsafeSsl,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http connection, %v", err)
	}

	if meta.arangoDBAuth.EnableBasicAuth {
		auth = driver.BasicAuthentication(meta.arangoDBAuth.Username, meta.arangoDBAuth.Password)
	} else if meta.arangoDBAuth.EnableBearerAuth {
		hdr, err := jwt.CreateArangodJwtAuthorizationHeader(meta.arangoDBAuth.BearerToken, meta.serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bearer token authorization header, %v", err)
		}
		auth = driver.RawAuthentication(hdr)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: auth,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new client, %v", err)
	}

	return client, nil
}

func parseArangoDBMetadata(config *ScalerConfig) (*arangoDBMetadata, error) {
	var err error

	// setting default metadata
	meta := arangoDBMetadata{}

	// parse metaData from ScaledJob config
	meta.endpoints, err = GetFromAuthOrMeta(config, "endpoints")
	if err != nil {
		return nil, err
	}

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
		queryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert queryValue to int, %v", err)
		}
		meta.queryValue = queryValue
	} else {
		return nil, fmt.Errorf("no queryValue given")
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert activationQueryValue to int, %v", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	meta.dbName, err = GetFromAuthOrMeta(config, "dbName")
	if err != nil {
		return nil, err
	}

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok && val != "" {
		unsafeSslValue, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unsafeSsl, %v", err)
		}
		meta.unsafeSsl = unsafeSslValue
	}

	if val, ok := config.TriggerMetadata["connectionLimit"]; ok {
		connectionLimit, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert connectionLimit to int, %v", err)
		}
		meta.connectionLimit = connectionLimit
	}

	// parse auth configs from ScalerConfig
	meta.arangoDBAuth, err = authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}

	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *arangoDBScaler) IsActive(ctx context.Context) (bool, error) {
	result, err := s.getQueryResult(ctx)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("failed to get query result by arangoDB, %v", err))
		return false, err
	}
	return result > s.metadata.activationQueryValue, nil
}

// Close disposes of arangoDB connections
func (s *arangoDBScaler) Close(ctx context.Context) error {
	return nil
}

func (s *arangoDBScaler) getQueryResult(ctx context.Context) (int64, error) {

	dbExists, err := s.client.DatabaseExists(ctx, s.metadata.dbName)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s database exists, %v", s.metadata.dbName, err)
	}

	if !dbExists {
		return -1, fmt.Errorf("%s database not found", s.metadata.dbName)
	}

	db, err := s.client.Database(ctx, s.metadata.dbName)
	if err != nil {
		return -1, fmt.Errorf("failed to connect to %s db, %v", s.metadata.dbName, err)
	}

	collectionExists, err := db.CollectionExists(ctx, s.metadata.collection)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s collection exists, %v", s.metadata.collection, err)
	}

	if !collectionExists {
		return -1, fmt.Errorf("%s collection not found in %s database", s.metadata.collection, s.metadata.dbName)
	}

	ctx = driver.WithQueryCount(ctx)

	cursor, err := db.Query(ctx, s.metadata.query, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to execute the query, %v", err)
	}

	defer cursor.Close()

	return cursor.Count(), nil
}

// GetMetricsAndActivity query from arangoDB,and return to external metrics and activity
func (s *arangoDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect arangoDB, %v", err)
	}

	metric := GenerateMetricInMili(metricName, float64(num))

	return append([]external_metrics.ExternalMetricValue{}, metric), num > s.metadata.activationQueryValue, nil
}

// GetMetricSpecForScaling get the query value for scaling
func (s *arangoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, "arangodb"),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

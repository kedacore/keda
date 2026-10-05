package scalers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	driver "github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
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
	// Specify arangoDB server endpoint URL or comma separated URL Endpoints of all the coordinators.
	// +required
	Endpoints string `keda:"name=endpoints, order=authParams;triggerMetadata"`
	// Authentication parameters for connecting to the database
	// +required
	ArangoDBAuth *authentication.Config `keda:"optional"`
	// Specify the unique arangoDB server ID. Only required if bearer JWT is being used.
	// +optional
	serverID string

	// The name of the database to be queried.
	// +required
	DbName string `keda:"name=dbName, order=triggerMetadata;authParams"`
	// The name of the Collection to be queried.
	// +required
	Collection string `keda:"name=collection, order=triggerMetadata"`
	// The arangoDB Query to be executed.
	// +required
	Query string `keda:"name=query, order=triggerMetadata"`
	// A threshold that is used as targetAverageValue in HPA.
	// +required
	QueryValue float64 `keda:"name=queryValue, order=triggerMetadata, default=0"`
	// A threshold that is used to check if scaler is active.
	// +optional
	ActivationQueryValue float64 `keda:"name=activationQueryValue, order=triggerMetadata, default=0"`
	// Specify whether to verify the server's certificate chain and host name.
	// +optional
	UnsafeSsl bool `keda:"name=unsafeSsl, order=triggerMetadata ,default=false"`
	// Specify the max size of the active connection pool.
	// +optional
	ConnectionLimit int64 `keda:"name=connectionLimit, order=triggerMetadata, optional"`

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

// arangodJWTIssuer is the JWT issuer claim expected by arangod for internal authentication.
const arangodJWTIssuer = "arangodb"

func getNewArangoDBClient(meta *arangoDBMetadata) (driver.Client, error) {
	var auth connection.Authentication

	if meta.ArangoDBAuth.EnabledBasicAuth() {
		auth = connection.NewBasicAuth(meta.ArangoDBAuth.Username, meta.ArangoDBAuth.Password)
	} else if meta.ArangoDBAuth.EnabledBearerAuth() {
		hdr, err := createArangodJWTAuthorizationHeader(meta.ArangoDBAuth.BearerToken, meta.serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bearer token authorization header, %w", err)
		}
		auth = connection.NewHeaderAuth("Authorization", hdr)
	}

	conn := connection.NewHttpConnection(connection.HttpConfiguration{
		Authentication: auth,
		Endpoint:       connection.NewRoundRobinEndpoints(strings.Split(meta.Endpoints, ",")),
		Transport: &http.Transport{
			TLSClientConfig: util.CreateTLSClientConfig(meta.UnsafeSsl),
		},
	})

	return driver.NewClient(conn), nil
}

// createArangodJWTAuthorizationHeader mirrors the removed go-driver/jwt helper for internal arangod authentication.
func createArangodJWTAuthorizationHeader(jwtSecret, serverID string) (string, error) {
	if jwtSecret == "" || serverID == "" {
		return "", nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":       arangodJWTIssuer,
		"server_id": serverID,
	})

	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return "bearer " + signedToken, nil
}

func parseArangoDBMetadata(config *scalersconfig.ScalerConfig) (*arangoDBMetadata, error) {
	meta := &arangoDBMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing arango metadata: %w", err)
	}

	if !config.AsMetricSource && meta.QueryValue == 0 {
		return nil, fmt.Errorf("no QueryValue given")
	}

	return meta, nil
}

// Close disposes of arangoDB connections
func (s *arangoDBScaler) Close(_ context.Context) error {
	return nil
}

func (s *arangoDBScaler) getQueryResult(ctx context.Context) (float64, error) {
	dbExists, err := s.client.DatabaseExists(ctx, s.metadata.DbName)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s database exists, %w", s.metadata.DbName, err)
	}

	if !dbExists {
		return -1, fmt.Errorf("%s database not found", s.metadata.DbName)
	}

	db, err := s.client.GetDatabase(ctx, s.metadata.DbName, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to connect to %s db, %w", s.metadata.DbName, err)
	}

	collectionExists, err := db.CollectionExists(ctx, s.metadata.Collection)
	if err != nil {
		return -1, fmt.Errorf("failed to check if %s collection exists, %w", s.metadata.Collection, err)
	}

	if !collectionExists {
		return -1, fmt.Errorf("%s collection not found in %s database", s.metadata.Collection, s.metadata.DbName)
	}

	cursor, err := db.Query(ctx, s.metadata.Query, &driver.QueryOptions{Count: true})
	if err != nil {
		return -1, fmt.Errorf("failed to execute the query, %w", err)
	}

	defer cursor.Close()

	if cursor.Count() != 1 {
		return -1, fmt.Errorf("more than one values received, please check the query, %w", err)
	}

	var result dbResult
	if _, err = cursor.ReadDocument(ctx, &result); err != nil {
		return -1, fmt.Errorf("query result is not in the specified format, please check the query, %w", err)
	}

	return result.Value, nil
}

// GetMetricsAndActivity Query from arangoDB, and return to external metrics and activity
func (s *arangoDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect arangoDB, %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return append([]external_metrics.ExternalMetricValue{}, metric), num > s.metadata.ActivationQueryValue, nil
}

// GetMetricSpecForScaling get the query value for scaling
func (s *arangoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "arangodb"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.QueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

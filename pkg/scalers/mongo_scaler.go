package scalers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type mongoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   mongoDBMetadata
	client     *mongo.Client
	logger     logr.Logger
}

type mongoDBMetadata struct {
	ConnectionString     string  `keda:"name=connectionString,     order=authParams;triggerMetadata;resolvedEnv,optional"`
	Scheme               string  `keda:"name=scheme,               order=authParams;triggerMetadata,default=mongodb"`
	Host                 string  `keda:"name=host,                 order=authParams;triggerMetadata,optional"`
	Port                 string  `keda:"name=port,                 order=authParams;triggerMetadata,optional"`
	Username             string  `keda:"name=username,             order=authParams;triggerMetadata,optional"`
	Password             string  `keda:"name=password,             order=authParams;triggerMetadata;resolvedEnv,optional"`
	DBName               string  `keda:"name=dbName,               order=authParams;triggerMetadata"`
	Collection           string  `keda:"name=collection,           order=triggerMetadata"`
	Query                string  `keda:"name=query,                order=triggerMetadata"`
	QueryValue           float64 `keda:"name=queryValue,           order=triggerMetadata"`
	ActivationQueryValue float64 `keda:"name=activationQueryValue, order=triggerMetadata,default=0"`
	TriggerIndex         int
}

func (m *mongoDBMetadata) Validate() error {
	if m.ConnectionString == "" {
		if m.Host == "" {
			return fmt.Errorf("no host given")
		}
		if m.Port == "" && m.Scheme != "mongodb+srv" {
			return fmt.Errorf("no port given")
		}
		if m.Username == "" {
			return fmt.Errorf("no username given")
		}
		if m.Password == "" {
			return fmt.Errorf("no password given")
		}
	}
	return nil
}

// NewMongoDBScaler creates a new mongoDB scaler
func NewMongoDBScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseMongoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing mongodb metadata: %w", err)
	}

	client, err := createMongoDBClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error creating mongodb client: %w", err)
	}

	return &mongoDBScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
		logger:     InitializeLogger(config, "mongodb_scaler"),
	}, nil
}

func parseMongoDBMetadata(config *scalersconfig.ScalerConfig) (mongoDBMetadata, error) {
	meta := mongoDBMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing mongodb metadata: %w", err)
	}

	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func createMongoDBClient(ctx context.Context, meta mongoDBMetadata) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var connString string
	if meta.ConnectionString != "" {
		connString = meta.ConnectionString
	} else {
		host := meta.Host
		if meta.Scheme != "mongodb+srv" {
			host = net.JoinHostPort(meta.Host, meta.Port)
		}
		u := &url.URL{
			Scheme: meta.Scheme,
			User:   url.UserPassword(meta.Username, meta.Password),
			Host:   host,
			Path:   meta.DBName,
		}
		connString = u.String()
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connString))
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb client: %w", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

func (s *mongoDBScaler) Close(ctx context.Context) error {
	if s.client != nil {
		err := s.client.Disconnect(ctx)
		if err != nil {
			s.logger.Error(err, "Error closing mongodb connection")
			return err
		}
	}
	return nil
}

func (s *mongoDBScaler) getQueryResult(ctx context.Context) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	collection := s.client.Database(s.metadata.DBName).Collection(s.metadata.Collection)

	filter, err := json2BsonDoc(s.metadata.Query)
	if err != nil {
		return 0, fmt.Errorf("failed to parse query: %w", err)
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	return float64(count), nil
}

func (s *mongoDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect mongodb: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationQueryValue, nil
}

func (s *mongoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("mongodb-%s", s.metadata.Collection))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.QueryValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// json2BsonDoc convert Json to bson.D
func json2BsonDoc(js string) (doc bson.D, err error) {
	doc = bson.D{}
	err = bson.UnmarshalExtJSON([]byte(js), true, &doc)
	if err != nil {
		return nil, err
	}

	if len(doc) == 0 {
		return nil, errors.New("empty bson document")
	}

	return doc, nil
}

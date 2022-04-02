package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// mongoDBScaler is support for mongoDB in keda.
type mongoDBScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *mongoDBMetadata
	client     *mongo.Client
}

// mongoDBMetadata specify mongoDB scaler params.
type mongoDBMetadata struct {
	// The string is used by connected with mongoDB.
	// +optional
	connectionString string
	// Specify the host to connect to the mongoDB server,if the connectionString be provided, don't need specify this param.
	// +optional
	host string
	// Specify the port to connect to the mongoDB server,if the connectionString be provided, don't need specify this param.
	// +optional
	port string
	// Specify the username to connect to the mongoDB server,if the connectionString be provided, don't need specify this param.
	// +optional
	username string
	// Specify the password to connect to the mongoDB server,if the connectionString be provided, don't need specify this param.
	// +optional
	password string

	// The name of the database to be queried.
	// +required
	dbName string
	// The name of the collection to be queried.
	// +required
	collection string
	// A mongoDB filter doc,used by specify DB.
	// +required
	query string
	// A threshold that is used as targetAverageValue in HPA
	// +required
	queryValue int64
	// The name of the metric to use in the Horizontal Pod Autoscaler. This value will be prefixed with "mongodb-".
	// +optional
	metricName string

	// The index of the scaler inside the ScaledObject
	// +internal
	scalerIndex int
}

// Default variables and settings
const (
	mongoDBDefaultTimeOut = 10 * time.Second
)

var mongoDBLog = logf.Log.WithName("mongodb_scaler")

// NewMongoDBScaler creates a new mongoDB scaler
func NewMongoDBScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	ctx, cancel := context.WithTimeout(ctx, mongoDBDefaultTimeOut)
	defer cancel()

	meta, connStr, err := parseMongoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parsing mongoDB metadata, because of %v", err)
	}

	opt := options.Client().ApplyURI(connStr)
	client, err := mongo.Connect(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection with mongoDB, because of %v", err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongoDB, because of %v", err)
	}

	return &mongoDBScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
	}, nil
}

func parseMongoDBMetadata(config *ScalerConfig) (*mongoDBMetadata, string, error) {
	var connStr string
	var err error
	// setting default metadata
	meta := mongoDBMetadata{}

	// parse metaData from ScaledJob config
	if val, ok := config.TriggerMetadata["collection"]; ok {
		meta.collection = val
	} else {
		return nil, "", fmt.Errorf("no collection given")
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, "", fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert %v to int, because of %v", queryValue, err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, "", fmt.Errorf("no queryValue given")
	}

	meta.dbName, err = GetFromAuthOrMeta(config, "dbName")
	if err != nil {
		return nil, "", err
	}

	// Resolve connectionString
	switch {
	case config.AuthParams["connectionString"] != "":
		meta.connectionString = config.AuthParams["connectionString"]
	case config.TriggerMetadata["connectionStringFromEnv"] != "":
		meta.connectionString = config.ResolvedEnv[config.TriggerMetadata["connectionStringFromEnv"]]
	default:
		meta.connectionString = ""
		meta.host, err = GetFromAuthOrMeta(config, "host")
		if err != nil {
			return nil, "", err
		}

		meta.port, err = GetFromAuthOrMeta(config, "port")
		if err != nil {
			return nil, "", err
		}

		meta.username, err = GetFromAuthOrMeta(config, "username")
		if err != nil {
			return nil, "", err
		}

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
		addr := fmt.Sprintf("%s:%s", meta.host, meta.port)
		auth := fmt.Sprintf("%s:%s", meta.username, meta.password)
		connStr = "mongodb://" + auth + "@" + addr + "/" + meta.dbName
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mongodb-%s", val))
	} else {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("mongodb-%s", meta.collection))
	}
	meta.scalerIndex = config.ScalerIndex
	return &meta, connStr, nil
}

func (s *mongoDBScaler) IsActive(ctx context.Context) (bool, error) {
	result, err := s.getQueryResult(ctx)
	if err != nil {
		mongoDBLog.Error(err, fmt.Sprintf("failed to get query result by mongoDB, because of %v", err))
		return false, err
	}
	return result > 0, nil
}

// Close disposes of mongoDB connections
func (s *mongoDBScaler) Close(ctx context.Context) error {
	if s.client != nil {
		err := s.client.Disconnect(ctx)
		if err != nil {
			mongoDBLog.Error(err, fmt.Sprintf("failed to close mongoDB connection, because of %v", err))
			return err
		}
	}

	return nil
}

// getQueryResult query mongoDB by meta.query
func (s *mongoDBScaler) getQueryResult(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, mongoDBDefaultTimeOut)
	defer cancel()

	filter, err := json2BsonDoc(s.metadata.query)
	if err != nil {
		mongoDBLog.Error(err, fmt.Sprintf("failed to convert query param to bson.Doc, because of %v", err))
		return 0, err
	}

	docsNum, err := s.client.Database(s.metadata.dbName).Collection(s.metadata.collection).CountDocuments(ctx, filter)
	if err != nil {
		mongoDBLog.Error(err, fmt.Sprintf("failed to query %v in %v, because of %v", s.metadata.dbName, s.metadata.collection, err))
		return 0, err
	}

	return docsNum, nil
}

// GetMetrics query from mongoDB,and return to external metrics
func (s *mongoDBScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("failed to inspect momgoDB, because of %v", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(num, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// GetMetricSpecForScaling get the query value for scaling
func (s *mongoDBScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.queryValue),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// json2BsonDoc convert Json to Bson.Doc
func json2BsonDoc(js string) (doc bsonx.Doc, err error) {
	doc = bsonx.Doc{}
	err = bson.UnmarshalExtJSON([]byte(js), true, &doc)
	if err != nil {
		return nil, err
	}

	if len(doc) == 0 {
		return nil, errors.New("empty bson document")
	}

	return doc, nil
}

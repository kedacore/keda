package scalers

import (
	"context"
	"fmt"
	"github.com/InfluxCommunity/influxdb3-go/influxdb3"
	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"slices"
	"strconv"
	"strings"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

type influxDBScaler struct {
	client     influxdb2.Client
	metricType v2.MetricTargetType
	metadata   *influxDBMetadata
	logger     logr.Logger
}

type influxDBScalerV3 struct {
	client     influxdb3.Client
	metricType v2.MetricTargetType
	metadata   *influxDBMetadata
	logger     logr.Logger
}

type influxDBMetadata struct {
	authToken                string
	organizationName         string
	query                    string
	influxVersion            string
	queryType                string
	database                 string
	metricKey                string
	serverURL                string
	unsafeSsl                bool
	thresholdValue           float64
	activationThresholdValue float64
	triggerIndex             int
}

// NewInfluxDBScaler creates a new influx db scaler
func NewInfluxDBScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "influxdb_scaler")

	meta, err := parseInfluxDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing influxdb metadata: %w", err)
	}

	if meta.influxVersion == "3" {

		logger.Info("starting up influxdb v3 client")

		clientv3, err := influxdb3.New(influxdb3.ClientConfig{
			Host:     meta.serverURL,
			Token:    meta.authToken,
			Database: meta.database,
		})

		if err != nil {
			panic(err)
		}

		return &influxDBScalerV3{
			client:     *clientv3,
			metricType: metricType,
			metadata:   meta,
			logger:     logger,
		}, nil
	}

	logger.Info("starting up influxdb v2 client")

	client := influxdb2.NewClientWithOptions(
		meta.serverURL,
		meta.authToken,
		influxdb2.DefaultOptions().SetTLSConfig(util.CreateTLSClientConfig(meta.unsafeSsl)))

	return &influxDBScaler{
		client:     client,
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

// parseInfluxDBMetadata parses the metadata passed in from the ScaledObject config
func parseInfluxDBMetadata(config *scalersconfig.ScalerConfig) (*influxDBMetadata, error) {
	var authToken string
	var organizationName string
	var query string
	var influxVersion string
	var queryType string
	var metricKey string
	var database string
	var serverURL string
	var unsafeSsl bool
	var thresholdValue float64
	var activationThresholdValue float64

	val, ok := config.TriggerMetadata["authToken"]
	switch {
	case ok && val != "":
		authToken = val
	case config.TriggerMetadata["authTokenFromEnv"] != "":
		if val, ok := config.ResolvedEnv[config.TriggerMetadata["authTokenFromEnv"]]; ok {
			authToken = val
		} else {
			return nil, fmt.Errorf("no auth token given")
		}
	case config.AuthParams["authToken"] != "":
		authToken = config.AuthParams["authToken"]
	default:
		return nil, fmt.Errorf("no auth token given")
	}

	val, ok = config.TriggerMetadata["organizationName"]
	switch {
	case ok && val != "":
		organizationName = val
	case config.TriggerMetadata["influxVersion"] == "3":
		organizationName = val
	case config.TriggerMetadata["organizationNameFromEnv"] != "":
		if val, ok := config.ResolvedEnv[config.TriggerMetadata["organizationNameFromEnv"]]; ok {
			organizationName = val
		} else {
			return nil, fmt.Errorf("no organization name given")
		}
	case config.AuthParams["organizationName"] != "":
		organizationName = config.AuthParams["organizationName"]
	default:
		return nil, fmt.Errorf("no organization name given")
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		query = val
	} else {
		return nil, fmt.Errorf("no query provided")
	}

	if val, ok := config.TriggerMetadata["serverURL"]; ok {
		serverURL = val
	} else if val, ok := config.AuthParams["serverURL"]; ok {
		serverURL = val
	} else {
		return nil, fmt.Errorf("no server url given")
	}

	if val, ok := config.TriggerMetadata["activationThresholdValue"]; ok {
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThresholdValue: failed to parse activationThresholdValue %w", err)
		}
		activationThresholdValue = value
	}

	if val, ok := config.TriggerMetadata["thresholdValue"]; ok {
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("thresholdValue: failed to parse thresholdValue length %w", err)
		}
		thresholdValue = value
	} else {
		if config.AsMetricSource {
			thresholdValue = 0
		} else {
			return nil, fmt.Errorf("no threshold value given")
		}
	}
	unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		parsedVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		unsafeSsl = parsedVal
	}
	if val, ok := config.TriggerMetadata["influxVersion"]; ok {
		versions := []string{"", "2", "3"}
		if !slices.Contains(versions, val) {
			return nil, fmt.Errorf("unsupported influxVersion")
		}
		influxVersion = val
		if val == "3" {
			if val, ok := config.TriggerMetadata["database"]; ok {
				database = val
			} else {
				return nil, fmt.Errorf("database is required for influxdb v3")
			}
			if val, ok := config.TriggerMetadata["metricKey"]; ok {
				metricKey = val
			} else {
				return nil, fmt.Errorf("metricKey is required for influxdb v3")
			}
			val, ok = config.TriggerMetadata["queryType"]
			queryType = val
		}
	}

	return &influxDBMetadata{
		authToken:                authToken,
		organizationName:         organizationName,
		query:                    query,
		influxVersion:            influxVersion,
		queryType:                queryType,
		database:                 database,
		metricKey:                metricKey,
		serverURL:                serverURL,
		thresholdValue:           thresholdValue,
		activationThresholdValue: activationThresholdValue,
		unsafeSsl:                unsafeSsl,
		triggerIndex:             config.TriggerIndex,
	}, nil
}

// Close closes the connection of the client to the server
func (s *influxDBScaler) Close(context.Context) error {
	s.client.Close()
	return nil
}

// Close closes the connection of the client to the server
func (s *influxDBScalerV3) Close(context.Context) error {
	s.client.Close()
	return nil
}

// queryOptionsInfluxDBV3 returns influxdb QueryOptions based on the database and queryType (InfluxQL or FlightSQL)
func queryOptionsInfluxDBV3(d string, q string) *influxdb3.QueryOptions {
	switch strings.ToLower(q) {
	case "influxql":
		return &influxdb3.QueryOptions{Database: d, QueryType: influxdb3.QueryType(1)}
	}
	return &influxdb3.QueryOptions{Database: d, QueryType: influxdb3.QueryType(0)}
}

// queryInfluxDB runs the query against the associated influxdb v2 database
// there is an implicit assumption here that the first value returned from the iterator
// will be the value of interest
func queryInfluxDB(ctx context.Context, queryAPI api.QueryAPI, query string) (float64, error) {
	result, err := queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	valueExists := result.Next()
	if !valueExists {
		return 0, fmt.Errorf("no results found from query")
	}

	switch valRaw := result.Record().Value().(type) {
	case float64:
		return valRaw, nil
	case int64:
		return float64(valRaw), nil
	default:
		return 0, fmt.Errorf("value of type %T could not be converted into a float", valRaw)
	}
}

// queryInfluxDBV3 runs the query against the associated influxdb v3 database
// there is an implicit assumption here that the first value returned from the iterator
// will be the value of interest
func queryInfluxDBV3(ctx context.Context, client influxdb3.Client, metadata influxDBMetadata) (float64, error) {
	queryOptions := queryOptionsInfluxDBV3(metadata.database, metadata.queryType)

	result, err := client.QueryWithOptions(ctx, queryOptions, metadata.query)

	if err != nil {
		return 0, err
	}

	var parsedVal float64

	for result.Next() {
		value := result.Value()

		switch valRaw := value[metadata.metricKey].(type) {
		case float64:
			parsedVal = valRaw
		case int64:
			parsedVal = float64(valRaw)
		default:
			return 0, fmt.Errorf("value of type %T could not be converted into a float", valRaw)
		}
	}
	return parsedVal, nil
}

// GetMetricsAndActivity connects to influxdb via the client and returns a value based on the query
func (s *influxDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	// Grab QueryAPI to make queries to influxdb instance
	queryAPI := s.client.QueryAPI(s.metadata.organizationName)

	value, err := queryInfluxDB(ctx, queryAPI, s.metadata.query)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.activationThresholdValue, nil
}

// GetMetricsAndActivity connects to influxdb via the client and returns a value based on the query
func (s *influxDBScalerV3) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, err := queryInfluxDBV3(ctx, s.client, *s.metadata)

	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.activationThresholdValue, nil
}

// GetMetricSpecForScaling returns the metric spec for the Horizontal Pod Autoscaler
func (s *influxDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, util.NormalizeString(fmt.Sprintf("influxdb-%s", s.metadata.organizationName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.thresholdValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricSpecForScaling returns the metric spec for the Horizontal Pod Autoscaler
func (s *influxDBScalerV3) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, util.NormalizeString(fmt.Sprintf("influxdb-%s", s.metadata.database))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.thresholdValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

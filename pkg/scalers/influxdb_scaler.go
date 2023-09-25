package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/util"
)

type influxDBScaler struct {
	client     influxdb2.Client
	metricType v2.MetricTargetType
	metadata   *influxDBMetadata
	logger     logr.Logger
}

type influxDBMetadata struct {
	authToken                string
	organizationName         string
	query                    string
	serverURL                string
	unsafeSsl                bool
	thresholdValue           float64
	activationThresholdValue float64
	scalerIndex              int
}

// NewInfluxDBScaler creates a new influx db scaler
func NewInfluxDBScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "influxdb_scaler")

	meta, err := parseInfluxDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing influxdb metadata: %w", err)
	}

	logger.Info("starting up influxdb client")
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
func parseInfluxDBMetadata(config *ScalerConfig) (*influxDBMetadata, error) {
	var authToken string
	var organizationName string
	var query string
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

	return &influxDBMetadata{
		authToken:                authToken,
		organizationName:         organizationName,
		query:                    query,
		serverURL:                serverURL,
		thresholdValue:           thresholdValue,
		activationThresholdValue: activationThresholdValue,
		unsafeSsl:                unsafeSsl,
		scalerIndex:              config.ScalerIndex,
	}, nil
}

// Close closes the connection of the client to the server
func (s *influxDBScaler) Close(context.Context) error {
	s.client.Close()
	return nil
}

// queryInfluxDB runs the query against the associated influxdb database
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

// GetMetricSpecForScaling returns the metric spec for the Horizontal Pod Autoscaler
func (s *influxDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, util.NormalizeString(fmt.Sprintf("influxdb-%s", s.metadata.organizationName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.thresholdValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

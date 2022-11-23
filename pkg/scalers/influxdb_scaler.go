package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type influxDBScaler struct {
	client     influxdb2.Client
	metricType v2.MetricTargetType
	metadata   *influxDBMetadata
	logger     logr.Logger
}

type influxDBMetadata struct {
	authToken                string
	metricName               string
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
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "influxdb_scaler")

	meta, err := parseInfluxDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing influxdb metadata: %s", err)
	}

	logger.Info("starting up influxdb client")
	client := influxdb2.NewClientWithOptions(
		meta.serverURL,
		meta.authToken,
		influxdb2.DefaultOptions().SetTLSConfig(&tls.Config{InsecureSkipVerify: meta.unsafeSsl}))

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
	var metricName string
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

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		metricName = kedautil.NormalizeString(fmt.Sprintf("influxdb-%s", val))
	} else {
		metricName = kedautil.NormalizeString(fmt.Sprintf("influxdb-%s", organizationName))
	}

	if val, ok := config.TriggerMetadata["activationThresholdValue"]; ok {
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThresholdValue: failed to parse activationThresholdValue %s", err.Error())
		}
		activationThresholdValue = value
	}

	if val, ok := config.TriggerMetadata["thresholdValue"]; ok {
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("thresholdValue: failed to parse thresholdValue length %s", err.Error())
		}
		thresholdValue = value
	} else {
		return nil, fmt.Errorf("no threshold value given")
	}
	unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		parsedVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %s", err)
		}
		unsafeSsl = parsedVal
	}

	return &influxDBMetadata{
		authToken:                authToken,
		metricName:               metricName,
		organizationName:         organizationName,
		query:                    query,
		serverURL:                serverURL,
		thresholdValue:           thresholdValue,
		activationThresholdValue: activationThresholdValue,
		unsafeSsl:                unsafeSsl,
		scalerIndex:              config.ScalerIndex,
	}, nil
}

// IsActive returns true if queried value is above the minimum value
func (s *influxDBScaler) IsActive(ctx context.Context) (bool, error) {
	queryAPI := s.client.QueryAPI(s.metadata.organizationName)

	value, err := queryInfluxDB(ctx, queryAPI, s.metadata.query)
	if err != nil {
		return false, err
	}

	return value > s.metadata.activationThresholdValue, nil
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

// GetMetrics connects to influxdb via the client and returns a value based on the query
func (s *influxDBScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	// Grab QueryAPI to make queries to influxdb instance
	queryAPI := s.client.QueryAPI(s.metadata.organizationName)

	value, err := queryInfluxDB(ctx, queryAPI, s.metadata.query)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// GetMetricSpecForScaling returns the metric spec for the Horizontal Pod Autoscaler
func (s *influxDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.thresholdValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

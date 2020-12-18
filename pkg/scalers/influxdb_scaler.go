package scalers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type influxDBScaler struct {
	client   influxdb2.Client
	metadata *influxDBMetadata
}

type influxDBMetadata struct {
	authToken        string
	metricName       string
	organizationName string
	query            string
	serverURL        string
	thresholdValue   float64
}

var influxDBLog = logf.Log.WithName("influxdb_scaler")

// NewInfluxDBScaler creates a new influx db scaler
func NewInfluxDBScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseInfluxDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing influxdb metadata: %s", err)
	}

	influxDBLog.Info("starting up influxdb client")

	return &influxDBScaler{
		client:   influxdb2.NewClient(meta.serverURL, meta.authToken),
		metadata: meta,
	}, nil
}

// parseInfluxDBMetadata parses the metadata passed in from the ScaledObject config
func parseInfluxDBMetadata(config *ScalerConfig) (*influxDBMetadata, error) {
	var authToken string
	var metricName string
	var organizationName string
	var query string
	var serverURL string
	var thresholdValue float64

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
	} else {
		return nil, fmt.Errorf("no server url given")
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		metricName = val
	} else {
		parsedURL, err := url.Parse(serverURL)
		if err != nil {
			return nil, fmt.Errorf("unable to parse server url")
		}

		metricName, err = kedautil.MaskPartOfURL(parsedURL.String(), kedautil.Hostname)
		if err != nil {
			return nil, fmt.Errorf("failure masking part of url")
		}
	}

	if val, ok := config.TriggerMetadata["thresholdValue"]; ok {
		value, err := strconv.ParseFloat(val, 10)
		if err != nil {
			return nil, fmt.Errorf("thresholdValue: failed to parse thresholdValue length %s", err.Error())
		}
		thresholdValue = value
	} else {
		return nil, fmt.Errorf("no threshold value given")
	}

	return &influxDBMetadata{
		authToken:        authToken,
		metricName:       metricName,
		organizationName: organizationName,
		query:            query,
		serverURL:        serverURL,
		thresholdValue:   thresholdValue,
	}, nil
}

// IsActive returns true if queried value is above the minimum value
func (s *influxDBScaler) IsActive(ctx context.Context) (bool, error) {
	queryAPI := s.client.QueryAPI(s.metadata.organizationName)

	value, err := queryInfluxDB(queryAPI, s.metadata.query)
	if err != nil {
		return false, err
	}

	return value > 0, nil
}

// Close closes the connection of the client to the server
func (s *influxDBScaler) Close() error {
	s.client.Close()
	return nil
}

// queryInfluxDB runs the query against the associated influxdb database
// there is an implicit assumption here that the first value returned from the iterator
// will be the value of interest
func queryInfluxDB(queryAPI api.QueryAPI, query string) (float64, error) {
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return 0, err
	}

	valueExists := result.Next()
	if !valueExists {
		return 0, fmt.Errorf("no results found from query")
	}

	val, ok := result.Record().Value().(float64)
	if !ok {
		return 0, fmt.Errorf("value could not be parsed into a float")
	}

	return val, nil
}

// GetMetrics connects to influxdb via the client and returns a value based on the query
func (s *influxDBScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	// Grab QueryAPI to make queries to influxdb instance
	queryAPI := s.client.QueryAPI(s.metadata.organizationName)

	value, err := queryInfluxDB(queryAPI, s.metadata.query)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(value), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// GetMetricSpecForScaling returns the metric spec for the Horizontal Pod Autoscaler
func (s *influxDBScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.thresholdValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "influxdb", s.metadata.metricName, s.metadata.organizationName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

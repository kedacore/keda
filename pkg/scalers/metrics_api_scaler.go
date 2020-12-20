package scalers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	neturl "net/url"

	"github.com/tidwall/gjson"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type metricsAPIScaler struct {
	metadata *metricsAPIScalerMetadata
	client   *http.Client
}

type metricsAPIScalerMetadata struct {
	targetValue   int
	url           string
	valueLocation string

	// apiKeyAuth
	enableAPIKeyAuth bool
	method           string // way of providing auth key, either "header" (default) or "query"
	// keyParamName  is either header key or query param used for passing apikey
	// default header is "X-API-KEY", defaul query param is "api_key"
	keyParamName string
	apiKey       string

	// base auth
	enableBaseAuth bool
	username       string
	password       string // +optional

	// client certification
	enableTLS bool
	cert      string
	key       string
	ca        string
}

type authenticationType string

const (
	apiKeyAuth       authenticationType = "apiKey"
	basicAuth        authenticationType = "basic"
	tlsAuth          authenticationType = "tls"
	methodValueQuery                    = "query"
)

var httpLog = logf.Log.WithName("metrics_api_scaler")

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseMetricsAPIMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %s", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	if meta.enableTLS {
		config, err := kedautil.NewTLSConfig(meta.cert, meta.key, meta.ca)
		if err != nil {
			return nil, err
		}

		httpClient.Transport = &http.Transport{TLSClientConfig: config}
	}

	return &metricsAPIScaler{
		metadata: meta,
		client:   httpClient,
	}, nil
}

func parseMetricsAPIMetadata(config *ScalerConfig) (*metricsAPIScalerMetadata, error) {
	meta := metricsAPIScalerMetadata{}

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	if val, ok := config.TriggerMetadata["url"]; ok {
		meta.url = val
	} else {
		return nil, fmt.Errorf("no url given in metadata")
	}

	if val, ok := config.TriggerMetadata["valueLocation"]; ok {
		meta.valueLocation = val
	} else {
		return nil, fmt.Errorf("no valueLocation given in metadata")
	}

	authMode, ok := config.TriggerMetadata["authMode"]
	// no authMode specified
	if !ok {
		return &meta, nil
	}

	authType := authenticationType(strings.TrimSpace(authMode))
	switch authType {
	case apiKeyAuth:
		if len(config.AuthParams["apiKey"]) == 0 {
			return nil, errors.New("no apikey provided")
		}

		meta.apiKey = config.AuthParams["apiKey"]
		// default behaviour is header. only change if query param requested
		meta.method = "header"
		meta.enableAPIKeyAuth = true

		if config.TriggerMetadata["method"] == methodValueQuery {
			meta.method = methodValueQuery
		}

		if len(config.TriggerMetadata["keyParamName"]) > 0 {
			meta.keyParamName = config.TriggerMetadata["keyParamName"]
		}
	case basicAuth:
		if len(config.AuthParams["username"]) == 0 {
			return nil, errors.New("no username given")
		}

		meta.username = config.AuthParams["username"]
		// password is optional. For convenience, many application implements basic auth with
		// username as apikey and password as empty
		meta.password = config.AuthParams["password"]
		meta.enableBaseAuth = true
	case tlsAuth:
		if len(config.AuthParams["ca"]) == 0 {
			return nil, errors.New("no ca given")
		}
		meta.ca = config.AuthParams["ca"]

		if len(config.AuthParams["cert"]) == 0 {
			return nil, errors.New("no cert given")
		}
		meta.cert = config.AuthParams["cert"]

		if len(config.AuthParams["key"]) == 0 {
			return nil, errors.New("no key given")
		}

		meta.key = config.AuthParams["key"]
		meta.enableTLS = true
	default:
		return nil, fmt.Errorf("err incorrect value for authMode is given: %s", authMode)
	}

	return &meta, nil
}

// GetValueFromResponse uses provided valueLocation to access the numeric value in provided body
func GetValueFromResponse(body []byte, valueLocation string) (int64, error) {
	r := gjson.GetBytes(body, valueLocation)
	if r.Type != gjson.Number {
		msg := fmt.Sprintf("valueLocation must point to value of type number got: %s", r.Type.String())
		return 0, errors.New(msg)
	}
	return int64(r.Num), nil
}

func (s *metricsAPIScaler) getMetricValue() (int64, error) {
	request, err := getMetricAPIServerRequest(s.metadata)
	if err != nil {
		return 0, err
	}

	r, err := s.client.Do(request)
	if err != nil {
		return 0, err
	}

	if r.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("api returned %d", r.StatusCode)
		return 0, errors.New(msg)
	}

	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	v, err := GetValueFromResponse(b, s.metadata.valueLocation)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Close does nothing in case of metricsAPIScaler
func (s *metricsAPIScaler) Close() error {
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *metricsAPIScaler) IsActive(ctx context.Context) (bool, error) {
	v, err := s.getMetricValue()
	if err != nil {
		httpLog.Error(err, fmt.Sprintf("Error when checking metric value: %s", err))
		return false, err
	}

	return v > 0.0, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *metricsAPIScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetValue := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	metricName := kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "http", s.metadata.url, s.metadata.valueLocation))
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: metricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *metricsAPIScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	v, err := s.getMetricValue()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error requesting metrics endpoint: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(v, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func getMetricAPIServerRequest(meta *metricsAPIScalerMetadata) (*http.Request, error) {
	var req *http.Request
	var err error

	switch {
	case meta.enableAPIKeyAuth:
		if meta.method == methodValueQuery {
			url, _ := neturl.Parse(meta.url)
			queryString := url.Query()
			if len(meta.keyParamName) == 0 {
				queryString.Set("api_key", meta.apiKey)
			} else {
				queryString.Set(meta.keyParamName, meta.apiKey)
			}

			url.RawQuery = queryString.Encode()
			req, err = http.NewRequest("GET", url.String(), nil)
			if err != nil {
				return nil, err
			}
		} else {
			// default behaviour is to use header method
			req, err = http.NewRequest("GET", meta.url, nil)
			if err != nil {
				return nil, err
			}

			if len(meta.keyParamName) == 0 {
				req.Header.Add("X-API-KEY", meta.apiKey)
			} else {
				req.Header.Add(meta.keyParamName, meta.apiKey)
			}
		}
	case meta.enableBaseAuth:
		req, err = http.NewRequest("GET", meta.url, nil)
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(meta.username, meta.password)
	default:
		req, err = http.NewRequest("GET", meta.url, nil)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

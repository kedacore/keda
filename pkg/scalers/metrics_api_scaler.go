package scalers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/tidwall/gjson"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	neturl "net/url"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"

	kedautil "github.com/kedacore/keda/pkg/util"
)

type metricsAPIScaler struct {
	metadata *metricsAPIScalerMetadata
	client   *http.Client
}

type metricsAPIScalerMetadata struct {
	targetValue   int
	url           string
	valueLocation string

	//apiKeyAuth
	enableAPIKeyAuth bool
	// +default is header
	method method
	// +option default header key is X-API-KEY and default query key is api_key
	keyParamName string
	apiKey       string

	//base auth
	enableBaseAuth bool
	username       string
	// +optional
	password string

	//client certification
	enableTLS bool
	cert      string
	key       string
	ca        string
}

type authenticationType string

const (
	apiKeyAuth authenticationType = "apiKeyAuth"
	basicAuth                     = "basicAuth"
	tlsAuth                       = "tlsAuth"
)

type method string

const (
	header     method = "header"
	queryParam        = "query"
)

var httpLog = logf.Log.WithName("metrics_api_scaler")

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := metricsAPIMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %s", err)
	}

	if meta.enableTLS {
		config, err := kedautil.NewTLSConfig(meta.cert, meta.key, meta.ca)
		if err != nil {
			return nil, err
		}

		transport := &http.Transport{TLSClientConfig: config}
		return &metricsAPIScaler{
			metadata: meta,
			client: &http.Client{
				Timeout:   3 * time.Second,
				Transport: transport,
			},
		}, nil
	}

	return &metricsAPIScaler{
		metadata: meta,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}, nil
}

func metricsAPIMetadata(metadata map[string]string) (*metricsAPIScalerMetadata, error) {
	meta := metricsAPIScalerMetadata{}

	if val, ok := metadata["targetValue"]; ok {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	if val, ok := metadata["url"]; ok {
		meta.url = val
	} else {
		return nil, fmt.Errorf("no url given in metadata")
	}

	if val, ok := metadata["valueLocation"]; ok {
		meta.valueLocation = val
	} else {
		return nil, fmt.Errorf("no valueLocation given in metadata")
	}

	// no authMode specified
	if _, ok := authParams["authMode"]; !ok {
		return &meta, nil
	}

	val, _ := authParams["authMode"]
	authType := authenticationType(strings.TrimSpace(val))
	if authType == apiKeyAuth {
		if len(authParams["apiKey"]) == 0 {
			return nil, errors.New("no apikey provided")
		}

		meta.apiKey = authParams["apiKey"]
		// default behaviour is header. only change if query param requested
		meta.method = header
		meta.enableAPIKeyAuth = true

		if authParams["method"] == queryParam {
			meta.method = queryParam
		}

		if len(authParams["keyParamName"]) > 0 {
			meta.keyParamName = authParams["keyParamName"]
		}
	} else if authType == basicAuth {
		if authParams["username"] == "" {
			return nil, errors.New("no username given")
		}

		meta.username = authParams["username"]
		// password is optional. For convenience, many application implements basic auth with
		// username as apikey and password as empty
		meta.password = authParams["password"]
		meta.enableBaseAuth = true

	} else if authType == tlsAuth {
		if authParams["ca"] == "" {
			return nil, errors.New("no ca given")
		}
		meta.ca = authParams["ca"]

		if authParams["cert"] == "" {
			return nil, errors.New("no cert given")
		}
		meta.cert = authParams["cert"]

		if authParams["key"] == "" {
			return nil, errors.New("no key given")
		}
		meta.key = authParams["key"]
		meta.enableTLS = true
	} else {
		return nil, fmt.Errorf("err incorrect value for authMode is given: %s", val)
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

	if meta.enableAPIKeyAuth {
		if header == meta.method {
			req, err = http.NewRequest("GET", meta.url, nil)
			if err != nil {
				return nil, err
			}

			if len(meta.keyParamName) == 0 {
				req.Header.Add("X-API-KEY", meta.apiKey)
			} else {
				req.Header.Add(meta.keyParamName, meta.apiKey)
			}

		} else if queryParam == meta.method {
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
		}
	} else if meta.enableBaseAuth {
		req, err = http.NewRequest("GET", meta.url, nil)
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(meta.username, meta.password)
	} else {
		req, err = http.NewRequest("GET", meta.url, nil)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type metricsAPIScaler struct {
	metricType v2.MetricTargetType
	metadata   *metricsAPIScalerMetadata
	client     *http.Client
	logger     logr.Logger
}

type metricsAPIScalerMetadata struct {
	targetValue           float64
	activationTargetValue float64
	url                   string
	valueLocation         string
	unsafeSsl             bool

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

	// bearer
	enableBearerAuth bool
	bearerToken      string

	scalerIndex int
}

const (
	methodValueQuery = "query"
)

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseMetricsAPIMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.unsafeSsl)

	if meta.enableTLS || len(meta.ca) > 0 {
		config, err := kedautil.NewTLSConfig(meta.cert, meta.key, meta.ca)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = &http.Transport{TLSClientConfig: config}
	}

	return &metricsAPIScaler{
		metricType: metricType,
		metadata:   meta,
		client:     httpClient,
		logger:     InitializeLogger(config, "metrics_api_scaler"),
	}, nil
}

func parseMetricsAPIMetadata(config *ScalerConfig) (*metricsAPIScalerMetadata, error) {
	meta := metricsAPIScalerMetadata{}
	meta.scalerIndex = config.ScalerIndex

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		unsafeSsl, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %w", err)
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %w", err)
		}
		meta.activationTargetValue = activationTargetValue
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

	authType := authentication.Type(strings.TrimSpace(authMode))
	switch authType {
	case authentication.APIKeyAuthType:
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
	case authentication.BasicAuthType:
		if len(config.AuthParams["username"]) == 0 {
			return nil, errors.New("no username given")
		}

		meta.username = config.AuthParams["username"]
		// password is optional. For convenience, many application implements basic auth with
		// username as apikey and password as empty
		meta.password = config.AuthParams["password"]
		meta.enableBaseAuth = true
	case authentication.TLSAuthType:
		if len(config.AuthParams["ca"]) == 0 {
			return nil, errors.New("no ca given")
		}

		if len(config.AuthParams["cert"]) == 0 {
			return nil, errors.New("no cert given")
		}
		meta.cert = config.AuthParams["cert"]

		if len(config.AuthParams["key"]) == 0 {
			return nil, errors.New("no key given")
		}

		meta.key = config.AuthParams["key"]
		meta.enableTLS = true
	case authentication.BearerAuthType:
		if len(config.AuthParams["token"]) == 0 {
			return nil, errors.New("no token provided")
		}

		meta.bearerToken = config.AuthParams["token"]
		meta.enableBearerAuth = true
	default:
		return nil, fmt.Errorf("err incorrect value for authMode is given: %s", authMode)
	}

	if len(config.AuthParams["ca"]) > 0 {
		meta.ca = config.AuthParams["ca"]
	}
	return &meta, nil
}

// GetValueFromResponse uses provided valueLocation to access the numeric value in provided body
func GetValueFromResponse(body []byte, valueLocation string) (float64, error) {
	r := gjson.GetBytes(body, valueLocation)
	errorMsg := "valueLocation must point to value of type number or a string representing a Quantity got: '%s'"
	if r.Type == gjson.String {
		v, err := resource.ParseQuantity(r.String())
		if err != nil {
			return 0, fmt.Errorf(errorMsg, r.String())
		}
		return v.AsApproximateFloat64(), nil
	}
	if r.Type != gjson.Number {
		return 0, fmt.Errorf(errorMsg, r.Type.String())
	}
	return r.Num, nil
}

func (s *metricsAPIScaler) getMetricValue(ctx context.Context) (float64, error) {
	request, err := getMetricAPIServerRequest(ctx, s.metadata)
	if err != nil {
		return 0, err
	}

	r, err := s.client.Do(request)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
		return 0, errors.New(msg)
	}

	b, err := io.ReadAll(r.Body)
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
func (s *metricsAPIScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *metricsAPIScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("metric-api-%s", s.metadata.valueLocation))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *metricsAPIScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error requesting metrics endpoint: %w", err)
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationTargetValue, nil
}

func getMetricAPIServerRequest(ctx context.Context, meta *metricsAPIScalerMetadata) (*http.Request, error) {
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
			req, err = http.NewRequestWithContext(ctx, "GET", url.String(), nil)
			if err != nil {
				return nil, err
			}
		} else {
			// default behaviour is to use header method
			req, err = http.NewRequestWithContext(ctx, "GET", meta.url, nil)
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
		req, err = http.NewRequestWithContext(ctx, "GET", meta.url, nil)
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(meta.username, meta.password)
	case meta.enableBearerAuth:
		req, err = http.NewRequestWithContext(ctx, "GET", meta.url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", meta.bearerToken))
	default:
		req, err = http.NewRequestWithContext(ctx, "GET", meta.url, nil)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

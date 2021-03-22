package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultValueWhenError          = 0
	metricDefaultHTTPClientTimeout = 30
)

/* expected structure declarations */

type openstackMetricMetadata struct {
	metricsURL        string
	metricID          string
	aggregationMethod string
	granularity       int
	threshold         float64
	timeout           int
}

type openstackMetricAuthenticationMetadata struct {
	userID                string
	password              string
	authURL               string
	appCredentialSecret   string
	appCredentialSecretID string
}

type openstackMetricScaler struct {
	metadata     *openstackMetricMetadata
	metricClient openstack.Client
}

type measureResult struct {
	measures [][]interface{}
}

/*  end of declarations */

var openstackMetricLog = logf.Log.WithName("openstack_metric_scaler")

// NewOpenstackMetricScaler creates new openstack metrics scaler instance
func NewOpenstackMetricScaler(config *ScalerConfig) (Scaler, error) {
	var keystoneAuth *openstack.KeystoneAuthRequest
	var metricsClient openstack.Client

	openstackMetricMetadata, err := parseOpenstackMetricMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing openstack Metric metadata: %s", err)
	}

	authMetadata, err := parseOpenstackMetricAuthenticationMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing openstack metric authentication metadata: %s", err)
	}

	// User choose the "application_credentials" authentication method
	if authMetadata.appCredentialSecretID != "" {
		keystoneAuth, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialSecretID, authMetadata.appCredentialSecret, openstackMetricMetadata.timeout)

		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %s", err)
		}
	} else {
		// User choose the "password" authentication method
		if authMetadata.userID != "" {
			keystoneAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, "", openstackMetricMetadata.timeout)

			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %s", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	metricsClient, err = keystoneAuth.RequestClient()
	if err != nil {
		openstackMetricLog.Error(err, "Fail to retrieve new keystone clinet for openstack metrics scaler")
		return nil, err
	}

	return &openstackMetricScaler{
		metadata:     openstackMetricMetadata,
		metricClient: metricsClient,
	}, nil
}

func parseOpenstackMetricMetadata(config *ScalerConfig) (*openstackMetricMetadata, error) {
	meta := openstackMetricMetadata{}
	triggerMetadata := config.TriggerMetadata

	if val, ok := triggerMetadata["metricsURL"]; ok && val != "" {
		meta.metricsURL = val
	} else {
		openstackMetricLog.Error(fmt.Errorf("no metrics url could be read"), "Error readig metricsURL")
		return nil, fmt.Errorf("no metrics url was declared")
	}

	if val, ok := triggerMetadata["metricID"]; ok && val != "" {
		meta.metricID = val
	} else {
		openstackMetricLog.Error(fmt.Errorf("no metric id could be read"), "Error reading metricID")
		return nil, fmt.Errorf("no metric id was declared")
	}

	if val, ok := triggerMetadata["aggregationMethod"]; ok && val != "" {
		meta.aggregationMethod = val
	} else {
		openstackMetricLog.Error(fmt.Errorf("no aggregation method could be read"), "Error reading aggregation method")
		return nil, fmt.Errorf("no aggregation method could be read")
	}

	if val, ok := triggerMetadata["granularity"]; ok && val != "" {
		granularity, err := strconv.Atoi(val)
		if err != nil {
			openstackMetricLog.Error(err, "Error converting granulality information %s", err.Error)
			return nil, err
		}
		meta.granularity = granularity
	} else {
		return nil, fmt.Errorf("no granularity found")
	}

	if val, ok := triggerMetadata["threshold"]; ok && val != "" {
		// converts the string to float64 but its value is convertible to float32 without changing
		_threshold, err := strconv.ParseFloat(val, 32)
		if err != nil {
			openstackMetricLog.Error(err, "error parsing openstack metric metadata", "threshold", "threshold")
			return nil, fmt.Errorf("error parsing openstack metric metadata : %s", err.Error())
		}

		meta.threshold = _threshold
	}

	if val, ok := triggerMetadata["timeout"]; ok && val != "" {
		httpClientTimeout, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("httpClientTimeout parsing error: %s", err.Error())
		}
		meta.timeout = httpClientTimeout
	} else {
		meta.timeout = metricDefaultHTTPClientTimeout
	}

	return &meta, nil
}

func parseOpenstackMetricAuthenticationMetadata(config *ScalerConfig) (openstackMetricAuthenticationMetadata, error) {
	authMeta := openstackMetricAuthenticationMetadata{}
	authParams := config.AuthParams

	if val, ok := authParams["authURL"]; ok && val != "" {
		authMeta.authURL = authParams["authURL"]
	} else {
		return authMeta, fmt.Errorf("authURL doesn't exist in the authParams")
	}

	if val, ok := authParams["userID"]; ok && val != "" {
		authMeta.userID = val

		if val, ok := authParams["password"]; ok && val != "" {
			authMeta.password = val
		} else {
			return authMeta, fmt.Errorf("password doesn't exist in the authParams")
		}
	} else if val, ok := authParams["appCredentialSecret"]; ok && val != "" {
		authMeta.appCredentialSecretID = val
	} else {
		return authMeta, fmt.Errorf("neither userID or appCredentialSecretID exist in the authParams")
	}

	return authMeta, nil
}

func (a *openstackMetricScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(a.metadata.threshold), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("openstack-metric-%s", a.metadata.aggregationMethod)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricVal,
		},
	}

	metricSpec := v2beta2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

func (a *openstackMetricScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := a.readOpenstackMetrics()

	if err != nil {
		openstackMetricLog.Error(err, "Error collecting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (a *openstackMetricScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := a.readOpenstackMetrics()

	if err != nil {
		return false, err
	}

	return val > 0, nil
}

func (a *openstackMetricScaler) Close() error {
	return nil
}

// Gets measureament from API as float64, converts it to int and return the value.
func (a *openstackMetricScaler) readOpenstackMetrics() (float64, error) {
	var metricURL string = a.metadata.metricsURL

	isValid, validationError := a.metricClient.IsTokenValid()

	if validationError != nil {
		openstackMetricLog.Error(validationError, "Unable to check token validity.")
		return 0, validationError
	}

	if !isValid {
		tokenRequestError := a.metricClient.RenewToken()
		if tokenRequestError != nil {
			openstackMetricLog.Error(tokenRequestError, "The token being used is invalid")
			return defaultValueWhenError, tokenRequestError
		}
	}

	token := a.metricClient.Token

	openstackMetricsURL, err := url.Parse(metricURL)

	if err != nil {
		openstackMetricLog.Error(err, "metric url provided is invalid")
		return defaultValueWhenError, fmt.Errorf("metric url is invalid: %s", err.Error())
	}

	openstackMetricsURL.Path = path.Join(openstackMetricsURL.Path, a.metadata.metricID+"/measures")
	queryParameter := openstackMetricsURL.Query()
	granularity := 0 // We start with granularity with value 2 cause gnocchi APIm which is used by openstack, consider a time window, and we want to get the last value

	if a.metadata.granularity <= 0 {
		openstackMetricLog.Error(fmt.Errorf("granularity value is less than 1"), "Minimum accepatble value expected for ganularity is 1.")
		return defaultValueWhenError, fmt.Errorf("granularity value is less than 1")
	}

	if (a.metadata.granularity / 60) > 0 {
		granularity = (a.metadata.granularity / 60) - 1
	}

	queryParameter.Set("granularity", strconv.Itoa(a.metadata.granularity))
	queryParameter.Set("aggregation", a.metadata.aggregationMethod)

	var currTimeWithWindow string

	if granularity > 0 {
		currTimeWithWindow = time.Now().Add(time.Minute * time.Duration(granularity)).Format(time.RFC3339)
	} else {
		currTimeWithWindow = time.Now().Format(time.RFC3339)
	}

	queryParameter.Set("start", currTimeWithWindow[:17]+"00")

	openstackMetricsURL.RawQuery = queryParameter.Encode()

	openstackMetricRequest, newReqErr := http.NewRequest("GET", openstackMetricsURL.String(), nil)
	if newReqErr != nil {
		openstackMetricLog.Error(newReqErr, "Could not build metrics request", nil)
	}
	openstackMetricRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := a.metricClient.HTTPClient.Do(openstackMetricRequest)

	if requestError != nil {
		openstackMetricLog.Error(requestError, "Unable to request Metrics from URL: %s.", a.metadata.metricsURL)
		return defaultValueWhenError, requestError
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyError, readError := ioutil.ReadAll(resp.Body)

		if readError != nil {
			openstackMetricLog.Error(readError, "Request failed with code: %s for URL: %s", resp.StatusCode, a.metadata.metricsURL)
			return defaultValueWhenError, readError
		}

		return defaultValueWhenError, fmt.Errorf(string(bodyError))
	}

	m := measureResult{}
	body, errConvertJSON := ioutil.ReadAll(resp.Body)

	if errConvertJSON != nil {
		openstackMetricLog.Error(errConvertJSON, "Failed to convert Body format response to json")
		return defaultValueWhenError, err
	}

	if body == nil {
		return defaultValueWhenError, nil
	}

	errUnMarshall := json.Unmarshal(body, &m.measures)

	if errUnMarshall != nil {
		openstackMetricLog.Error(errUnMarshall, "Failed converting json format Body structure.")
		return defaultValueWhenError, errUnMarshall
	}

	var targetMeasure []interface{}

	if len(m.measures) > 0 {
		targetMeasure = m.measures[len(m.measures)-1]
	} else {
		openstackMetricLog.Info("No measure was returned from openstack")
		return defaultValueWhenError, nil
	}

	if len(targetMeasure) != 3 {
		openstackMetricLog.Error(fmt.Errorf("unexpected json response"), "unexpected json tuple, expected structure is [string, float, float]")
		return defaultValueWhenError, fmt.Errorf("unexpected json response")
	}

	if val, ok := targetMeasure[2].(float64); ok {
		return val, nil
	}

	openstackMetricLog.Error(fmt.Errorf("failed to convert interface type to float64"), "unable to convert target measure to expected format float64")
	return defaultValueWhenError, fmt.Errorf("failed to convert interface type to float64")
}

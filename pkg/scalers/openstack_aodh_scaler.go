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
	defaultValueWhenError        = 0
	aodhDefaultHTTPClientTimeout = 30
)

/* expected structure declarations */

type aodhMetadata struct {
	metricsURL        string
	metricID          string
	aggregationMethod string
	granularity       int
	threshold         float64
	timeout           int
}

type aodhAuthenticationMetadata struct {
	userID                string
	password              string
	authURL               string
	appCredentialSecret   string
	appCredentialSecretID string
}

type aodhScaler struct {
	metadata     *aodhMetadata
	authMetadata *openstack.Client
}

type measureResult struct {
	measures [][]interface{}
}

/*  end of declarations */

var aodhLog = logf.Log.WithName("aodh_scaler")

// NewOpenstackAodhScaler creates new AODH openstack scaler instance
func NewOpenstackAodhScaler(config *ScalerConfig) (Scaler, error) {
	var keystoneAuth *openstack.Client

	aodhMetadata, err := parseAodhMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing AODH metadata: %s", err)
	}

	authMetadata, err := parseAodhAuthenticationMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing AODH authentication metadata: %s", err)
	}

	// User choose the "application_credentials" authentication method
	if authMetadata.appCredentialSecretID != "" {
		keystoneAuth, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialSecretID, authMetadata.appCredentialSecret, aodhMetadata.timeout)

		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %s", err)
		}

	} else {
		// User choose the "password" authentication method
		if authMetadata.userID != "" {
			keystoneAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, "", aodhMetadata.timeout)

			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %s", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	return &aodhScaler{
		metadata:     aodhMetadata,
		authMetadata: keystoneAuth,
	}, nil
}

func parseAodhMetadata(config *ScalerConfig) (*aodhMetadata, error) {
	meta := aodhMetadata{}
	triggerMetadata := config.TriggerMetadata

	if val, ok := triggerMetadata["metricsURL"]; ok && val != "" {
		meta.metricsURL = val
	} else {
		aodhLog.Error(fmt.Errorf("No metricsURL could be read"), "Error readig metricsURL")
		return nil, fmt.Errorf("No metricsURL was declared")
	}

	if val, ok := triggerMetadata["metricID"]; ok && val != "" {
		meta.metricID = val
	} else {
		aodhLog.Error(fmt.Errorf("No metricID could be read"), "Error reading metricID")
		return nil, fmt.Errorf("No metricID was declared")
	}

	if val, ok := triggerMetadata["aggregationMethod"]; ok && val != "" {
		meta.aggregationMethod = val
	} else {
		aodhLog.Error(fmt.Errorf("No aggregationMethod could be read"), "Error reading aggregation method")
		return nil, fmt.Errorf("No aggregationMethod could be read")
	}

	if val, ok := triggerMetadata["granularity"]; ok && val != "" {
		if granularity, err := strconv.Atoi(val); err != nil {
			if err != nil {
				aodhLog.Error(err, "Error converting granulality information %s", err.Error)
				return nil, err
			}

			meta.granularity = granularity

		}

	} else {
		return nil, fmt.Errorf("No granularity found")
	}

	if val, ok := triggerMetadata["threshold"]; ok && val != "" {
		// converts the string to float64 but its value is convertible to float32 without changing
		_threshold, err := strconv.ParseFloat(val, 32)
		if err != nil {
			aodhLog.Error(err, "Error parsing AODH metadata", "threshold", "threshold")
			return nil, fmt.Errorf("Error parsing AODH metadata : %s", err.Error())
		}

		meta.threshold = _threshold
	}

	return &meta, nil
}

func parseAodhAuthenticationMetadata(config *ScalerConfig) (aodhAuthenticationMetadata, error) {
	authMeta := aodhAuthenticationMetadata{}
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

	} else if val, ok := authParams["appCredentialSecretId"]; ok && val != "" {
		authMeta.appCredentialSecretID = val

		if val, ok := authParams["appCredentialSecret"]; ok && val != "" {
			authMeta.appCredentialSecret = val
		}

	} else {
		return authMeta, fmt.Errorf("neither userID or appCredentialSecretID exist in the authParams")
	}

	return authMeta, nil
}

func (a *aodhScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(a.metadata.threshold), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			//aux1, err := fmt.Sprintf("%s-%s", "openstack-AODH", a.authMetadata.AuthURL)
			Name: kedautil.NormalizeString(fmt.Sprintf("openstack-aodh-%s", a.metadata.aggregationMethod)),
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

func (a *aodhScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := a.readOpenstackMetrics()

	if err != nil {
		aodhLog.Error(err, "Error collecting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (a *aodhScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := a.readOpenstackMetrics()

	if err != nil {
		return false, err
	}

	return val > 0, nil
}

func (a *aodhScaler) Close() error {
	return nil
}

// Gets measureament from API as float64, converts it to int and return the value.
func (a *aodhScaler) readOpenstackMetrics() (float64, error) {

	var token string = ""
	var metricURL string = a.metadata.metricsURL

	isValid, validationError := openstack.IsTokenValid(*a.authMetadata)

	if validationError != nil {
		aodhLog.Error(validationError, "Unable to check token validity.")
		return 0, validationError
	}

	if !isValid {
		var tokenRequestError error
		token, tokenRequestError = a.authMetadata.GetToken()
		a.authMetadata.AuthToken = token
		if tokenRequestError != nil {
			aodhLog.Error(tokenRequestError, "The token being used is invalid")
			return defaultValueWhenError, tokenRequestError
		}
	}

	token = a.authMetadata.AuthToken

	aodhMetricsURL, err := url.Parse(metricURL)

	if err != nil {
		aodhLog.Error(err, "The metrics URL provided is invalid")
		return defaultValueWhenError, fmt.Errorf("The metrics URL is invalid: %s", err.Error())
	}

	aodhMetricsURL.Path = path.Join(aodhMetricsURL.Path, a.metadata.metricID+"/measures")
	queryParameter := aodhMetricsURL.Query()
	granularity := 2 // We start with granularity with value 2 cause gnocchi APIm which is used by openstack, consider a time window, and we want to get the last value

	if a.metadata.granularity <= 0 {
		aodhLog.Error(fmt.Errorf("Granularity Value is less than 1"), "Minimum accepatble value expected for ganularity is 1.")
		return defaultValueWhenError, fmt.Errorf("Granularity Value is less than 1")
	}

	if (a.metadata.granularity / 60) > 1 {
		granularity = (a.metadata.granularity / 60)
	}

	granularity--

	queryParameter.Set("granularity", strconv.Itoa(a.metadata.granularity))
	queryParameter.Set("aggregation", a.metadata.aggregationMethod)

	currTimeWithWindow := time.Now().Add(time.Minute + time.Duration(granularity)).Format(time.RFC3339)
	queryParameter.Set("start", string(currTimeWithWindow)[:17]+"00")

	aodhMetricsURL.RawQuery = queryParameter.Encode()

	aodhRequest, newReqErr := http.NewRequest("GET", aodhMetricsURL.String(), nil)
	if newReqErr != nil {
		aodhLog.Error(newReqErr, "Could not build metrics request", nil)
	}
	aodhRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := a.authMetadata.HTTPClient.Do(aodhRequest)

	if requestError != nil {
		aodhLog.Error(requestError, "Unable to request Metrics from URL: %s.", a.metadata.metricsURL)
		return defaultValueWhenError, requestError
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyError, readError := ioutil.ReadAll(resp.Body)

		if readError != nil {
			aodhLog.Error(readError, "Request failed with code: %s for URL: %s", resp.StatusCode, a.metadata.metricsURL)
			return defaultValueWhenError, readError
		}

		return defaultValueWhenError, fmt.Errorf(string(bodyError))
	}

	m := measureResult{}
	body, errConvertJSON := ioutil.ReadAll(resp.Body)

	if errConvertJSON != nil {
		aodhLog.Error(errConvertJSON, "Failed to convert Body format response to json")
		return defaultValueWhenError, err
	}

	if body == nil {
		return defaultValueWhenError, nil
	}

	errUnMarshall := json.Unmarshal([]byte(body), &m.measures)

	if errUnMarshall != nil {
		aodhLog.Error(errUnMarshall, "Failed converting json format Body structure.")
		return defaultValueWhenError, errUnMarshall
	}

	var targetMeasure []interface{} = nil

	if len(m.measures) > 1 {
		targetMeasure = m.measures[len(m.measures)-1]
	}

	if len(targetMeasure) != 3 {
		aodhLog.Error(fmt.Errorf("Unexpected json response"), "Unexpected json tuple, expected structure is [string, float, float].")
		return defaultValueWhenError, fmt.Errorf("Unexpected json response")
	}

	if val, ok := targetMeasure[2].(float64); ok {
		return val, nil
	}

	aodhLog.Error(fmt.Errorf("Failed to convert interface type to flaot64"), "Unable to convert targetMeasure to expected format float64")
	return defaultValueWhenError, fmt.Errorf("Failed to convert interface type to flaot64")

}

package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultValueWhenError          = 0
	metricDefaultHTTPClientTimeout = 30
)

/* expected structure declarations */

type openstackMetricMetadata struct {
	MetricsURL          string  `keda:"name=metricsURL,          order=triggerMetadata"`
	MetricID            string  `keda:"name=metricID,            order=triggerMetadata"`
	AggregationMethod   string  `keda:"name=aggregationMethod,   order=triggerMetadata"`
	Granularity         int     `keda:"name=granularity,         order=triggerMetadata"`
	Threshold           float64 `keda:"name=threshold,           order=triggerMetadata, optional"`
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, optional"`
	Timeout             int     `keda:"name=timeout,             order=triggerMetadata, default=30"`

	triggerIndex int
}

type openstackMetricAuthenticationMetadata struct {
	userID                string
	password              string
	authURL               string
	appCredentialSecret   string
	appCredentialSecretID string
}

type openstackMetricScaler struct {
	metricType   v2.MetricTargetType
	metadata     *openstackMetricMetadata
	metricClient openstack.Client
	logger       logr.Logger
}

type measureResult struct {
	measures [][]interface{}
}

/*  end of declarations */

// NewOpenstackMetricScaler creates a new openstack metrics scaler instance
func NewOpenstackMetricScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	var keystoneAuth *openstack.KeystoneAuthRequest
	var metricsClient openstack.Client

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "openstack_metric_scaler")

	openstackMetricMetadata, err := parseOpenstackMetricMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing openstack Metric metadata: %w", err)
	}

	authMetadata, err := parseOpenstackMetricAuthenticationMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing openstack metric authentication metadata: %w", err)
	}

	if authMetadata.appCredentialSecretID != "" {
		keystoneAuth, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialSecretID, authMetadata.appCredentialSecret, openstackMetricMetadata.Timeout)

		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %w", err)
		}
	} else {
		if authMetadata.userID != "" {
			keystoneAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, "", openstackMetricMetadata.Timeout)

			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	metricsClient, err = keystoneAuth.RequestClient(ctx)
	if err != nil {
		logger.Error(err, "Fail to retrieve new keystone client for openstack metrics scaler")
		return nil, err
	}

	return &openstackMetricScaler{
		metricType:   metricType,
		metadata:     openstackMetricMetadata,
		metricClient: metricsClient,
		logger:       logger,
	}, nil
}

func parseOpenstackMetricMetadata(config *scalersconfig.ScalerConfig) (*openstackMetricMetadata, error) {
	meta := &openstackMetricMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing openstack metric metadata: %w", err)
	}

	return meta, nil
}

func parseOpenstackMetricAuthenticationMetadata(config *scalersconfig.ScalerConfig) (openstackMetricAuthenticationMetadata, error) {
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

func (s *openstackMetricScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("openstack-metric-%s", s.metadata.MetricID))

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

func (s *openstackMetricScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.readOpenstackMetrics(ctx)

	if err != nil {
		s.logger.Error(err, "Error collecting metric value")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

func (s *openstackMetricScaler) Close(context.Context) error {
	s.metricClient.HTTPClient.CloseIdleConnections()
	return nil
}

// Gets measurement from API as float64, converts it to int and return the value.
func (s *openstackMetricScaler) readOpenstackMetrics(ctx context.Context) (float64, error) {
	var metricURL = s.metadata.MetricsURL

	isValid, validationError := s.metricClient.IsTokenValid(ctx)

	if validationError != nil {
		s.logger.Error(validationError, "Unable to check token validity.")
		return 0, validationError
	}

	if !isValid {
		tokenRequestError := s.metricClient.RenewToken(ctx)
		if tokenRequestError != nil {
			s.logger.Error(tokenRequestError, "The token being used is invalid")
			return defaultValueWhenError, tokenRequestError
		}
	}

	token := s.metricClient.Token

	openstackMetricsURL, err := url.Parse(metricURL)

	if err != nil {
		s.logger.Error(err, "metric url provided is invalid")
		return defaultValueWhenError, fmt.Errorf("metric url is invalid: %w", err)
	}

	openstackMetricsURL.Path = path.Join(openstackMetricsURL.Path, s.metadata.MetricID+"/measures")
	queryParameter := openstackMetricsURL.Query()
	granularity := 0 // We start with granularity with value 2 cause gnocchi APIm which is used by openstack, consider a time window, and we want to get the last value

	if s.metadata.Granularity <= 0 {
		s.logger.Error(fmt.Errorf("granularity value is less than 1"), "Minimum acceptable value expected for granularity is 1.")
		return defaultValueWhenError, fmt.Errorf("granularity value is less than 1")
	}

	if (s.metadata.Granularity / 60) > 0 {
		granularity = (s.metadata.Granularity / 60) - 1
	}

	queryParameter.Set("granularity", strconv.Itoa(s.metadata.Granularity))
	queryParameter.Set("aggregation", s.metadata.AggregationMethod)

	var currTimeWithWindow string

	if granularity > 0 {
		currTimeWithWindow = time.Now().Add(time.Minute * time.Duration(granularity)).Format(time.RFC3339)
	} else {
		currTimeWithWindow = time.Now().Format(time.RFC3339)
	}

	queryParameter.Set("start", currTimeWithWindow[:17]+"00")

	openstackMetricsURL.RawQuery = queryParameter.Encode()

	openstackMetricRequest, newReqErr := http.NewRequestWithContext(ctx, "GET", openstackMetricsURL.String(), nil)
	if newReqErr != nil {
		s.logger.Error(newReqErr, "Could not build metrics request", nil)
	}
	openstackMetricRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := s.metricClient.HTTPClient.Do(openstackMetricRequest)

	if requestError != nil {
		s.logger.Error(requestError, "Unable to request Metrics from URL: %s.", s.metadata.MetricsURL)
		return defaultValueWhenError, requestError
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyError, readError := io.ReadAll(resp.Body)

		if readError != nil {
			s.logger.Error(readError, "Request failed with code: %s for URL: %s", resp.StatusCode, s.metadata.MetricsURL)
			return defaultValueWhenError, readError
		}

		return defaultValueWhenError, fmt.Errorf("%s", string(bodyError))
	}

	m := measureResult{}
	body, errConvertJSON := io.ReadAll(resp.Body)

	if errConvertJSON != nil {
		s.logger.Error(errConvertJSON, "Failed to convert Body format response to json")
		return defaultValueWhenError, err
	}

	if body == nil {
		return defaultValueWhenError, nil
	}

	errUnMarshall := json.Unmarshal(body, &m.measures)

	if errUnMarshall != nil {
		s.logger.Error(errUnMarshall, "Failed converting json format Body structure.")
		return defaultValueWhenError, errUnMarshall
	}

	var targetMeasure []interface{}

	if len(m.measures) > 0 {
		targetMeasure = m.measures[len(m.measures)-1]
	} else {
		s.logger.Info("No measure was returned from openstack")
		return defaultValueWhenError, nil
	}

	if len(targetMeasure) != 3 {
		s.logger.Error(fmt.Errorf("unexpected json response"), "unexpected json tuple, expected structure is [string, float, float]")
		return defaultValueWhenError, fmt.Errorf("unexpected json response")
	}

	if val, ok := targetMeasure[2].(float64); ok {
		return val, nil
	}

	s.logger.Error(fmt.Errorf("failed to convert interface type to float64"), "unable to convert target measure to expected format float64")
	return defaultValueWhenError, fmt.Errorf("failed to convert interface type to float64")
}

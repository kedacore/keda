package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/splunk"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	splunkDefaultHTTPTimeout = 10 * time.Second
)

// SplunkScaler assigns struct data pointer to metadata variable
type SplunkScaler struct {
	client     *splunk.Client
	metricType v2.MetricTargetType
	metadata   SplunkMetadata
	logger     logr.Logger
}

// SplunkMetadata Metadata used by KEDA to search Splunk events and scale
type SplunkMetadata struct {
	apiToken        string
	password        string
	username        string
	host            string
	httpTimeout     time.Duration
	verifyTLS       bool
	targetValue     int
	activationValue int
	savedSearchName string
	valueField      string
	triggerIndex    int
}

func NewSplunkScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseSplunkMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Splunk metadata: %w", err)
	}

	client, err := splunk.NewClient(&splunk.Config{
		APIToken:  meta.apiToken,
		Password:  meta.password,
		Username:  meta.username,
		Host:      meta.host,
		VerifyTLS: meta.verifyTLS,
	})
	if err != nil {
		return nil, err
	}

	return &SplunkScaler{
		client:     client,
		metricType: metricType,
		logger:     InitializeLogger(config, "splunk_scaler"),
		metadata:   *meta,
	}, nil
}

func (s *SplunkScaler) Close(context.Context) error {
	return nil
}

func parseSplunkMetadata(config *scalersconfig.ScalerConfig) (*SplunkMetadata, error) {
	meta := &SplunkMetadata{}

	if apiToken, ok := config.AuthParams["apiToken"]; ok {
		meta.apiToken = apiToken
	}

	if password, ok := config.AuthParams["password"]; ok {
		meta.password = password
	}

	username, ok := config.TriggerMetadata["username"]
	if !ok {
		return nil, errors.New("no username given")
	}
	meta.username = username

	hostStr, ok := config.TriggerMetadata["host"]
	if !ok {
		return nil, errors.New("no host given")
	}

	hostURI, err := url.ParseRequestURI(hostStr)
	if err != nil {
		return nil, errors.New("invalid value for host. Must be a valid URL such as https://localhost:8089")
	}
	meta.host = hostURI.String()

	meta.httpTimeout = splunkDefaultHTTPTimeout
	if httpTimeout, ok := config.TriggerMetadata["httpTimeout"]; ok {
		parsedHTTPTimeout, err := time.ParseDuration(httpTimeout)
		if err != nil {
			return nil, errors.New("invalid value for httpTimeout")
		}
		meta.httpTimeout = parsedHTTPTimeout
	}

	if verifyTLS, ok := config.TriggerMetadata["verifyTLS"]; !ok {
		meta.verifyTLS = true
	} else {
		parsedVerifyTLS, err := strconv.ParseBool(verifyTLS)
		if err != nil {
			return nil, errors.New("invalid value for verifyTLS")
		}
		meta.verifyTLS = parsedVerifyTLS
	}

	targetValueStr, ok := config.TriggerMetadata["targetValue"]
	if !ok {
		return nil, errors.New("no targetValue given")
	}

	targetValue, err := strconv.Atoi(targetValueStr)
	if err != nil {
		return meta, errors.New("invalid value for targetValue. Must be an int")
	}
	meta.targetValue = targetValue

	activationValueStr, ok := config.TriggerMetadata["activationValue"]
	if !ok {
		return nil, errors.New("no activationValue given")
	}

	activationValue, err := strconv.Atoi(activationValueStr)
	if err != nil {
		return nil, errors.New("invalid value for activationValue. Must be an int")
	}
	meta.activationValue = activationValue

	savedSearchName, ok := config.TriggerMetadata["savedSearchName"]
	if !ok {
		return nil, errors.New("no savedSearchName given")
	}
	meta.savedSearchName = savedSearchName

	valueField, ok := config.TriggerMetadata["valueField"]
	if !ok {
		return nil, errors.New("no valueField given")
	}
	meta.valueField = valueField

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *SplunkScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("splunk-%s", s.metadata.savedSearchName))),
		},
		Target: GetMetricTarget(s.metricType, int64(s.metadata.targetValue)),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *SplunkScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	response, err := s.client.SavedSearch(s.metadata.savedSearchName)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error fetching saved search data from splunk: %w", err)
	}

	metricValue, err := response.ToMetric(s.metadata.valueField)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error finding metric value field: %w", err)
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, int(metricValue) > s.metadata.activationValue, nil
}

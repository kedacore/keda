package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	"github.com/kedacore/keda/v2/keda-scalers/splunk"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
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
	APIToken        string `keda:"name=apiToken,        order=authParams, optional"`
	Password        string `keda:"name=password,        order=authParams, optional"`
	Username        string `keda:"name=username,        order=authParams"`
	Host            string `keda:"name=host,            order=triggerMetadata"`
	UnsafeSsl       bool   `keda:"name=unsafeSsl,       order=triggerMetadata, optional"`
	TargetValue     int    `keda:"name=targetValue,     order=triggerMetadata"`
	ActivationValue int    `keda:"name=activationValue, order=triggerMetadata"`
	SavedSearchName string `keda:"name=savedSearchName, order=triggerMetadata"`
	ValueField      string `keda:"name=valueField,      order=triggerMetadata"`
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
		APIToken:  meta.APIToken,
		Password:  meta.Password,
		Username:  meta.Username,
		Host:      meta.Host,
		UnsafeSsl: meta.UnsafeSsl,
	}, config)
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
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing splunk metadata: %w", err)
	}

	_, err := url.ParseRequestURI(meta.Host)
	if err != nil {
		return meta, errors.New("invalid value for host. Must be a valid URL such as https://localhost:8089")
	}

	return meta, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *SplunkScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("splunk-%s", s.metadata.SavedSearchName))),
		},
		Target: GetMetricTarget(s.metricType, int64(s.metadata.TargetValue)),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *SplunkScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	response, err := s.client.SavedSearch(s.metadata.SavedSearchName)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error fetching saved search data from splunk: %w", err)
	}

	metricValue, err := response.ToMetric(s.metadata.ValueField)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error finding metric value field: %w", err)
	}

	metric := GenerateMetricInMili(metricName, metricValue)
	return []external_metrics.ExternalMetricValue{metric}, int(metricValue) > s.metadata.ActivationValue, nil
}

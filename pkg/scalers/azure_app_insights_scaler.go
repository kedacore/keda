package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	azureAppInsightsMetricID                      = "metricId"
	azureAppInsightsTargetValueName               = "targetValue"
	azureAppInsightsActivationTargetValueName     = "activationTargetValue"
	azureAppInsightsAppIDName                     = "applicationInsightsId"
	azureAppInsightsMetricAggregationTimespanName = "metricAggregationTimespan"
	azureAppInsightsMetricAggregationTypeName     = "metricAggregationType"
	azureAppInsightsMetricFilterName              = "metricFilter"
	azureAppInsightsTenantIDName                  = "tenantId"
)

type azureAppInsightsMetadata struct {
	azureAppInsightsInfo  azure.AppInsightsInfo
	targetValue           float64
	activationTargetValue float64
	scalerIndex           int
}

type azureAppInsightsScaler struct {
	metricType  v2.MetricTargetType
	metadata    *azureAppInsightsMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

// NewAzureAppInsightsScaler creates a new AzureAppInsightsScaler
func NewAzureAppInsightsScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_app_insights_scaler")

	meta, err := parseAzureAppInsightsMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure app insights metadata: %w", err)
	}

	return &azureAppInsightsScaler{
		metricType:  metricType,
		metadata:    meta,
		podIdentity: config.PodIdentity,
		logger:      logger,
	}, nil
}

func parseAzureAppInsightsMetadata(config *ScalerConfig, logger logr.Logger) (*azureAppInsightsMetadata, error) {
	meta := azureAppInsightsMetadata{
		azureAppInsightsInfo: azure.AppInsightsInfo{},
	}

	val, err := getParameterFromConfig(config, azureAppInsightsTargetValueName, false)
	if err != nil {
		return nil, err
	}
	targetValue, err := strconv.ParseFloat(val, 64)
	if err != nil {
		logger.Error(err, "Error parsing azure app insights metadata", azureAppInsightsTargetValueName, azureAppInsightsTargetValueName)
		return nil, fmt.Errorf("error parsing azure app insights metadata %s: %w", azureAppInsightsTargetValueName, err)
	}
	meta.targetValue = targetValue

	meta.activationTargetValue = 0
	val, err = getParameterFromConfig(config, azureAppInsightsActivationTargetValueName, false)
	if err == nil {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure app insights metadata", azureAppInsightsActivationTargetValueName, azureAppInsightsActivationTargetValueName)
			return nil, fmt.Errorf("error parsing azure app insights metadata %s: %w", azureAppInsightsActivationTargetValueName, err)
		}
		meta.activationTargetValue = activationTargetValue
	}

	val, err = getParameterFromConfig(config, azureAppInsightsMetricID, false)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.MetricID = val

	val, err = getParameterFromConfig(config, azureAppInsightsMetricAggregationTimespanName, false)
	if err != nil {
		return nil, err
	}
	aggregationTimespan := strings.Split(val, ":")
	if len(aggregationTimespan) != 2 {
		return nil, fmt.Errorf("%s not in the correct format. Should be hh:mm", azureAppInsightsMetricAggregationTimespanName)
	}
	meta.azureAppInsightsInfo.AggregationTimespan = val

	val, err = getParameterFromConfig(config, azureAppInsightsMetricAggregationTypeName, false)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.AggregationType = val

	if val, ok := config.TriggerMetadata[azureAppInsightsMetricFilterName]; ok && val != "" {
		meta.azureAppInsightsInfo.Filter = val
	} else {
		meta.azureAppInsightsInfo.Filter = ""
	}

	meta.azureAppInsightsInfo.AppInsightsResourceURL = azure.DefaultAppInsightsResourceURL

	if cloud, ok := config.TriggerMetadata["cloud"]; ok {
		if strings.EqualFold(cloud, azure.PrivateCloud) {
			if resource, ok := config.TriggerMetadata["appInsightsResourceURL"]; ok && resource != "" {
				meta.azureAppInsightsInfo.AppInsightsResourceURL = resource
			} else {
				return nil, fmt.Errorf("appInsightsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
		} else if resource, ok := azure.AppInsightsResourceURLInCloud[strings.ToUpper(cloud)]; ok {
			meta.azureAppInsightsInfo.AppInsightsResourceURL = resource
		} else {
			return nil, fmt.Errorf("there is no cloud environment matching the name %s", cloud)
		}
	}

	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.ActiveDirectoryEndpoint = activeDirectoryEndpoint

	// Required authentication parameters below

	val, err = getParameterFromConfig(config, azureAppInsightsAppIDName, true)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.ApplicationInsightsID = val

	val, err = getParameterFromConfig(config, azureAppInsightsTenantIDName, true)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.TenantID = val

	clientID, clientPassword, err := parseAzurePodIdentityParams(config)
	if err != nil {
		return nil, err
	}
	meta.azureAppInsightsInfo.ClientID = clientID
	meta.azureAppInsightsInfo.ClientPassword = clientPassword

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func (s *azureAppInsightsScaler) Close(context.Context) error {
	return nil
}

func (s *azureAppInsightsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-app-insights-%s", s.metadata.azureAppInsightsInfo.MetricID))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureAppInsightsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := azure.GetAzureAppInsightsMetricValue(ctx, s.metadata.azureAppInsightsInfo, s.podIdentity)
	if err != nil {
		s.logger.Error(err, "error getting azure app insights metric")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationTargetValue, nil
}

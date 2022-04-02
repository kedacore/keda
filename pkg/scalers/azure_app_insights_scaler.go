package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	azureAppInsightsMetricID                      = "metricId"
	azureAppInsightsTargetValueName               = "targetValue"
	azureAppInsightsAppIDName                     = "applicationInsightsId"
	azureAppInsightsMetricAggregationTimespanName = "metricAggregationTimespan"
	azureAppInsightsMetricAggregationTypeName     = "metricAggregationType"
	azureAppInsightsMetricFilterName              = "metricFilter"
	azureAppInsightsTenantIDName                  = "tenantId"
)

type azureAppInsightsMetadata struct {
	azureAppInsightsInfo azure.AppInsightsInfo
	targetValue          int64
	scalerIndex          int
}

var azureAppInsightsLog = logf.Log.WithName("azure_app_insights_scaler")

type azureAppInsightsScaler struct {
	metricType  v2beta2.MetricTargetType
	metadata    *azureAppInsightsMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
}

// NewAzureAppInsightsScaler creates a new AzureAppInsightsScaler
func NewAzureAppInsightsScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAzureAppInsightsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure app insights metadata: %s", err)
	}

	return &azureAppInsightsScaler{
		metricType:  metricType,
		metadata:    meta,
		podIdentity: config.PodIdentity,
	}, nil
}

func parseAzureAppInsightsMetadata(config *ScalerConfig) (*azureAppInsightsMetadata, error) {
	meta := azureAppInsightsMetadata{
		azureAppInsightsInfo: azure.AppInsightsInfo{},
	}

	val, err := getParameterFromConfig(config, azureAppInsightsTargetValueName, false)
	if err != nil {
		return nil, err
	}
	meta.targetValue, err = strconv.ParseInt(val, 10, 64)
	if err != nil {
		azureAppInsightsLog.Error(err, "Error parsing azure app insights metadata", "targetValue", targetValueName)
		return nil, fmt.Errorf("error parsing azure app insights metadata %s: %s", targetValueName, err.Error())
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

// Returns true if the Azure App Insights metric value is greater than the target value
func (s *azureAppInsightsScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := azure.GetAzureAppInsightsMetricValue(ctx, s.metadata.azureAppInsightsInfo, s.podIdentity)
	if err != nil {
		azureAppInsightsLog.Error(err, "error getting azure app insights metric")
		return false, err
	}

	return val > 0, nil
}

func (s *azureAppInsightsScaler) Close(context.Context) error {
	return nil
}

func (s *azureAppInsightsScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-app-insights-%s", s.metadata.azureAppInsightsInfo.MetricID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureAppInsightsScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := azure.GetAzureAppInsightsMetricValue(ctx, s.metadata.azureAppInsightsInfo, s.podIdentity)
	if err != nil {
		azureAppInsightsLog.Error(err, "error getting azure app insights metric")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(val, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

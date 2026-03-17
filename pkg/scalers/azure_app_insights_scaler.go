package scalers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	azureAppInsightsIgnoreNullValues              = "ignoreNullValues"
)

type azureAppInsightsMetadata struct {
	TargetValue           float64 `keda:"name=targetValue,order=triggerMetadata,optional"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue,order=triggerMetadata,default=0"`

	MetricID            string `keda:"name=metricId,order=triggerMetadata"`
	AggregationTimespan string `keda:"name=metricAggregationTimespan,order=triggerMetadata"`
	AggregationType     string `keda:"name=metricAggregationType,order=triggerMetadata"`
	Filter              string `keda:"name=metricFilter,order=triggerMetadata,optional"`

	IgnoreNullValues bool `keda:"name=ignoreNullValues,order=triggerMetadata,default=false"`

	Cloud                       string `keda:"name=cloud,order=triggerMetadata,optional"`
	AppInsightsResourceURLParam string `keda:"name=appInsightsResourceURL,order=triggerMetadata,optional"`

	ApplicationInsightsID string `keda:"name=applicationInsightsId,order=triggerMetadata;authParams;resolvedEnv"`
	TenantID              string `keda:"name=tenantId,order=triggerMetadata;authParams;resolvedEnv"`

	ClientID       string `keda:"name=activeDirectoryClientId,order=triggerMetadata;authParams;resolvedEnv,optional"`
	ClientPassword string `keda:"name=activeDirectoryClientPassword,order=authParams;resolvedEnv,optional"`

	azureAppInsightsInfo azure.AppInsightsInfo
	triggerIndex         int
}

func (m *azureAppInsightsMetadata) Validate() error {
	aggregationTimespan := strings.Split(m.AggregationTimespan, ":")
	if len(aggregationTimespan) != 2 {
		return fmt.Errorf("%s not in the correct format. Should be hh:mm", azureAppInsightsMetricAggregationTimespanName)
	}
	return nil
}

type azureAppInsightsScaler struct {
	metricType  v2.MetricTargetType
	metadata    *azureAppInsightsMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
	httpClient  *http.Client
	creds       azcore.TokenCredential
}

// NewAzureAppInsightsScaler creates a new AzureAppInsightsScaler
func NewAzureAppInsightsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_app_insights_scaler")

	meta, err := parseAzureAppInsightsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure app insights metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	creds, err := getAuthConfig(meta.azureAppInsightsInfo, config.PodIdentity)
	if err != nil {
		return nil, fmt.Errorf("error getting auth config: %w", err)
	}

	return &azureAppInsightsScaler{
		metricType:  metricType,
		metadata:    meta,
		podIdentity: config.PodIdentity,
		logger:      logger,
		httpClient:  httpClient,
		creds:       creds,
	}, nil
}

func getAuthConfig(info azure.AppInsightsInfo, podIdentity kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		return azidentity.NewClientSecretCredential(info.TenantID, info.ClientID, info.ClientPassword, nil)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		return azure.NewADWorkloadIdentityCredential(podIdentity.GetIdentityID(), podIdentity.GetIdentityTenantID())
	default:
		return nil, fmt.Errorf("unknown pod identity provider: %s", podIdentity.Provider)
	}
}

func parseAzureAppInsightsMetadata(config *scalersconfig.ScalerConfig) (*azureAppInsightsMetadata, error) {
	meta := azureAppInsightsMetadata{}
	if err := config.TypedConfig(&meta); err != nil {
		return nil, fmt.Errorf("error parsing azure app insights metadata: %w", err)
	}

	if meta.TargetValue == 0 && !config.AsMetricSource {
		return nil, fmt.Errorf("no %s given", azureAppInsightsTargetValueName)
	}

	// Resolve AppInsights resource URL based on cloud
	meta.azureAppInsightsInfo.AppInsightsResourceURL = azure.DefaultAppInsightsResourceURL
	if meta.Cloud != "" {
		if strings.EqualFold(meta.Cloud, azure.PrivateCloud) {
			if meta.AppInsightsResourceURLParam == "" {
				return nil, fmt.Errorf("appInsightsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
			meta.azureAppInsightsInfo.AppInsightsResourceURL = meta.AppInsightsResourceURLParam
		} else if resource, ok := azure.AppInsightsResourceURLInCloud[strings.ToUpper(meta.Cloud)]; ok {
			meta.azureAppInsightsInfo.AppInsightsResourceURL = resource
		} else {
			return nil, fmt.Errorf("there is no cloud environment matching the name %s", meta.Cloud)
		}
	}

	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}

	if err := parsePodIdentityParams(config, &meta); err != nil {
		return nil, err
	}

	meta.azureAppInsightsInfo = azure.AppInsightsInfo{
		MetricID:                meta.MetricID,
		AggregationTimespan:     meta.AggregationTimespan,
		AggregationType:         meta.AggregationType,
		Filter:                  meta.Filter,
		AppInsightsResourceURL:  meta.azureAppInsightsInfo.AppInsightsResourceURL,
		ActiveDirectoryEndpoint: activeDirectoryEndpoint,
		ApplicationInsightsID:   meta.ApplicationInsightsID,
		TenantID:                meta.TenantID,
		ClientID:                meta.ClientID,
		ClientPassword:          meta.ClientPassword,
	}

	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

// parsePodIdentityParams validates and resolves clientID/clientPassword based on pod identity provider
func parsePodIdentityParams(config *scalersconfig.ScalerConfig, meta *azureAppInsightsMetadata) error {
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if meta.ClientID == "" {
			return fmt.Errorf("no activeDirectoryClientId given")
		}
		if meta.ClientPassword == "" {
			return fmt.Errorf("no activeDirectoryClientPassword given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// no params required
	default:
		return fmt.Errorf("azure Monitor doesn't support pod identity %s", config.PodIdentity.Provider)
	}
	return nil
}

func (s *azureAppInsightsScaler) Close(context.Context) error {
	return nil
}

func (s *azureAppInsightsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-app-insights-%s", s.metadata.azureAppInsightsInfo.MetricID))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureAppInsightsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := azure.GetAzureAppInsightsMetricValue(ctx, s.metadata.azureAppInsightsInfo, s.metadata.IgnoreNullValues, s.httpClient, s.creds)
	if err != nil {
		s.logger.Error(err, "error getting azure app insights metric")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationTargetValue, nil
}

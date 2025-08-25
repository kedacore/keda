package azure

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	DefaultAppInsightsResourceURL = "https://api.applicationinsights.io"
)

var AppInsightsResourceURLInCloud = map[string]string{
	"AZUREPUBLICCLOUD":       "https://api.applicationinsights.io",
	"AZUREUSGOVERNMENTCLOUD": "https://api.applicationinsights.us",
	"AZURECHINACLOUD":        "https://api.applicationinsights.azure.cn",
}

type AppInsightsInfo struct {
	ApplicationInsightsID   string
	TenantID                string
	MetricID                string
	AggregationTimespan     string
	AggregationType         string
	Filter                  string
	ClientID                string
	ClientPassword          string
	AppInsightsResourceURL  string
	ActiveDirectoryEndpoint string
}

type ApplicationInsightsMetric struct {
	Value map[string]interface{}
}

var azureAppInsightsLog = logf.Log.WithName("azure_app_insights_scaler")

func toISO8601(time string) (string, error) {
	timeSegments := strings.Split(time, ":")
	if len(timeSegments) != 2 {
		return "", fmt.Errorf("invalid interval %s", time)
	}

	hours, herr := strconv.Atoi(timeSegments[0])
	minutes, merr := strconv.Atoi(timeSegments[1])

	if herr != nil || merr != nil {
		return "", fmt.Errorf("errors parsing time: %v, %w", herr, merr)
	}

	return fmt.Sprintf("PT%02dH%02dM", hours, minutes), nil
}

func getAuthConfig(ctx context.Context, info AppInsightsInfo, podIdentity kedav1alpha1.AuthPodIdentity) auth.AuthorizerConfig {
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		config := auth.NewClientCredentialsConfig(info.ClientID, info.ClientPassword, info.TenantID)
		config.Resource = info.AppInsightsResourceURL
		config.AADEndpoint = info.ActiveDirectoryEndpoint
		return config
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		return NewAzureADWorkloadIdentityConfig(ctx, podIdentity.GetIdentityID(), podIdentity.GetIdentityTenantID(), podIdentity.GetIdentityAuthorityHost(), info.AppInsightsResourceURL)
	}
	return nil
}

func extractAppInsightValue(info AppInsightsInfo, metric ApplicationInsightsMetric) (float64, error) {
	if _, ok := metric.Value[info.MetricID]; !ok {
		return -1, fmt.Errorf("metric named %s not found in app insights response", info.MetricID)
	}

	floatVal := 0.0
	if val, ok := metric.Value[info.MetricID].(map[string]interface{})[info.AggregationType]; ok {
		if val == nil {
			return -1, fmt.Errorf("metric %s was nil for aggregation type %s", info.MetricID, info.AggregationType)
		}
		floatVal = val.(float64)
	} else {
		return -1, fmt.Errorf("metric %s did not contain aggregation type %s", info.MetricID, info.AggregationType)
	}

	azureAppInsightsLog.V(2).Info("value extracted from metric request", "metric type", info.AggregationType, "metric value", floatVal)

	return floatVal, nil
}

func queryParamsForAppInsightsRequest(info AppInsightsInfo) (map[string]interface{}, error) {
	timespan, err := toISO8601(info.AggregationTimespan)
	if err != nil {
		return nil, err
	}

	queryParams := map[string]interface{}{
		"aggregation": info.AggregationType,
		"timespan":    timespan,
	}
	if info.Filter != "" {
		queryParams["filter"] = info.Filter
	}

	return queryParams, nil
}

// GetAzureAppInsightsMetricValue returns the value of an Azure App Insights metric, rounded to the nearest int
func GetAzureAppInsightsMetricValue(ctx context.Context, info AppInsightsInfo, podIdentity kedav1alpha1.AuthPodIdentity, ignoreNullValues bool) (float64, error) {
	config := getAuthConfig(ctx, info, podIdentity)
	authorizer, err := config.Authorizer()
	if err != nil {
		return -1, err
	}

	queryParams, err := queryParamsForAppInsightsRequest(info)
	if err != nil {
		return -1, err
	}

	req, err := autorest.Prepare(&http.Request{},
		autorest.WithBaseURL(info.AppInsightsResourceURL),
		autorest.WithPath("v1/apps"),
		autorest.WithPath(info.ApplicationInsightsID),
		autorest.WithPath("metrics"),
		autorest.WithPath(info.MetricID),
		autorest.WithQueryParameters(queryParams),
		authorizer.WithAuthorization())
	if err != nil {
		return -1, err
	}

	resp, err := autorest.Send(req,
		autorest.DoErrorUnlessStatusCode(http.StatusOK),
		autorest.DoCloseIfError())
	if err != nil {
		return -1, err
	}

	metric := &ApplicationInsightsMetric{}
	err = autorest.Respond(resp,
		autorest.ByUnmarshallingJSON(metric),
		autorest.ByClosing())
	if err != nil {
		return -1, err
	}

	val, err := extractAppInsightValue(info, *metric)
	if err != nil && ignoreNullValues {
		return 0.0, nil
	}
	return val, err
}

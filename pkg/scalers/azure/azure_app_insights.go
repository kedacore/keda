package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
func GetAzureAppInsightsMetricValue(ctx context.Context, info AppInsightsInfo, ignoreNullValues bool,
	httpClient *http.Client, creds azcore.TokenCredential) (float64, error) {
	queryParams, err := queryParamsForAppInsightsRequest(info)
	if err != nil {
		return -1, err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/apps/%s/metrics/%s", info.AppInsightsResourceURL,
		info.ApplicationInsightsID, info.MetricID), nil)
	if err != nil {
		return -1, err
	}

	bearerToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{getScopedResource(info.AppInsightsResourceURL)}})
	if err != nil {
		return -1, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken.Token))

	parameters := mapToValues(queryParams)
	v := req.URL.Query()
	for key, value := range parameters {
		for i := range value {
			d, err := url.QueryUnescape(value[i])
			if err != nil {
				return -1, err
			}
			value[i] = d
		}
		v[key] = value
	}
	req.URL.RawQuery = v.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("app insights request failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	metric := &ApplicationInsightsMetric{}
	err = json.Unmarshal(body, metric)
	if err != nil {
		return -1, err
	}

	val, err := extractAppInsightValue(info, *metric)
	if err != nil && ignoreNullValues {
		return 0.0, nil
	}
	return val, err
}

// mapToValues method converts map[string]interface{} to url.Values.
func mapToValues(m map[string]interface{}) url.Values {
	v := url.Values{}
	for key, value := range m {
		x := reflect.ValueOf(value)
		if x.Kind() == reflect.Array || x.Kind() == reflect.Slice {
			for i := 0; i < x.Len(); i++ {
				v.Add(key, ensureValueString(x.Index(i)))
			}
		} else {
			v.Add(key, ensureValueString(value))
		}
	}
	return v
}

func ensureValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

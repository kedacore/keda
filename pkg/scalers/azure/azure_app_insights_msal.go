package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

var azureAppInsightsMSALLog = logf.Log.WithName("azure_app_insights_msal_scaler")

// ApplicationInsightsMetric represents a metric response from App Insights
type ApplicationInsightsMetric struct {
	Value map[string]interface{}
}

// MSALAppInsightsClient handles App Insights requests using pure MSAL authentication
type MSALAppInsightsClient struct {
	httpClient *http.Client
	credential azcore.TokenCredential
	info       AppInsightsInfo
}

// NewMSALAppInsightsClient creates a new client using MSAL authentication
func NewMSALAppInsightsClient(ctx context.Context, info AppInsightsInfo, podIdentity kedav1alpha1.AuthPodIdentity) (*MSALAppInsightsClient, error) {
	var credential azcore.TokenCredential
	var err error

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Use client credentials flow with MSAL
		credential, err = createClientCredentialsMSAL(info)
		if err != nil {
			return nil, fmt.Errorf("failed to create client credentials: %w", err)
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// Use Azure Workload Identity with azidentity
		options := &azidentity.WorkloadIdentityCredentialOptions{}
		if identityID := podIdentity.GetIdentityID(); identityID != "" {
			options.ClientID = identityID
		}
		if identityTenantID := podIdentity.GetIdentityTenantID(); identityTenantID != "" {
			options.TenantID = identityTenantID
		}
		credential, err = azidentity.NewWorkloadIdentityCredential(options)
		if err != nil {
			return nil, fmt.Errorf("failed to create workload identity credential: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported pod identity provider: %s", podIdentity.Provider)
	}

	return &MSALAppInsightsClient{
		httpClient: &http.Client{},
		credential: credential,
		info:       info,
	}, nil
}

// createClientCredentialsMSAL creates an MSAL client credentials flow
func createClientCredentialsMSAL(info AppInsightsInfo) (azcore.TokenCredential, error) {
	// For client credentials, we use azidentity.ClientSecretCredential
	// which internally uses MSAL
	options := &azidentity.ClientSecretCredentialOptions{}
	if info.ActiveDirectoryEndpoint != "" {
		options.AuthorityHost = info.ActiveDirectoryEndpoint
	}

	return azidentity.NewClientSecretCredential(
		info.TenantID,
		info.ClientID,
		info.ClientPassword,
		options,
	)
}

// GetMetricValue retrieves a metric value from App Insights using MSAL authentication
func (c *MSALAppInsightsClient) GetMetricValue(ctx context.Context, ignoreNullValues bool) (float64, error) {
	// Get access token
	scope := fmt.Sprintf("%s/.default", c.info.AppInsightsResourceURL)
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to get access token: %w", err)
	}

	// Prepare query parameters
	queryParams, err := queryParamsForAppInsightsRequest(c.info)
	if err != nil {
		return -1, err
	}

	// Build URL
	baseURL := fmt.Sprintf("%s/v1/apps/%s/metrics/%s",
		c.info.AppInsightsResourceURL,
		c.info.ApplicationInsightsID,
		c.info.MetricID)

	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return -1, err
	}

	// Add query parameters
	q := reqURL.Query()
	for key, value := range queryParams {
		q.Add(key, fmt.Sprintf("%v", value))
	}
	reqURL.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return -1, err
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return -1, fmt.Errorf("App Insights API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var metric ApplicationInsightsMetric
	if err := json.NewDecoder(resp.Body).Decode(&metric); err != nil {
		return -1, fmt.Errorf("failed to decode response: %w", err)
	}

	val, err := extractAppInsightValue(c.info, metric)
	if err != nil && ignoreNullValues {
		return 0.0, nil
	}
	return val, err
}

// GetAzureAppInsightsMetricValueMSAL is the new MSAL-only function that replaces the ADAL version
func GetAzureAppInsightsMetricValueMSAL(ctx context.Context, info AppInsightsInfo, podIdentity kedav1alpha1.AuthPodIdentity, ignoreNullValues bool) (float64, error) {
	client, err := NewMSALAppInsightsClient(ctx, info, podIdentity)
	if err != nil {
		return -1, err
	}

	return client.GetMetricValue(ctx, ignoreNullValues)
}

// toISO8601 converts time format for App Insights API
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

// extractAppInsightValue extracts the metric value from App Insights response
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

	azureAppInsightsMSALLog.V(2).Info("value extracted from metric request", "metric type", info.AggregationType, "metric value", floatVal)

	return floatVal, nil
}

// queryParamsForAppInsightsRequest prepares query parameters for App Insights API request
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

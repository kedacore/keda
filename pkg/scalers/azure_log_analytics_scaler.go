/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/v4/auth"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	aadTokenEndpoint               = "%s/%s/oauth2/token"
	laQueryEndpoint                = "%s/v1/workspaces/%s/query"
	defaultLogAnalyticsResourceURL = "https://api.loganalytics.io"
)

type azureLogAnalyticsScaler struct {
	metricType v2.MetricTargetType
	metadata   *azureLogAnalyticsMetadata
	name       string
	namespace  string
	httpClient *http.Client
	logger     logr.Logger
}

type azureLogAnalyticsMetadata struct {
	tenantID                string
	clientID                string
	clientSecret            string
	workspaceID             string
	podIdentity             kedav1alpha1.AuthPodIdentity
	query                   string
	threshold               float64
	activationThreshold     float64
	triggerIndex            int
	logAnalyticsResourceURL string
	activeDirectoryEndpoint string
	unsafeSsl               bool
}

type tokenData struct {
	TokenType               string `json:"token_type"`
	ExpiresIn               int    `json:"expires_in,string"`
	ExtExpiresIn            int    `json:"ext_expires_in,string"`
	ExpiresOn               int64  `json:"expires_on,string"`
	NotBefore               int64  `json:"not_before,string"`
	Resource                string `json:"resource"`
	AccessToken             string `json:"access_token"`
	IsWorkloadIdentityToken bool   `json:"isWorkloadIdentityToken"`
}

type metricsData struct {
	value     float64
	threshold float64
}

type queryResult struct {
	Tables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
		Rows [][]interface{} `json:"rows"`
	} `json:"tables"`
}

var tokenCache = struct {
	sync.RWMutex
	m map[string]tokenData
}{m: make(map[string]tokenData)}

var logAnalyticsResourceURLInCloud = map[string]string{
	"AZUREPUBLICCLOUD":       "https://api.loganalytics.io",
	"AZUREUSGOVERNMENTCLOUD": "https://api.loganalytics.us",
	"AZURECHINACLOUD":        "https://api.loganalytics.azure.cn",
}

// NewAzureLogAnalyticsScaler creates a new Azure Log Analytics Scaler
func NewAzureLogAnalyticsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	azureLogAnalyticsMetadata, err := parseAzureLogAnalyticsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Log Analytics scaler. Scaled object: %s. Namespace: %s. Inner Error: %w", config.ScalableObjectName, config.ScalableObjectNamespace, err)
	}

	useSsl := azureLogAnalyticsMetadata.unsafeSsl

	return &azureLogAnalyticsScaler{
		metricType: metricType,
		metadata:   azureLogAnalyticsMetadata,
		name:       config.ScalableObjectName,
		namespace:  config.ScalableObjectNamespace,
		httpClient: kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, useSsl),
		logger:     InitializeLogger(config, "azure_log_analytics_scaler"),
	}, nil
}

func parseAzureLogAnalyticsMetadata(config *scalersconfig.ScalerConfig) (*azureLogAnalyticsMetadata, error) {
	meta := azureLogAnalyticsMetadata{}
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Getting tenantId
		tenantID, err := getParameterFromConfig(config, "tenantId", true)
		if err != nil {
			return nil, err
		}
		meta.tenantID = tenantID

		// Getting clientId
		clientID, err := getParameterFromConfig(config, "clientId", true)
		if err != nil {
			return nil, err
		}
		meta.clientID = clientID

		// Getting clientSecret
		clientSecret, err := getParameterFromConfig(config, "clientSecret", true)
		if err != nil {
			return nil, err
		}
		meta.clientSecret = clientSecret

		meta.podIdentity = config.PodIdentity
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		meta.podIdentity = config.PodIdentity
	default:
		return nil, fmt.Errorf("error parsing metadata. Details: Log Analytics Scaler doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	// Getting workspaceId
	workspaceID, err := getParameterFromConfig(config, "workspaceId", true)
	if err != nil {
		return nil, err
	}
	meta.workspaceID = workspaceID

	// Getting query, observe that we dont check AuthParams for query
	query, err := getParameterFromConfig(config, "query", false)
	if err != nil {
		return nil, err
	}
	meta.query = query

	// Getting threshold, observe that we don't check AuthParams for threshold
	val, err := getParameterFromConfig(config, "threshold", false)
	if err != nil {
		if config.AsMetricSource {
			val = "0"
		} else {
			return nil, err
		}
	}
	threshold, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %w", err)
	}
	meta.threshold = threshold

	// Getting activationThreshold
	meta.activationThreshold = 0
	val, err = getParameterFromConfig(config, "activationThreshold", false)
	if err == nil {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %w", err)
		}
		meta.activationThreshold = activationThreshold
	}
	meta.triggerIndex = config.TriggerIndex

	meta.logAnalyticsResourceURL = defaultLogAnalyticsResourceURL
	if cloud, ok := config.TriggerMetadata["cloud"]; ok {
		if strings.EqualFold(cloud, azure.PrivateCloud) {
			if resource, ok := config.TriggerMetadata["logAnalyticsResourceURL"]; ok && resource != "" {
				meta.logAnalyticsResourceURL = resource
			} else {
				return nil, fmt.Errorf("logAnalyticsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
		} else if resource, ok := logAnalyticsResourceURLInCloud[strings.ToUpper(cloud)]; ok {
			meta.logAnalyticsResourceURL = resource
		} else {
			return nil, fmt.Errorf("there is no cloud environment matching the name %s", cloud)
		}
	}

	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	meta.activeDirectoryEndpoint = activeDirectoryEndpoint

	// Getting unsafeSsl, observe that we don't check AuthParams for unsafeSsl
	meta.unsafeSsl = false
	unsafeSslVal, err := getParameterFromConfig(config, "unsafeSsl", false)
	if err == nil {
		unsafeSsl, err := strconv.ParseBool(unsafeSslVal)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse unsafeSsl. Inner Error: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	return &meta, nil
}

// getParameterFromConfig gets the parameter from the configs, if checkAuthParams is true
// then AuthParams is also check for the parameter
func getParameterFromConfig(config *scalersconfig.ScalerConfig, parameter string, checkAuthParams bool) (string, error) {
	if val, ok := config.AuthParams[parameter]; checkAuthParams && ok && val != "" {
		return val, nil
	} else if val, ok := config.TriggerMetadata[parameter]; ok && val != "" {
		return val, nil
	} else if val, ok := config.TriggerMetadata[fmt.Sprintf("%sFromEnv", parameter)]; ok && val != "" {
		return config.ResolvedEnv[config.TriggerMetadata[fmt.Sprintf("%sFromEnv", parameter)]], nil
	}
	return "", fmt.Errorf("error parsing metadata. Details: %s was not found in metadata. Check your ScaledObject configuration", parameter)
}

func (s *azureLogAnalyticsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-log-analytics", s.metadata.workspaceID))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureLogAnalyticsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	receivedMetric, err := s.getMetricData(ctx)

	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to get metrics. Scaled object: %s. Namespace: %s. Inner Error: %w", s.name, s.namespace, err)
	}

	metric := GenerateMetricInMili(metricName, receivedMetric.value)

	return []external_metrics.ExternalMetricValue{metric}, receivedMetric.value > s.metadata.activationThreshold, nil
}

func (s *azureLogAnalyticsScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *azureLogAnalyticsScaler) getMetricData(ctx context.Context) (metricsData, error) {
	tokenInfo, err := s.getAccessToken(ctx)
	if err != nil {
		return metricsData{}, err
	}

	metricsInfo, err := s.executeQuery(ctx, s.metadata.query, tokenInfo)
	if err != nil {
		return metricsData{}, err
	}

	s.logger.V(1).Info("Providing metric value", "metrics value", metricsInfo.value, "scaler name", s.name, "namespace", s.namespace)

	return metricsInfo, nil
}

func (s *azureLogAnalyticsScaler) getAccessToken(ctx context.Context) (tokenData, error) {
	// if there is no token yet or it will be expired in less, that 30 secs
	currentTimeSec := time.Now().Unix()
	tokenInfo := tokenData{}

	switch s.metadata.podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		tokenInfo, _ = getTokenFromCache(s.metadata.clientID, s.metadata.clientSecret)
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		tokenInfo, _ = getTokenFromCache(string(s.metadata.podIdentity.Provider), string(s.metadata.podIdentity.Provider))
	}

	if currentTimeSec+30 > tokenInfo.ExpiresOn {
		newTokenInfo, err := s.refreshAccessToken(ctx)
		if err != nil {
			return tokenData{}, err
		}

		switch s.metadata.podIdentity.Provider {
		case "", kedav1alpha1.PodIdentityProviderNone:
			s.logger.V(1).Info("Token for Service Principal has been refreshed", "clientID", s.metadata.clientID, "scaler name", s.name, "namespace", s.namespace)
			_ = setTokenInCache(s.metadata.clientID, s.metadata.clientSecret, newTokenInfo)
		case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
			s.logger.V(1).Info("Token for Pod Identity has been refreshed", "type", s.metadata.podIdentity, "scaler name", s.name, "namespace", s.namespace)
			_ = setTokenInCache(string(s.metadata.podIdentity.Provider), string(s.metadata.podIdentity.Provider), newTokenInfo)
		}

		return newTokenInfo, nil
	}
	return tokenInfo, nil
}

func (s *azureLogAnalyticsScaler) executeQuery(ctx context.Context, query string, tokenInfo tokenData) (metricsData, error) {
	queryData := queryResult{}
	var body []byte
	var statusCode int
	var err error

	body, statusCode, err = s.executeLogAnalyticsREST(ctx, query, tokenInfo)

	// Handle expired token
	if statusCode == 403 || (len(body) > 0 && strings.Contains(string(body), "TokenExpired")) {
		tokenInfo, err = s.refreshAccessToken(ctx)
		if err != nil {
			return metricsData{}, err
		}

		switch s.metadata.podIdentity.Provider {
		case "", kedav1alpha1.PodIdentityProviderNone:
			s.logger.V(1).Info("Token for Service Principal has been refreshed", "clientID", s.metadata.clientID, "scaler name", s.name, "namespace", s.namespace)
			_ = setTokenInCache(s.metadata.clientID, s.metadata.clientSecret, tokenInfo)
		case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
			s.logger.V(1).Info("Token for Pod Identity has been refreshed", "type", s.metadata.podIdentity, "scaler name", s.name, "namespace", s.namespace)
			_ = setTokenInCache(string(s.metadata.podIdentity.Provider), string(s.metadata.podIdentity.Provider), tokenInfo)
		}

		if err == nil {
			body, statusCode, err = s.executeLogAnalyticsREST(ctx, query, tokenInfo)
		} else {
			return metricsData{}, err
		}
	}

	if statusCode != 200 && statusCode != 0 {
		return metricsData{}, fmt.Errorf("error processing Log Analytics request. HTTP code %d. Inner Error: %v. Body: %s", statusCode, err, string(body))
	}

	if err != nil {
		return metricsData{}, err
	}

	if len(body) == 0 {
		return metricsData{}, fmt.Errorf("error processing Log Analytics request. Details: empty body. HTTP code: %d", statusCode)
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&queryData)
	if err != nil {
		return metricsData{}, fmt.Errorf("error processing Log Analytics request. Details: can't decode response body to JSON from REST API result. HTTP code: %d. Inner Error: %v. Body: %s", statusCode, err, string(body))
	}

	if statusCode == 200 {
		metricsInfo := metricsData{}
		metricsInfo.threshold = s.metadata.threshold
		metricsInfo.value = 0

		// Pre-validation of query result:
		switch {
		case len(queryData.Tables) == 0 || len(queryData.Tables[0].Columns) == 0 || len(queryData.Tables[0].Rows) == 0:
			return metricsData{}, fmt.Errorf("error validating Log Analytics request. Details: there is no results after running your query. HTTP code: %d. Body: %s", statusCode, string(body))
		case len(queryData.Tables) > 1:
			return metricsData{}, fmt.Errorf("error validating Log Analytics request. Details: too many tables in query result: %d, expected: 1. HTTP code: %d. Body: %s", len(queryData.Tables), statusCode, string(body))
		case len(queryData.Tables[0].Rows) > 1:
			return metricsData{}, fmt.Errorf("error validating Log Analytics request. Details: too many rows in query result: %d, expected: 1. HTTP code: %d. Body: %s", len(queryData.Tables[0].Rows), statusCode, string(body))
		}

		if len(queryData.Tables[0].Rows[0]) > 0 {
			metricDataType := queryData.Tables[0].Columns[0].Type
			metricVal := queryData.Tables[0].Rows[0][0]
			parsedMetricVal, err := parseTableValueToFloat64(metricVal, metricDataType)
			if err != nil {
				return metricsData{}, fmt.Errorf("%s. HTTP code: %d. Body: %s", err.Error(), statusCode, string(body))
			}
			metricsInfo.value = parsedMetricVal
		}

		if len(queryData.Tables[0].Rows[0]) > 1 {
			thresholdDataType := queryData.Tables[0].Columns[1].Type
			thresholdVal := queryData.Tables[0].Rows[0][1]
			parsedThresholdVal, err := parseTableValueToFloat64(thresholdVal, thresholdDataType)
			if err != nil {
				return metricsData{}, fmt.Errorf("%s. HTTP code: %d. Body: %s", err.Error(), statusCode, string(body))
			}
			metricsInfo.threshold = parsedThresholdVal
		} else {
			metricsInfo.threshold = -1
		}

		return metricsInfo, nil
	}

	return metricsData{}, fmt.Errorf("error processing Log Analytics request. Details: unknown error. HTTP code: %d. Body: %s", statusCode, string(body))
}

func parseTableValueToFloat64(value interface{}, dataType string) (float64, error) {
	if value != nil {
		// type can be: real, int, long
		if dataType == "real" || dataType == "int" || dataType == "long" {
			convertedValue, isConverted := value.(float64)
			if !isConverted {
				return 0, fmt.Errorf("error validating Log Analytics request. Details: cannot convert result to type float64")
			}
			if convertedValue < 0 {
				return 0, fmt.Errorf("error validating Log Analytics request. Details: value should be >=0, but received %f", value)
			}
			return convertedValue, nil
		}
		return 0, fmt.Errorf("error validating Log Analytics request. Details: value data type should be real, int or long, but received %s", dataType)
	}
	return 0, fmt.Errorf("error validating Log Analytics request. Details: value is empty, check your query")
}

func (s *azureLogAnalyticsScaler) refreshAccessToken(ctx context.Context) (tokenData, error) {
	tokenInfo, err := s.getAuthorizationToken(ctx)

	if err != nil {
		return tokenData{}, err
	}

	if tokenInfo.IsWorkloadIdentityToken {
		return tokenInfo, nil
	}

	// Now, let's check we can use this token. If no, wait until we can use it
	currentTimeSec := time.Now().Unix()
	if currentTimeSec < tokenInfo.NotBefore {
		if currentTimeSec < tokenInfo.NotBefore+10 {
			sleepDurationSec := int(tokenInfo.NotBefore - currentTimeSec + 1)
			s.logger.V(1).Info("AAD token not ready", "delay (seconds)", sleepDurationSec, "scaler name", s.name, "namespace", s.namespace)
			time.Sleep(time.Duration(sleepDurationSec) * time.Second)
		} else {
			return tokenData{}, fmt.Errorf("error getting access token. Details: AAD token has been received, but start date begins in %d seconds, so current operation will be skipped", tokenInfo.NotBefore-currentTimeSec)
		}
	}

	return tokenInfo, nil
}

func (s *azureLogAnalyticsScaler) getAuthorizationToken(ctx context.Context) (tokenData, error) {
	var body []byte
	var statusCode int
	var err error
	var tokenInfo tokenData

	switch s.metadata.podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		aadToken, err := azure.GetAzureADWorkloadIdentityToken(ctx, s.metadata.podIdentity.GetIdentityID(), s.metadata.podIdentity.GetIdentityTenantID(), s.metadata.podIdentity.GetIdentityAuthorityHost(), s.metadata.logAnalyticsResourceURL)
		if err != nil {
			return tokenData{}, nil
		}

		expiresOn := aadToken.ExpiresOnTimeObject.Unix()
		if err != nil {
			return tokenData{}, nil
		}

		tokenInfo = tokenData{
			TokenType:               string(auth.CBSTokenTypeJWT),
			AccessToken:             aadToken.AccessToken,
			ExpiresOn:               expiresOn,
			Resource:                s.metadata.logAnalyticsResourceURL,
			IsWorkloadIdentityToken: true,
		}

		return tokenInfo, nil
	case "", kedav1alpha1.PodIdentityProviderNone:
		body, statusCode, err = s.executeAADApicall(ctx)
	case kedav1alpha1.PodIdentityProviderAzure:
		body, statusCode, err = s.executeIMDSApicall(ctx)
	}

	if err != nil {
		return tokenData{}, fmt.Errorf("error getting access token. HTTP code: %d. Inner Error: %v. Body: %s", statusCode, err, string(body))
	} else if len(body) == 0 {
		return tokenData{}, fmt.Errorf("error getting access token. Details: empty body. HTTP code: %d", statusCode)
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&tokenInfo)
	if err != nil {
		return tokenData{}, fmt.Errorf("error getting access token. Details: can't decode response body to JSON after getting access token. HTTP code: %d. Inner Error: %v. Body: %s", statusCode, err, string(body))
	}

	if statusCode == 200 {
		return tokenInfo, nil
	}

	return tokenData{}, fmt.Errorf("error getting access token. Details: unknown error. HTTP code: %d. Body: %s", statusCode, string(body))
}

func (s *azureLogAnalyticsScaler) executeLogAnalyticsREST(ctx context.Context, query string, tokenInfo tokenData) ([]byte, int, error) {
	m := map[string]interface{}{"query": query}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, 0, fmt.Errorf("can't construct JSON for request to Log Analytics API. Inner Error: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(laQueryEndpoint, s.metadata.logAnalyticsResourceURL, s.metadata.workspaceID), bytes.NewBuffer(jsonBytes)) // URL-encoded payload
	if err != nil {
		return nil, 0, fmt.Errorf("can't construct HTTP request to Log Analytics API. Inner Error: %w", err)
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenInfo.AccessToken))
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

	return s.runHTTP(request, "Log Analytics REST api")
}

func (s *azureLogAnalyticsScaler) executeAADApicall(ctx context.Context) ([]byte, int, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.metadata.clientID},
		"redirect_uri":  {"http://"},
		"resource":      {s.metadata.logAnalyticsResourceURL},
		"client_secret": {s.metadata.clientSecret},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(aadTokenEndpoint, s.metadata.activeDirectoryEndpoint, s.metadata.tenantID), strings.NewReader(data.Encode())) // URL-encoded payload
	if err != nil {
		return nil, 0, fmt.Errorf("can't construct HTTP request to Azure Active Directory. Inner Error: %w", err)
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(data.Encode())))

	return s.runHTTP(request, "AAD")
}

func (s *azureLogAnalyticsScaler) executeIMDSApicall(ctx context.Context) ([]byte, int, error) {
	var urlStr string
	if s.metadata.podIdentity.GetIdentityID() == "" {
		urlStr = fmt.Sprintf(azure.MSIURL, s.metadata.logAnalyticsResourceURL)
	} else {
		urlStr = fmt.Sprintf(azure.MSIURLWithClientID, s.metadata.logAnalyticsResourceURL, url.QueryEscape(s.metadata.podIdentity.GetIdentityID()))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("can't construct HTTP request to Azure Instance Metadata service. Inner Error: %w", err)
	}

	request.Header.Add("Metadata", "true")

	return s.runHTTP(request, "IMDS")
}

func (s *azureLogAnalyticsScaler) runHTTP(request *http.Request, caller string) ([]byte, int, error) {
	request.Header.Add("Cache-Control", "no-cache")
	request.Header.Add("User-Agent", "keda/2.0.0")

	resp, err := s.httpClient.Do(request)
	if err != nil && resp != nil {
		return nil, resp.StatusCode, fmt.Errorf("error calling %s. Inner Error: %w", caller, err)
	} else if err != nil {
		return nil, 0, fmt.Errorf("error calling %s. Inner Error: %w", caller, err)
	}

	defer resp.Body.Close()
	s.httpClient.CloseIdleConnections()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading %s response body: Inner Error: %w", caller, err)
	}

	return body, resp.StatusCode, nil
}

func getTokenFromCache(clientID string, clientSecret string) (tokenData, error) {
	key, err := getHash(clientID, clientSecret)
	if err != nil {
		return tokenData{}, fmt.Errorf("error calculating sha1 hash. Inner Error: %w", err)
	}

	tokenCache.RLock()

	if val, ok := tokenCache.m[key]; ok && val.AccessToken != "" {
		tokenCache.RUnlock()
		return val, nil
	}

	tokenCache.RUnlock()
	return tokenData{}, fmt.Errorf("error getting value from token cache. Details: unknown error")
}

func setTokenInCache(clientID string, clientSecret string, tokenInfo tokenData) error {
	key, err := getHash(clientID, clientSecret)
	if err != nil {
		return err
	}

	tokenCache.Lock()
	tokenCache.m[key] = tokenInfo
	tokenCache.Unlock()

	return nil
}

func getHash(clientID string, clientSecret string) (string, error) {
	sha256Hash := sha256.New()
	_, err := fmt.Fprintf(sha256Hash, "%s|%s", clientID, clientSecret)

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sha256Hash.Sum(nil)), nil
}

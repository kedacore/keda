package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type azureLogAnalyticsScaler struct {
	metadata *azureLogAnalyticsMetadata
	cache    *sessionCache
}

type azureLogAnalyticsMetadata struct {
	tenantID     string
	clientID     string
	clientSecret string
	workspaceID  string
	query        string
	threshold    int64
}

type sessionCache struct {
	accessToken     tokenData
	metricValue     int64
	metricThreshold int64
}

type tokenData struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,string"`
	ExtExpiresIn int    `json:"ext_expires_in,string"`
	ExpiresOn    int64  `json:"expires_on,string"`
	NotBefore    int64  `json:"not_before,string"`
	Resource     string `json:"resource"`
	AccessToken  string `json:"access_token"`
}

type metricsData struct {
	value     int64
	threshold int64
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

var logAnalyticsLog = logf.Log.WithName("azure_log_analytics_scaler")

// NewAzureLogAnalyticsScaler creates a new Azure Log Analytics Scaler
func NewAzureLogAnalyticsScaler(resolvedSecrets, metadata, authParams map[string]string) (Scaler, error) {
	azureLogAnalyticsMetadata, err := parseAzureLogAnalyticsMetadata(resolvedSecrets, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing artemis metadata: %s", err)
	}

	return &azureLogAnalyticsScaler{
		metadata: azureLogAnalyticsMetadata,
		cache:    &sessionCache{metricValue: -1, metricThreshold: -1},
	}, nil
}

func parseAzureLogAnalyticsMetadata(resolvedEnv, metadata, authParams map[string]string) (*azureLogAnalyticsMetadata, error) {

	meta := azureLogAnalyticsMetadata{}

	//Getting tenantId
	if val, ok := authParams["tenantId"]; ok && val != "" {
		meta.tenantID = val
	} else if val, ok := metadata["tenantId"]; ok && val != "" {
		meta.tenantID = val
	} else if val, ok := metadata["tenantIdFromEnv"]; ok && val != "" {
		meta.tenantID = resolvedEnv[metadata["tenantIdFromEnv"]]
	} else {
		return nil, fmt.Errorf("tenantId was not found in metadata. Check your ScaledObject configuration")
	}

	//Getting clientId
	if val, ok := authParams["clientId"]; ok && val != "" {
		meta.clientID = val
	} else if val, ok := metadata["clientId"]; ok && val != "" {
		meta.clientID = val
	} else if val, ok := metadata["clientIdFromEnv"]; ok && val != "" {
		meta.clientID = resolvedEnv[metadata["clientIdFromEnv"]]
	} else {
		return nil, fmt.Errorf("clientId was not found in metadata. Check your ScaledObject configuration")
	}

	//Getting clientSecret
	if val, ok := authParams["clientSecret"]; ok && val != "" {
		meta.clientSecret = val
	} else if val, ok := metadata["clientSecret"]; ok && val != "" {
		meta.clientSecret = val
	} else if val, ok := metadata["clientSecretFromEnv"]; ok && val != "" {
		meta.clientSecret = resolvedEnv[metadata["clientSecretFromEnv"]]
	} else {
		return nil, fmt.Errorf("clientSecret was not found in metadata. Check your ScaledObject configuration")
	}

	//Getting workspaceId
	if val, ok := authParams["workspaceId"]; ok && val != "" {
		meta.workspaceID = val
	} else if val, ok := metadata["workspaceId"]; ok && val != "" {
		meta.workspaceID = val
	} else if val, ok := metadata["workspaceIdFromEnv"]; ok && val != "" {
		meta.workspaceID = resolvedEnv[metadata["workspaceIdFromEnv"]]
	} else {
		return nil, fmt.Errorf("workspaceId was not found in metadata. Check your ScaledObject configuration")
	}

	//Getting query
	if val, ok := metadata["query"]; ok && val != "" {
		meta.query = val
	} else if val, ok := metadata["queryFromEnv"]; ok && val != "" {
		meta.query = resolvedEnv[metadata["queryFromEnv"]]
	} else {
		return nil, fmt.Errorf("query was not found in metadata. Check your ScaledObject configuration")
	}

	//Getting threshold
	if val, ok := metadata["threshold"]; ok && val != "" {
		threshold, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse threshold: %s", err)
		}
		meta.threshold = threshold
	} else if val, ok := metadata["thresholdFromEnv"]; ok && val != "" {
		threshold, err := strconv.ParseInt(resolvedEnv[metadata["thresholdFromEnv"]], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse threshold: %s", err)
		}
		meta.threshold = threshold
	} else {
		return nil, fmt.Errorf("threshold was not found in metadata. Check your ScaledObject configuration")
	}

	return &meta, nil
}

// IsActive determines if we need to scale from zero
func (s *azureLogAnalyticsScaler) IsActive(ctx context.Context) (bool, error) {
	err := s.updateCache()

	if err != nil {
		logAnalyticsLog.Error(err, "Error processing Log Analytics query")
		return false, err
	}

	return s.cache.metricValue > 0, nil
}

func (s *azureLogAnalyticsScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	err := s.updateCache()

	if err != nil {
		logAnalyticsLog.Error(err, "Error processing Log Analytics query")
		return nil
	}

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: fmt.Sprintf("%s-%s", "azure-log-analytics", s.metadata.workspaceID),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: resource.NewQuantity(s.cache.metricThreshold, resource.DecimalSI),
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureLogAnalyticsScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	receivedMetric, err := s.getMetricData()

	if err != nil {
		logAnalyticsLog.Error(err, "Failed to get metrics.")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(receivedMetric.value, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *azureLogAnalyticsScaler) Close() error {
	return nil
}

func (s *azureLogAnalyticsScaler) updateCache() error {
	if s.cache.metricValue < 0 {
		receivedMetric, err := s.getMetricData()

		if err != nil {
			logAnalyticsLog.Error(err, "Error processing Log Analytics query")
			return err
		}

		s.cache.metricValue = receivedMetric.value

		if receivedMetric.threshold > 0 {
			s.cache.metricThreshold = receivedMetric.threshold
		} else {
			s.cache.metricThreshold = s.metadata.threshold
		}
	}

	return nil
}

func (s *azureLogAnalyticsScaler) getMetricData() (metricsData, error) {
	tokenInfo, err := s.getAccessToken()
	if err != nil {
		return metricsData{}, err
	}

	metricsInfo, err := s.executeQuery(s.metadata.query, tokenInfo)
	if err != nil {
		return metricsData{}, err
	}

	logAnalyticsLog.V(1).Info("Getting metrics value", "metrics value", metricsInfo.value)
	return metricsInfo, nil
}

func (s *azureLogAnalyticsScaler) getAccessToken() (tokenData, error) {
	//if there is no token yet or it will be expired in less, that 30 secs
	currentTimeSec := time.Now().Unix()
	if currentTimeSec+30 > s.cache.accessToken.ExpiresOn {
		tokenInfo, err := s.refreshAccessToken()
		if err != nil {
			return tokenData{}, err
		}

		s.cache.accessToken = tokenInfo
		return tokenInfo, nil
	}
	return s.cache.accessToken, nil
}

func (s *azureLogAnalyticsScaler) executeQuery(query string, tokenInfo tokenData) (metricsData, error) {
	queryData := queryResult{}

	body, statusCode, err := s.executeLogAnalyticsREST(query, tokenInfo)

	if body == nil || len(body) == 0 {
		return metricsData{}, fmt.Errorf("Error executing Log Analytics REST API request: empty body")
	}

	//Handle expired token
	if statusCode == 403 || strings.Contains(string(body), "TokenExpired") {
		logAnalyticsLog.Info("Token expired, refreshing token...")

		tokenInfo, err := s.refreshAccessToken()

		if err == nil {
			body, statusCode, err = s.executeLogAnalyticsREST(query, tokenInfo)
		} else {
			return metricsData{}, err
		}
	}

	if err != nil {
		return metricsData{}, err
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&queryData)
	if err != nil {
		if statusCode != 200 {
			return metricsData{}, fmt.Errorf("Error processing Log Analytics REST API request: %d", statusCode)
		}
		return metricsData{}, fmt.Errorf("Can't decode JSON from Log Analytics REST API result: %v", err)
	}

	if statusCode == 200 {

		metricsInfo := metricsData{}
		metricsInfo.threshold = s.metadata.threshold
		metricsInfo.value = 0

		//Pre-validation of query result:
		if len(queryData.Tables) == 0 || len(queryData.Tables[0].Columns) == 0 || len(queryData.Tables[0].Rows) == 0 {
			return metricsData{}, fmt.Errorf("There is no results after running your query")
		} else if len(queryData.Tables) > 1 {
			return metricsData{}, fmt.Errorf("Too many tables in query result: %d. Expected: 1", len(queryData.Tables))
		} else if len(queryData.Tables[0].Rows) > 1 {
			return metricsData{}, fmt.Errorf("Too many rows in query result: %d. Expected: 1", len(queryData.Tables[0].Rows))
		}

		if len(queryData.Tables[0].Rows) == 1 {
			if len(queryData.Tables[0].Rows[0]) > 0 {
				metricDataType := queryData.Tables[0].Columns[0].Type
				metricVal := queryData.Tables[0].Rows[0][0]

				if metricVal != nil {
					//type can be: real, int, long
					if metricDataType == "real" || metricDataType == "int" || metricDataType == "long" {
						metricValue, isConverted := metricVal.(float64)
						if !isConverted {
							return metricsData{}, fmt.Errorf("Cannot convert result to type float64")
						}
						if metricValue < 0 {
							return metricsData{}, fmt.Errorf("Metric value should be >=0. Received value: %f", metricValue)
						}
						metricsInfo.value = int64(metricValue)
					} else {
						return metricsData{}, fmt.Errorf("Invalid data type in query result: \"%s\". Allowed data types: real, int, long", metricDataType)
					}
				}
			}

			if len(queryData.Tables[0].Rows[0]) > 1 {
				thresholdDataType := queryData.Tables[0].Columns[1].Type
				thresholdVal := queryData.Tables[0].Rows[0][1]

				if thresholdVal != nil {
					//type can be: real, int, long
					if thresholdDataType == "real" || thresholdDataType == "int" || thresholdDataType == "long" {
						thresholdValue, isConverted := thresholdVal.(float64)
						if !isConverted {
							return metricsData{}, fmt.Errorf("Cannot convert threshold result to type float64")
						}
						if thresholdValue < 0 {
							return metricsData{}, fmt.Errorf("Threshold value should be >=0. Received value: %f", thresholdValue)
						}
						metricsInfo.threshold = int64(thresholdValue)
					} else {
						return metricsData{}, fmt.Errorf("Invalid data type in query result: \"%s\". Allowed data types: real, int, long", thresholdDataType)
					}
				} else {
					return metricsData{}, fmt.Errorf("Threshold value is empty. Check your query")
				}
			} else {
				metricsInfo.threshold = -1
			}
		}

		return metricsInfo, nil
	}

	return metricsData{}, fmt.Errorf("Error processing request. HTTP code:  %d, details: %s", statusCode, string(body))
}

func (s *azureLogAnalyticsScaler) refreshAccessToken() (tokenData, error) {

	tokenInfo, err := s.getAuthorizationToken()

	if err != nil {
		return tokenData{}, err
	}

	//Now, let's check we can use this token. If no, wait until we can use it
	currentTimeSec := time.Now().Unix()
	if currentTimeSec < tokenInfo.NotBefore {
		if currentTimeSec < tokenInfo.NotBefore+10 {
			sleepDurationSec := int(tokenInfo.NotBefore - currentTimeSec + 1)
			logAnalyticsLog.V(1).Info("AAD token not ready", "delay (seconds)", sleepDurationSec)
			time.Sleep(time.Duration(sleepDurationSec) * time.Second)
		} else {
			return tokenData{}, fmt.Errorf("AAD token has been received, but start date begins in %d seconds. Current operation will be skipped", tokenInfo.NotBefore-currentTimeSec)
		}
	}

	return tokenInfo, nil
}

func (s *azureLogAnalyticsScaler) getAuthorizationToken() (tokenData, error) {
	body, statusCode, err := s.executeAADApicall()
	if err != nil {
		return tokenData{}, fmt.Errorf("Error executing AAD request: %v", err)
	} else if body == nil || len(body) == 0 {
		return tokenData{}, fmt.Errorf("Error executing AAD request: empty body")
	}

	tokenInfo := tokenData{}
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&tokenInfo)
	if err != nil {
		if statusCode != 200 {
			return tokenData{}, fmt.Errorf("Error processing AAD request: %d", statusCode)
		}
		return tokenData{}, fmt.Errorf("Can't decode JSON from AAD result: %v", err)
	}

	if statusCode == 200 {
		return tokenInfo, nil
	}

	return tokenData{}, fmt.Errorf("Error processing AAD request. HTTP code: %d. Details: %s", statusCode, string(body))
}

func (s *azureLogAnalyticsScaler) executeLogAnalyticsREST(query string, tokenInfo tokenData) ([]byte, int, error) {
	m := map[string]interface{}{"query": query}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, 0, fmt.Errorf("Can't construct JSON for request to Log Analytics query API: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.loganalytics.io/v1/workspaces/%s/query", s.metadata.workspaceID), bytes.NewBuffer(jsonBytes)) // URL-encoded payload
	if err != nil {
		return nil, 0, fmt.Errorf("Can't construct HTTP request to Log Analytics query API: %v", err)
	}

	request.Header.Add("Cache-Control", "no-cache")
	request.Header.Add("User-Agent", "keda/2.0.0")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenInfo.AccessToken))
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

	httpClient := &http.Client{}

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("Error calling Log Analytics query API: %v", err)
	}

	defer resp.Body.Close()
	httpClient.CloseIdleConnections()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("Error reading Log Analytics responce body: %v", err)
	}

	return body, resp.StatusCode, nil
}

func (s *azureLogAnalyticsScaler) executeAADApicall() ([]byte, int, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.metadata.clientID},
		"redirect_uri":  {"http://"},
		"resource":      {"https://api.loganalytics.io/"},
		"client_secret": {s.metadata.clientSecret},
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", s.metadata.tenantID), strings.NewReader(data.Encode())) // URL-encoded payload
	if err != nil {
		return nil, 0, fmt.Errorf("Can't construct HTTP request to Azure Active Directory: %v", err)
	}
	request.Header.Add("Cache-Control", "no-cache")
	request.Header.Add("User-Agent", "keda/2.0.0")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(data.Encode())))

	httpClient := &http.Client{}

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("Error calling AAD: %v", err)
	}

	defer resp.Body.Close()
	httpClient.CloseIdleConnections()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("Error reading AAD responce body: %v", err)
	}

	return body, resp.StatusCode, nil
}

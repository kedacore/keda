package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetPipelinesQueueLength = 1
)

type azurePipelinesPoolNameResponse struct {
	Value []struct {
		ID int `json:"id"`
	} `json:"value"`
}

type azurePipelinesPoolIDResponse struct {
	ID int `json:"id"`
}

type azurePipelinesScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *azurePipelinesMetadata
	httpClient *http.Client
}

type azurePipelinesMetadata struct {
	organizationURL            string
	organizationName           string
	personalAccessToken        string
	poolID                     int
	targetPipelinesQueueLength int64
	scalerIndex                int
}

var azurePipelinesLog = logf.Log.WithName("azure_pipelines_scaler")

// NewAzurePipelinesScaler creates a new AzurePipelinesScaler
func NewAzurePipelinesScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAzurePipelinesMetadata(ctx, config, httpClient)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure Pipelines metadata: %s", err)
	}

	return &azurePipelinesScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
	}, nil
}

func parseAzurePipelinesMetadata(ctx context.Context, config *ScalerConfig, httpClient *http.Client) (*azurePipelinesMetadata, error) {
	meta := azurePipelinesMetadata{}
	meta.targetPipelinesQueueLength = defaultTargetPipelinesQueueLength

	if val, ok := config.TriggerMetadata["targetPipelinesQueueLength"]; ok {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing azure pipelines metadata targetPipelinesQueueLength: %s", err.Error())
		}

		meta.targetPipelinesQueueLength = queueLength
	}

	if val, ok := config.AuthParams["organizationURL"]; ok && val != "" {
		// Found the organizationURL in a parameter from TriggerAuthentication
		meta.organizationURL = val
	} else if val, ok := config.TriggerMetadata["organizationURLFromEnv"]; ok && val != "" {
		meta.organizationURL = config.ResolvedEnv[val]
	} else {
		return nil, fmt.Errorf("no organizationURL given")
	}

	if val := meta.organizationURL[strings.LastIndex(meta.organizationURL, "/")+1:]; val != "" {
		meta.organizationName = meta.organizationURL[strings.LastIndex(meta.organizationURL, "/")+1:]
	} else {
		return nil, fmt.Errorf("failed to extract organization name from organizationURL")
	}

	if val, ok := config.AuthParams["personalAccessToken"]; ok && val != "" {
		// Found the personalAccessToken in a parameter from TriggerAuthentication
		meta.personalAccessToken = config.AuthParams["personalAccessToken"]
	} else if val, ok := config.TriggerMetadata["personalAccessTokenFromEnv"]; ok && val != "" {
		meta.personalAccessToken = config.ResolvedEnv[config.TriggerMetadata["personalAccessTokenFromEnv"]]
	} else {
		return nil, fmt.Errorf("no personalAccessToken given")
	}

	if val, ok := config.TriggerMetadata["poolName"]; ok && val != "" {
		var err error
		meta.poolID, err = getPoolIDFromName(ctx, val, &meta, httpClient)
		if err != nil {
			return nil, err
		}
	} else {
		if val, ok := config.TriggerMetadata["poolID"]; ok && val != "" {
			var err error
			meta.poolID, err = validatePoolID(ctx, val, &meta, httpClient)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("no poolName or poolID given")
		}
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func getPoolIDFromName(ctx context.Context, poolName string, metadata *azurePipelinesMetadata, httpClient *http.Client) (int, error) {
	url := fmt.Sprintf("%s/_apis/distributedtask/pools?poolName=%s", metadata.organizationURL, poolName)
	body, err := getAzurePipelineRequest(ctx, url, metadata, httpClient)
	if err != nil {
		return -1, err
	}

	var result azurePipelinesPoolNameResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return -1, err
	}

	count := len(result.Value)
	if count == 0 {
		return -1, fmt.Errorf("agent pool with name `%s` not found", poolName)
	}

	if count != 1 {
		return -1, fmt.Errorf("found %d agent pool with name `%s`", count, poolName)
	}

	return result.Value[0].ID, nil
}

func validatePoolID(ctx context.Context, poolID string, metadata *azurePipelinesMetadata, httpClient *http.Client) (int, error) {
	url := fmt.Sprintf("%s/_apis/distributedtask/pools?poolID=%s", metadata.organizationURL, poolID)
	body, err := getAzurePipelineRequest(ctx, url, metadata, httpClient)
	if err != nil {
		return -1, fmt.Errorf("agent pool with id `%s` not found", poolID)
	}

	var result azurePipelinesPoolIDResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return -1, err
	}

	return result.ID, nil
}

func getAzurePipelineRequest(ctx context.Context, url string, metadata *azurePipelinesMetadata, httpClient *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.SetBasicAuth("PAT", metadata.personalAccessToken)

	r, err := httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return []byte{}, err
	}
	r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		return []byte{}, fmt.Errorf("the Azure DevOps REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	return b, nil
}

func (s *azurePipelinesScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.GetAzurePipelinesQueueLength(ctx)

	if err != nil {
		azurePipelinesLog.Error(err, "error getting pipelines queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(queuelen, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *azurePipelinesScaler) GetAzurePipelinesQueueLength(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests", s.metadata.organizationURL, s.metadata.poolID)
	body, err := getAzurePipelineRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return -1, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return -1, err
	}

	var count int64
	jobs, ok := result["value"].([]interface{})

	if !ok {
		return -1, fmt.Errorf("the Azure DevOps REST API result returned no value data despite successful code. url: %s", url)
	}

	for _, value := range jobs {
		v := value.(map[string]interface{})
		if v["result"] == nil {
			count++
		}
	}

	return count, err
}

func (s *azurePipelinesScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-pipelines-%d", s.metadata.poolID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetPipelinesQueueLength),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s *azurePipelinesScaler) IsActive(ctx context.Context) (bool, error) {
	queuelen, err := s.GetAzurePipelinesQueueLength(ctx)

	if err != nil {
		azurePipelinesLog.Error(err, "error)")
		return false, err
	}

	return queuelen > 0, nil
}

func (s *azurePipelinesScaler) Close(context.Context) error {
	return nil
}

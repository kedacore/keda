package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultTargetPipelinesQueueLength = 1
)

type azurePipelinesPoolResponse struct {
	Value []struct {
		ID int `json:"id"`
	} `json:"value"`
}

type azurePipelinesScaler struct {
	metadata   *azurePipelinesMetadata
	httpClient *http.Client
}

type azurePipelinesMetadata struct {
	organizationURL            string
	organizationName           string
	personalAccessToken        string
	poolID                     int
	targetPipelinesQueueLength int
	scalerIndex                int
}

var azurePipelinesLog = logf.Log.WithName("azure_pipelines_scaler")

// NewAzurePipelinesScaler creates a new AzurePipelinesScaler
func NewAzurePipelinesScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	meta, err := parseAzurePipelinesMetadata(config, httpClient, ctx)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure Pipelines metadata: %s", err)
	}

	return &azurePipelinesScaler{
		metadata:   meta,
		httpClient: httpClient,
	}, nil
}

func parseAzurePipelinesMetadata(config *ScalerConfig, httpClient *http.Client, ctx context.Context) (*azurePipelinesMetadata, error) {
	meta := azurePipelinesMetadata{}
	meta.targetPipelinesQueueLength = defaultTargetPipelinesQueueLength

	if val, ok := config.TriggerMetadata["targetPipelinesQueueLength"]; ok {
		queueLength, err := strconv.Atoi(val)
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
		meta.poolID, err = getAzurePipelinesPoolID(val, meta, httpClient, ctx)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no poolName given")
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}
func getAzurePipelinesPoolID(poolName string, metadata azurePipelinesMetadata, httpClient *http.Client, ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/_apis/distributedtask/pools?poolName=%s", metadata.organizationURL, poolName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}

	req.SetBasicAuth("PAT", metadata.personalAccessToken)

	r, err := httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		return -1, fmt.Errorf("the Azure DevOps REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	var result azurePipelinesPoolResponse
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	count := len(result.Value)
	if count != 1 {
		return -1, fmt.Errorf("incorrect agent pool count, expected 1 and got %d", count)
	}

	return result.Value[0].ID, nil
}

func (s *azurePipelinesScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.GetAzurePipelinesQueueLength(ctx)

	if err != nil {
		azurePipelinesLog.Error(err, "error getting pipelines queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *azurePipelinesScaler) GetAzurePipelinesQueueLength(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests", s.metadata.organizationURL, s.metadata.poolID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}

	req.SetBasicAuth("PAT", s.metadata.personalAccessToken)

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		return -1, fmt.Errorf("the Azure DevOps REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	var result map[string]interface{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var count = 0
	jobs, ok := result["value"].([]interface{})

	if !ok {
		return -1, fmt.Errorf("the Azure DevOps REST API result returned no value data. url: %s status: %d", url, r.StatusCode)
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
	targetPipelinesQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetPipelinesQueueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-pipelines-%s", s.metadata.poolID))),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetPipelinesQueueLengthQty,
		},
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

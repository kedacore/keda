package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

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

type azurePipelinesScaler struct {
	metadata   *azurePipelinesMetadata
	httpClient *http.Client
}

type azurePipelinesMetadata struct {
	organizationURL            string
	personalAccessToken        string
	poolID                     string
	targetPipelinesQueueLength int
}

var azurePipelinesLog = logf.Log.WithName("azure_pipelines_scaler")

// NewAzurePipelinesScaler creates a new AzurePipelinesScaler
func NewAzurePipelinesScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseAzurePipelinesMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure Pipelines metadata: %s", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	return &azurePipelinesScaler{
		metadata:   meta,
		httpClient: httpClient,
	}, nil
}

func parseAzurePipelinesMetadata(config *ScalerConfig) (*azurePipelinesMetadata, error) {
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

	if val, ok := config.AuthParams["personalAccessToken"]; ok && val != "" {
		// Found the personalAccessToken in a parameter from TriggerAuthentication
		meta.personalAccessToken = config.AuthParams["personalAccessToken"]
	} else if val, ok := config.TriggerMetadata["personalAccessTokenFromEnv"]; ok && val != "" {
		meta.personalAccessToken = config.ResolvedEnv[config.TriggerMetadata["personalAccessTokenFromEnv"]]
	} else {
		return nil, fmt.Errorf("no personalAccessToken given")
	}

	if val, ok := config.TriggerMetadata["poolID"]; ok && val != "" {
		meta.poolID = val
	} else {
		return nil, fmt.Errorf("no poolID given")
	}

	return &meta, nil
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
	url := fmt.Sprintf("%s/_apis/distributedtask/pools/%s/jobrequests", s.metadata.organizationURL, s.metadata.poolID)
	req, err := http.NewRequest("GET", url, nil)
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
		return -1, fmt.Errorf("azure Devops REST api returned error. status: %d response: %s", r.StatusCode, string(b))
	}

	var result map[string]interface{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var count int = 0
	jobs, ok := result["value"].([]interface{})

	if !ok {
		return -1, fmt.Errorf("api result returned no value data")
	}

	for _, value := range jobs {
		v := value.(map[string]interface{})
		if v["result"] == nil {
			count++
		}
	}

	return count, err
}

func (s *azurePipelinesScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetPipelinesQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetPipelinesQueueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-pipelines-queue", s.metadata.poolID)),
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

func (s *azurePipelinesScaler) Close() error {
	return nil
}

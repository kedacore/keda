package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

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
	logger     logr.Logger
}

type azurePipelinesMetadata struct {
	organizationURL                      string
	organizationName                     string
	personalAccessToken                  string
	parent                               string
	demands                              string
	poolID                               int
	targetPipelinesQueueLength           int64
	activationTargetPipelinesQueueLength int64
	scalerIndex                          int
}

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
		logger:     InitializeLogger(config, "azure_pipelines_scaler"),
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

	meta.activationTargetPipelinesQueueLength = 0
	if val, ok := config.TriggerMetadata["activationTargetPipelinesQueueLength"]; ok {
		activationQueueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing azure pipelines metadata activationTargetPipelinesQueueLength: %s", err.Error())
		}

		meta.activationTargetPipelinesQueueLength = activationQueueLength
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

	if val, ok := config.TriggerMetadata["parent"]; ok && val != "" {
		meta.parent = config.TriggerMetadata["parent"]
	} else {
		meta.parent = ""
	}

	if val, ok := config.TriggerMetadata["demands"]; ok && val != "" {
		meta.demands = config.TriggerMetadata["demands"]
	} else {
		meta.demands = ""
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

	// Trim any trailing new lines from the Azure Pipelines PAT
	meta.personalAccessToken = strings.TrimSuffix(meta.personalAccessToken, "\n")
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

	req.SetBasicAuth("", metadata.personalAccessToken)

	r, err := httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	b, err := io.ReadAll(r.Body)
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
		s.logger.Error(err, "error getting pipelines queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

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

	// for each job check if it parent fulfilled, then demand fulfilled, then finally pool fulfilled
	for _, value := range jobs {
		v := value.(map[string]interface{})
		if v["result"] == nil {
			if s.metadata.parent == "" && s.metadata.demands == "" {
				// no plan defined, just add a count
				count++
			} else {
				if s.metadata.parent == "" {
					// doesn't use parent, switch to demand
					if getCanAgentDemandFulfilJob(v, s.metadata) {
						count++
					}
				} else {
					// does use parent
					if getCanAgentParentFulfilJob(v, s.metadata) {
						count++
					}
				}
			}
		}
	}
	return count, err
}

// Determine if the scaledjob has the right demands to spin up
func getCanAgentDemandFulfilJob(v map[string]interface{}, metadata *azurePipelinesMetadata) bool {
	var demandsReq = v["demands"].([]interface{})
	var demandsAvail = strings.Split(metadata.demands, ",")
	var countDemands = 0
	for _, dr := range demandsReq {
		for _, da := range demandsAvail {
			strDr := fmt.Sprintf("%v", dr)
			if !strings.HasPrefix(strDr, "Agent.Version") {
				if strDr == da {
					countDemands++
				}
			}
		}
	}

	return countDemands == len(demandsReq)-1
}

// Determine if the Job and Parent Agent Template have matching capabilities
func getCanAgentParentFulfilJob(v map[string]interface{}, metadata *azurePipelinesMetadata) bool {
	matchedAgents, ok := v["matchedAgents"].([]interface{})
	if !ok {
		// ADO is already processing
		return false
	}

	for _, m := range matchedAgents {
		n := m.(map[string]interface{})
		if metadata.parent == n["name"].(string) {
			return true
		}
	}
	return false
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
		s.logger.Error(err, "error)")
		return false, err
	}

	return queuelen > s.metadata.activationTargetPipelinesQueueLength, nil
}

func (s *azurePipelinesScaler) Close(context.Context) error {
	return nil
}

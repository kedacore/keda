package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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
	defaultTargetPipelinesQueueLength = 1
)

type JobRequests struct {
	Count int          `json:"count"`
	Value []JobRequest `json:"value"`
}

const (
	// "499b84ac-1321-427f-aa17-267ca6975798" is the azure id for DevOps resource
	// https://learn.microsoft.com/en-gb/azure/devops/integrate/get-started/authentication/service-principal-managed-identity?view=azure-devops
	devopsResource = "499b84ac-1321-427f-aa17-267ca6975798/.default"
)

type JobRequest struct {
	RequestID     int       `json:"requestId"`
	QueueTime     time.Time `json:"queueTime"`
	AssignTime    time.Time `json:"assignTime,omitempty"`
	ReceiveTime   time.Time `json:"receiveTime,omitempty"`
	LockedUntil   time.Time `json:"lockedUntil,omitempty"`
	ServiceOwner  string    `json:"serviceOwner"`
	HostID        string    `json:"hostId"`
	Result        *string   `json:"result"`
	ScopeID       string    `json:"scopeId"`
	PlanType      string    `json:"planType"`
	PlanID        string    `json:"planId"`
	JobID         string    `json:"jobId"`
	Demands       []string  `json:"demands"`
	ReservedAgent *struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			Web struct {
				Href string `json:"href"`
			} `json:"web"`
		} `json:"_links"`
		ID                int    `json:"id"`
		Name              string `json:"name"`
		Version           string `json:"version"`
		OsDescription     string `json:"osDescription"`
		Enabled           bool   `json:"enabled"`
		Status            string `json:"status"`
		ProvisioningState string `json:"provisioningState"`
		AccessPoint       string `json:"accessPoint"`
	} `json:"reservedAgent,omitempty"`
	Definition struct {
		Links struct {
			Web struct {
				Href string `json:"href"`
			} `json:"web"`
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"definition"`
	Owner struct {
		Links struct {
			Web struct {
				Href string `json:"href"`
			} `json:"web"`
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"owner"`
	Data struct {
		ParallelismTag string `json:"ParallelismTag"`
		IsScheduledKey string `json:"IsScheduledKey"`
	} `json:"data"`
	PoolID          int    `json:"poolId"`
	OrchestrationID string `json:"orchestrationId"`
	Priority        int    `json:"priority"`
	MatchedAgents   *[]struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			Web struct {
				Href string `json:"href"`
			} `json:"web"`
		} `json:"_links"`
		ID                int    `json:"id"`
		Name              string `json:"name"`
		Version           string `json:"version"`
		Enabled           bool   `json:"enabled"`
		Status            string `json:"status"`
		ProvisioningState string `json:"provisioningState"`
	} `json:"matchedAgents,omitempty"`
}

type azurePipelinesPoolNameResponse struct {
	Value []struct {
		ID int `json:"id"`
	} `json:"value"`
}

type azurePipelinesPoolIDResponse struct {
	ID int `json:"id"`
}

type azurePipelinesScaler struct {
	metricType  v2.MetricTargetType
	metadata    *azurePipelinesMetadata
	httpClient  *http.Client
	podIdentity kedav1alpha1.AuthPodIdentity
	logger      logr.Logger
}

type azurePipelinesMetadata struct {
	organizationURL                      string
	organizationName                     string
	authContext                          authContext
	parent                               string
	demands                              string
	poolID                               int
	targetPipelinesQueueLength           int64
	activationTargetPipelinesQueueLength int64
	jobsToFetch                          int64
	triggerIndex                         int
	requireAllDemands                    bool
}

type authContext struct {
	cred  *azidentity.ChainedTokenCredential
	pat   string
	token *azcore.AccessToken
}

// NewAzurePipelinesScaler creates a new AzurePipelinesScaler
func NewAzurePipelinesScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	logger := InitializeLogger(config, "azure_pipelines_scaler")
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, podIdentity, err := parseAzurePipelinesMetadata(ctx, logger, config, httpClient)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure Pipelines metadata: %w", err)
	}

	return &azurePipelinesScaler{
		metricType:  metricType,
		metadata:    meta,
		httpClient:  httpClient,
		podIdentity: podIdentity,
		logger:      logger,
	}, nil
}

func getAuthMethod(logger logr.Logger, config *scalersconfig.ScalerConfig) (string, *azidentity.ChainedTokenCredential, kedav1alpha1.AuthPodIdentity, error) {
	pat := ""
	if val, ok := config.AuthParams["personalAccessToken"]; ok && val != "" {
		// Found the personalAccessToken in a parameter from TriggerAuthentication
		pat = config.AuthParams["personalAccessToken"]
	} else if val, ok := config.TriggerMetadata["personalAccessTokenFromEnv"]; ok && val != "" {
		pat = config.ResolvedEnv[config.TriggerMetadata["personalAccessTokenFromEnv"]]
	} else {
		switch config.PodIdentity.Provider {
		case "", kedav1alpha1.PodIdentityProviderNone:
			return "", nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no personalAccessToken given or PodIdentity provider configured")
		case kedav1alpha1.PodIdentityProviderAzureWorkload:
			cred, err := azure.NewChainedCredential(logger, config.PodIdentity.GetIdentityID(), config.PodIdentity.GetIdentityTenantID(), config.PodIdentity.Provider)
			if err != nil {
				return "", nil, kedav1alpha1.AuthPodIdentity{}, err
			}
			return "", cred, kedav1alpha1.AuthPodIdentity{Provider: config.PodIdentity.Provider}, nil
		default:
			return "", nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("pod identity %s not supported for azure pipelines", config.PodIdentity.Provider)
		}
	}
	return pat, nil, kedav1alpha1.AuthPodIdentity{}, nil
}

func parseAzurePipelinesMetadata(ctx context.Context, logger logr.Logger, config *scalersconfig.ScalerConfig, httpClient *http.Client) (*azurePipelinesMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := azurePipelinesMetadata{}
	meta.targetPipelinesQueueLength = defaultTargetPipelinesQueueLength

	if val, ok := config.TriggerMetadata["targetPipelinesQueueLength"]; ok {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure pipelines metadata targetPipelinesQueueLength: %w", err)
		}

		meta.targetPipelinesQueueLength = queueLength
	}

	meta.activationTargetPipelinesQueueLength = 0
	if val, ok := config.TriggerMetadata["activationTargetPipelinesQueueLength"]; ok {
		activationQueueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure pipelines metadata activationTargetPipelinesQueueLength: %w", err)
		}

		meta.activationTargetPipelinesQueueLength = activationQueueLength
	}

	if val, ok := config.AuthParams["organizationURL"]; ok && val != "" {
		// Found the organizationURL in a parameter from TriggerAuthentication
		meta.organizationURL = val
	} else if val, ok := config.TriggerMetadata["organizationURLFromEnv"]; ok && val != "" {
		meta.organizationURL = config.ResolvedEnv[val]
	} else {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no organizationURL given")
	}

	if val := meta.organizationURL[strings.LastIndex(meta.organizationURL, "/")+1:]; val != "" {
		meta.organizationName = meta.organizationURL[strings.LastIndex(meta.organizationURL, "/")+1:]
	} else {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("failed to extract organization name from organizationURL")
	}

	pat, cred, podIdentity, err := getAuthMethod(logger, config)
	if err != nil {
		return nil, kedav1alpha1.AuthPodIdentity{}, err
	}
	// Trim any trailing new lines from the Azure Pipelines PAT
	meta.authContext = authContext{
		pat:   strings.TrimSuffix(pat, "\n"),
		cred:  cred,
		token: nil,
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

	meta.jobsToFetch = 250
	if val, ok := config.TriggerMetadata["jobsToFetch"]; ok && val != "" {
		jobsToFetch, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing jobsToFetch: %w", err)
		}
		meta.jobsToFetch = jobsToFetch
	}

	meta.requireAllDemands = false
	if val, ok := config.TriggerMetadata["requireAllDemands"]; ok && val != "" {
		requireAllDemands, err := strconv.ParseBool(val)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, err
		}
		meta.requireAllDemands = requireAllDemands
	}

	if val, ok := config.TriggerMetadata["poolName"]; ok && val != "" {
		var err error
		poolID, err := getPoolIDFromName(ctx, logger, val, &meta, podIdentity, httpClient)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, err
		}
		meta.poolID = poolID
	} else {
		if val, ok := config.TriggerMetadata["poolID"]; ok && val != "" {
			var err error
			poolID, err := validatePoolID(ctx, logger, val, &meta, podIdentity, httpClient)
			if err != nil {
				return nil, kedav1alpha1.AuthPodIdentity{}, err
			}
			meta.poolID = poolID
		} else {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no poolName or poolID given")
		}
	}

	meta.triggerIndex = config.TriggerIndex

	return &meta, podIdentity, nil
}

func getPoolIDFromName(ctx context.Context, logger logr.Logger, poolName string, metadata *azurePipelinesMetadata, podIdentity kedav1alpha1.AuthPodIdentity, httpClient *http.Client) (int, error) {
	urlString := fmt.Sprintf("%s/_apis/distributedtask/pools?poolName=%s", metadata.organizationURL, url.QueryEscape(poolName))
	body, err := getAzurePipelineRequest(ctx, logger, urlString, metadata, podIdentity, httpClient)

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

func validatePoolID(ctx context.Context, logger logr.Logger, poolID string, metadata *azurePipelinesMetadata, podIdentity kedav1alpha1.AuthPodIdentity, httpClient *http.Client) (int, error) {
	urlString := fmt.Sprintf("%s/_apis/distributedtask/pools?poolID=%s", metadata.organizationURL, poolID)
	body, err := getAzurePipelineRequest(ctx, logger, urlString, metadata, podIdentity, httpClient)

	if err != nil {
		return -1, fmt.Errorf("agent pool with id `%s` not found: %w", poolID, err)
	}

	var result azurePipelinesPoolIDResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return -1, err
	}

	return result.ID, nil
}

func getToken(ctx context.Context, metadata *azurePipelinesMetadata, scope string) (string, error) {
	if metadata.authContext.token != nil {
		//if token expires after more then minute from now let's reuse
		if metadata.authContext.token.ExpiresOn.After(time.Now().Add(time.Second * 60)) {
			return metadata.authContext.token.Token, nil
		}
	}
	token, err := metadata.authContext.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{
			scope,
		},
	})

	if err != nil {
		return "", err
	}

	metadata.authContext.token = &token

	return metadata.authContext.token.Token, nil
}

func getAzurePipelineRequest(ctx context.Context, logger logr.Logger, urlString string, metadata *azurePipelinesMetadata, podIdentity kedav1alpha1.AuthPodIdentity, httpClient *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlString, nil)
	if err != nil {
		return []byte{}, err
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		//PAT
		logger.V(1).Info("making request to ADO REST API using PAT")
		req.SetBasicAuth("", metadata.authContext.pat)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		//ADO Resource token
		logger.V(1).Info("making request to ADO REST API using managed identity")
		aadToken, err := getToken(ctx, metadata, devopsResource)
		if err != nil {
			return []byte{}, fmt.Errorf("cannot create workload identity credentials: %w", err)
		}
		logger.V(1).Info("token acquired setting auth header as 'bearer XXXXXX'")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", aadToken))
	}

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
		return []byte{}, fmt.Errorf("the Azure DevOps REST API returned error. urlString: %s status: %d response: %s", urlString, r.StatusCode, string(b))
	}

	return b, nil
}

func (s *azurePipelinesScaler) GetAzurePipelinesQueueLength(ctx context.Context) (int64, error) {
	// HotFix Issue (#4387), $top changes the format of the returned JSON
	var urlString string
	if s.metadata.parent != "" {
		urlString = fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests", s.metadata.organizationURL, s.metadata.poolID)
	} else {
		urlString = fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests?$top=%d", s.metadata.organizationURL, s.metadata.poolID, s.metadata.jobsToFetch)
	}
	body, err := getAzurePipelineRequest(ctx, s.logger, urlString, s.metadata, s.podIdentity, s.httpClient)
	if err != nil {
		return -1, err
	}

	var jrs JobRequests
	err = json.Unmarshal(body, &jrs)
	if err != nil {
		s.logger.Error(err, "Cannot unmarshal ADO JobRequests API response")
		return -1, err
	}

	// for each job check if its parent fulfilled, then demand fulfilled, then finally pool fulfilled
	var count int64
	for _, job := range stripDeadJobs(jrs.Value) {
		if s.metadata.parent == "" && s.metadata.demands == "" {
			// no plan defined, just add a count
			count++
		} else {
			if s.metadata.parent == "" {
				// doesn't use parent, switch to demand
				if getCanAgentDemandFulfilJob(job, s.metadata) {
					count++
				}
			} else {
				// does use parent
				if getCanAgentParentFulfilJob(job, s.metadata) {
					count++
				}
			}
		}
	}

	return count, err
}

func stripDeadJobs(jobs []JobRequest) []JobRequest {
	var filtered []JobRequest
	for _, job := range jobs {
		if job.Result == nil {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func stripAgentVFromArray(array []string) []string {
	var result []string

	for _, item := range array {
		if !strings.HasPrefix(item, "Agent.Version") {
			result = append(result, item)
		}
	}
	return result
}

// Determine if the scaledjob has the right demands to spin up
func getCanAgentDemandFulfilJob(jr JobRequest, metadata *azurePipelinesMetadata) bool {
	countDemands := 0
	demandsInJob := stripAgentVFromArray(jr.Demands)
	demandsInScaler := stripAgentVFromArray(strings.Split(metadata.demands, ","))

	for _, demandInJob := range demandsInJob {
		for _, demandInScaler := range demandsInScaler {
			if demandInJob == demandInScaler {
				countDemands++
			}
		}
	}

	if metadata.requireAllDemands {
		return countDemands == len(demandsInJob) && countDemands == len(demandsInScaler)
	}
	return countDemands == len(demandsInJob)
}

// Determine if the Job and Parent Agent Template have matching capabilities
func getCanAgentParentFulfilJob(jr JobRequest, metadata *azurePipelinesMetadata) bool {
	matchedAgents := jr.MatchedAgents

	if matchedAgents == nil {
		return false
	}

	for _, m := range *matchedAgents {
		if metadata.parent == m.Name {
			return true
		}
	}
	return false
}

func (s *azurePipelinesScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-pipelines-%d", s.metadata.poolID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetPipelinesQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *azurePipelinesScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueLen, err := s.GetAzurePipelinesQueueLength(ctx)

	if err != nil {
		s.logger.Error(err, "error getting pipelines queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen > s.metadata.activationTargetPipelinesQueueLength, nil
}

func (s *azurePipelinesScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

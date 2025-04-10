package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	OrganizationURL                      string `keda:"name=organizationURL,          order=resolvedEnv;authParams"`
	OrganizationName                     string
	authContext                          authContext
	Parent                               string `keda:"name=parent,          order=triggerMetadata, optional"`
	Demands                              string `keda:"name=demands,          order=triggerMetadata, optional"`
	PoolName                             string `keda:"name=poolName,          order=triggerMetadata, optional"`
	PoolID                               int    `keda:"name=poolID,          order=triggerMetadata, optional"`
	TargetPipelinesQueueLength           int64  `keda:"name=targetPipelinesQueueLength,          order=triggerMetadata, default=1, optional"`
	ActivationTargetPipelinesQueueLength int64  `keda:"name=activationTargetPipelinesQueueLength,          order=triggerMetadata, default=0, optional"`
	JobsToFetch                          int64  `keda:"name=jobsToFetch,          order=triggerMetadata, default=250, optional"`
	triggerIndex                         int
	RequireAllDemands                    bool `keda:"name=requireAllDemands,          order=triggerMetadata, default=false, optional"`
	RequireAllDemandsAndIgnoreOthers     bool `keda:"name=requireAllDemandsAndIgnoreOthers,          order=triggerMetadata, default=false, optional"`
}

type authContext struct {
	cred  *azidentity.ChainedTokenCredential
	pat   string
	token *azcore.AccessToken
}

func (a *azurePipelinesMetadata) Validate() error {
	if val := a.OrganizationURL[strings.LastIndex(a.OrganizationURL, "/")+1:]; val != "" {
		a.OrganizationName = a.OrganizationURL[strings.LastIndex(a.OrganizationURL, "/")+1:]
	} else {
		return fmt.Errorf("failed to extract organization name from organizationURL")
	}
	return nil
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
			cred, err := azure.NewChainedCredential(logger, config.PodIdentity)
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
	meta := &azurePipelinesMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure pipeline metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex

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

	if meta.PoolName != "" {
		var err error
		poolID, err := getPoolIDFromName(ctx, logger, meta.PoolName, meta, podIdentity, httpClient)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, err
		}
		meta.PoolID = poolID
	} else if meta.PoolID != 0 {
		var err error
		_, err = validatePoolID(ctx, logger, meta.PoolID, meta, podIdentity, httpClient)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, err
		}
	} else {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no poolName or poolID given")
	}

	meta.triggerIndex = config.TriggerIndex

	return meta, podIdentity, nil
}

func getPoolIDFromName(ctx context.Context, logger logr.Logger, poolName string, metadata *azurePipelinesMetadata, podIdentity kedav1alpha1.AuthPodIdentity, httpClient *http.Client) (int, error) {
	urlString := fmt.Sprintf("%s/_apis/distributedtask/pools?poolName=%s", metadata.OrganizationURL, url.QueryEscape(poolName))
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

func validatePoolID(ctx context.Context, logger logr.Logger, poolID int, metadata *azurePipelinesMetadata, podIdentity kedav1alpha1.AuthPodIdentity, httpClient *http.Client) (int, error) {
	urlString := fmt.Sprintf("%s/_apis/distributedtask/pools?poolID=%d", metadata.OrganizationURL, poolID)
	body, err := getAzurePipelineRequest(ctx, logger, urlString, metadata, podIdentity, httpClient)

	if err != nil {
		return -1, fmt.Errorf("agent pool with id `%d` not found: %w", poolID, err)
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

	// Log when API Rate Limits are reached
	rateLimitRemaining := r.Header[http.CanonicalHeaderKey("X-RateLimit-Remaining")]
	if rateLimitRemaining != nil {
		logger.V(1).Info(fmt.Sprintf("Warning: ADO TSTUs Left %s. When reaching zero requests are delayed, lower the polling interval. See https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits?view=azure-devops", rateLimitRemaining))
	}
	rateLimitDelay := r.Header[http.CanonicalHeaderKey("X-RateLimit-Delay")]
	if rateLimitDelay != nil {
		logger.V(1).Info(fmt.Sprintf("Warning: Request to ADO API is delayed by %s seconds. Sending additional requests will increase delay until results are being blocked entirely", rateLimitDelay))
	}

	return b, nil
}

func (s *azurePipelinesScaler) GetAzurePipelinesQueueLength(ctx context.Context) (int64, error) {
	// HotFix Issue (#4387), $top changes the format of the returned JSON
	var urlString string
	if s.metadata.Parent != "" {
		urlString = fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests", s.metadata.OrganizationURL, s.metadata.PoolID)
	} else {
		urlString = fmt.Sprintf("%s/_apis/distributedtask/pools/%d/jobrequests?$top=%d", s.metadata.OrganizationURL, s.metadata.PoolID, s.metadata.JobsToFetch)
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
		if s.metadata.Parent == "" && s.metadata.Demands == "" {
			// no plan defined, just add a count
			count++
		} else {
			if s.metadata.Parent == "" {
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
	demandsInScaler := stripAgentVFromArray(strings.Split(metadata.Demands, ","))

	for _, demandInJob := range demandsInJob {
		for _, demandInScaler := range demandsInScaler {
			if demandInJob == demandInScaler {
				countDemands++
			}
		}
	}

	if metadata.RequireAllDemands {
		return countDemands == len(demandsInJob) && countDemands == len(demandsInScaler)
	} else if metadata.RequireAllDemandsAndIgnoreOthers {
		return countDemands == len(demandsInScaler)
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
		if metadata.Parent == m.Name {
			return true
		}
	}
	return false
}

func (s *azurePipelinesScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-pipelines-%d", s.metadata.PoolID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetPipelinesQueueLength),
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

	return []external_metrics.ExternalMetricValue{metric}, queueLen > s.metadata.ActivationTargetPipelinesQueueLength, nil
}

func (s *azurePipelinesScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

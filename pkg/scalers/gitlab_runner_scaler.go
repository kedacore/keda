package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// externalMetricType is the type of the external metric.
	defaultTargetPipelineQueueLength = 1
	// defaultGitlabAPIURL is the default GitLab API base URL.
	defaultGitlabAPIURL = "https://gitlab.com"

	// pipelineWaitingForResourceStatus is the status of the pipelines that are waiting for resources.
	pipelineWaitingForResourceStatus = "waiting_for_resource"

	// maxGitlabAPIPageCount is the maximum number of pages to query for pipelines.
	maxGitlabAPIPageCount = 50
	// gitlabAPIPerPage is the number of pipelines to query per page.
	gitlabAPIPerPage = "200"
)

type gitlabRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *gitlabRunnerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type gitlabRunnerMetadata struct {
	gitlabAPIURL        *url.URL
	personalAccessToken string
	projectID           string

	targetPipelineQueueLength int64
	triggerIndex              int
}

// NewGitLabRunnerScaler creates a new GitLab Runner Scaler
func NewGitLabRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGitLabRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing GitLab Runner metadata: %w", err)
	}

	return &gitlabRunnerScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "gitlab_runner_scaler"),
	}, nil
}

func parseGitLabRunnerMetadata(config *scalersconfig.ScalerConfig) (*gitlabRunnerMetadata, error) {
	meta := &gitlabRunnerMetadata{}
	meta.targetPipelineQueueLength = defaultTargetWorkflowQueueLength

	// Get the projectID
	projectIDValue, err := getValueFromMetaOrEnv("projectID", config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.projectID = projectIDValue

	// Get the targetWorkflowQueueLength
	targetWorkflowQueueLength, err := getInt64ValueFromMetaOrEnv("targetWorkflowQueueLength", config)
	if err != nil || targetWorkflowQueueLength == 0 {
		meta.targetPipelineQueueLength = defaultTargetPipelineQueueLength
	}
	meta.targetPipelineQueueLength = targetWorkflowQueueLength

	// Get the personalAccessToken
	personalAccessToken, ok := config.AuthParams["personalAccessToken"]
	if !ok || personalAccessToken == "" {
		return nil, errors.New("no personalAccessToken provided")
	}

	meta.personalAccessToken = personalAccessToken

	// Get the GitLab API URL
	gitlabAPIURLValue, err := getValueFromMetaOrEnv("gitlabAPIURL", config.TriggerMetadata, config.ResolvedEnv)
	if err != nil || gitlabAPIURLValue == "" {
		gitlabAPIURLValue = defaultGitlabAPIURL
	}

	gitlabAPIURL, err := url.Parse(gitlabAPIURLValue)
	if err != nil {
		return nil, fmt.Errorf("parsing gitlabAPIURL: %w", err)
	}

	// Construct the GitLab API URL
	uri := constructGitlabAPIPipelinesURL(*gitlabAPIURL, projectIDValue, pipelineWaitingForResourceStatus)

	meta.gitlabAPIURL = &uri

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func (s *gitlabRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueLen, err := s.getPipelineQueueLength(ctx)

	if err != nil {
		s.logger.Error(err, "error getting workflow queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen >= s.metadata.targetPipelineQueueLength, nil
}

func (s *gitlabRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("gitlab-runner-%s", s.metadata.projectID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetPipelineQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *gitlabRunnerScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
func constructGitlabAPIPipelinesURL(baseURL url.URL, projectID string, status string) url.URL {
	baseURL.Path = "/api/v4/projects/" + projectID + "/pipelines"

	qParams := baseURL.Query()
	qParams.Set("status", status)
	qParams.Set("per_page", gitlabAPIPerPage)

	baseURL.RawQuery = qParams.Encode()

	return baseURL
}

// getPipelineCount returns the number of pipelines in the GitLab project (per the page set in url)
func (s *gitlabRunnerScaler) getPipelineCount(ctx context.Context, uri string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", s.metadata.personalAccessToken)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("doing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	gitlabPipelines := make([]struct{}, 0)
	if err := json.NewDecoder(res.Body).Decode(&gitlabPipelines); err != nil {
		return 0, fmt.Errorf("decoding response: %w", err)
	}

	return int64(len(gitlabPipelines)), nil
}

// getPipelineQueueLength returns the number of pipelines in the
// GitLab project that are waiting for resources.
func (s *gitlabRunnerScaler) getPipelineQueueLength(ctx context.Context) (int64, error) {
	var count int64

	page := 1
	for ; page < maxGitlabAPIPageCount; page++ {
		pagedURL := pagedURL(*s.metadata.gitlabAPIURL, fmt.Sprint(page))

		gitlabPipelinesLen, err := s.getPipelineCount(ctx, pagedURL.String())
		if err != nil {
			return 0, err
		}

		if gitlabPipelinesLen == 0 {
			break
		}

		count += gitlabPipelinesLen
	}

	return count, nil
}

func pagedURL(uri url.URL, page string) url.URL {
	qParams := uri.Query()
	qParams.Set("page", fmt.Sprint(page))

	uri.RawQuery = qParams.Encode()

	return uri
}

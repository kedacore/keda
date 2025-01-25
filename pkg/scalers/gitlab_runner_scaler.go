package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type pipelineStatus string

const (
	// pipelinePendingStatus is the status of the pending pipelines.
	pipelinePendingStatus pipelineStatus = "pending"
	// pipelineWaitingForResourceStatus is the status of the pipelines that are waiting for resources.
	pipelineWaitingForResourceStatus pipelineStatus = "waiting_for_resource"
	// pipelineRunningStatus is the status of the running pipelines.
	pipelineRunningStatus pipelineStatus = "running"

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
	GitLabAPIURL        *url.URL `keda:"name=gitlabAPIURL, order=triggerMetadata, default=https://gitlab.com, optional"`
	PersonalAccessToken string   `keda:"name=personalAccessToken, order=authParams"`
	ProjectID           string   `keda:"name=projectID, order=triggerMetadata"`

	TargetPipelineQueueLength int64 `keda:"name=targetPipelineQueueLength, order=triggerMetadata, default=1, optional"`
	TriggerIndex              int
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
	meta := gitlabRunnerMetadata{}

	meta.TriggerIndex = config.TriggerIndex
	if err := config.TypedConfig(&meta); err != nil {
		return nil, fmt.Errorf("error parsing gitlabRunner metadata: %w", err)
	}

	return &meta, nil
}

func (s *gitlabRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	// Get the number of pending, waiting, and running for resource pipelines
	eg, ctx := errgroup.WithContext(ctx)

	getLen := func(status pipelineStatus, length *int64) func() error {
		return func() error {
			uri := constructGitlabAPIPipelinesURL(*s.metadata.GitLabAPIURL, s.metadata.ProjectID, status)
			var err error
			*length, err = s.getPipelineQueueLength(ctx, uri)
			return err
		}
	}

	var pendingLen, waitingForResourceLen, runningLen int64

	eg.Go(getLen(pipelinePendingStatus, &pendingLen))
	eg.Go(getLen(pipelineWaitingForResourceStatus, &waitingForResourceLen))
	eg.Go(getLen(pipelineRunningStatus, &runningLen))

	err := eg.Wait()
	if err != nil {
		s.logger.Error(err, "error getting pipeline queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	queueLen := pendingLen + waitingForResourceLen + runningLen

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen >= s.metadata.TargetPipelineQueueLength, nil
}

func (s *gitlabRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("gitlab-runner-%s", s.metadata.ProjectID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetPipelineQueueLength),
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

func constructGitlabAPIPipelinesURL(baseURL url.URL, projectID string, status pipelineStatus) url.URL {
	baseURL.Path = "/api/v4/projects/" + projectID + "/pipelines"

	qParams := baseURL.Query()
	qParams.Set("status", string(status))
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
	req.Header.Set("PRIVATE-TOKEN", s.metadata.PersonalAccessToken)

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
func (s *gitlabRunnerScaler) getPipelineQueueLength(ctx context.Context, baseURL url.URL) (int64, error) {
	var count int64

	page := 1
	for ; page < maxGitlabAPIPageCount; page++ {
		pagedURL := pagedURL(baseURL, fmt.Sprint(page))

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

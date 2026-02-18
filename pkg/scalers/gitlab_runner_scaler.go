package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const gitlabDefaultPerPage = 100

type gitlabRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *gitlabRunnerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type gitlabRunnerMetadata struct {
	GitLabAPIURL                string   `keda:"name=gitlabAPIURL, order=triggerMetadata;resolvedEnv, default=https://gitlab.com"`
	PersonalAccessToken         string   `keda:"name=personalAccessToken, order=authParams;resolvedEnv"`
	ProjectID                   string   `keda:"name=projectID, order=triggerMetadata;resolvedEnv, optional"`
	GroupID                     string   `keda:"name=groupID, order=triggerMetadata;resolvedEnv, optional"`
	JobScopes                   string   `keda:"name=jobScopes, order=triggerMetadata;resolvedEnv, optional, default=pending"`
	TargetQueueLength           int64    `keda:"name=targetQueueLength, order=triggerMetadata;resolvedEnv, default=1"`
	ActivationTargetQueueLength int64    `keda:"name=activationTargetQueueLength, order=triggerMetadata, default=0"`
	IncludeSubgroups            bool     `keda:"name=includeSubgroups, order=triggerMetadata;resolvedEnv, optional, default=true"`
	TagList                     []string `keda:"name=tagList, order=triggerMetadata;resolvedEnv, optional"`
	RunUntagged                 bool     `keda:"name=runUntagged, order=triggerMetadata;resolvedEnv, optional, default=false"`
	UnsafeSsl                   bool     `keda:"name=unsafeSsl, order=triggerMetadata, optional, default=false"`
	TriggerIndex                int
}

type gitlabJob struct {
	ID      int64    `json:"id"`
	TagList []string `json:"tag_list"`
}

type gitlabProject struct {
	ID int64 `json:"id"`
}

func (meta *gitlabRunnerMetadata) Validate() error {
	if meta.ProjectID == "" && meta.GroupID == "" {
		return fmt.Errorf("one of projectID or groupID must be provided")
	}
	if meta.ProjectID != "" && meta.GroupID != "" {
		return fmt.Errorf("only one of projectID or groupID can be provided, not both")
	}
	if meta.TargetQueueLength < 1 {
		return fmt.Errorf("targetQueueLength must be at least 1")
	}
	if meta.ActivationTargetQueueLength < 0 {
		return fmt.Errorf("activationTargetQueueLength must be at least 0")
	}

	return nil
}

func parseGitLabRunnerMetadata(config *scalersconfig.ScalerConfig) (*gitlabRunnerMetadata, error) {
	meta := &gitlabRunnerMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing gitlab runner metadata: %w", err)
	}

	meta.TriggerIndex = config.TriggerIndex
	meta.GitLabAPIURL = strings.TrimSuffix(meta.GitLabAPIURL, "/")

	return meta, nil
}

func NewGitLabRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGitLabRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing GitLab Runner metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	logger := InitializeLogger(config, "gitlab_runner_scaler")

	return &gitlabRunnerScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func (s *gitlabRunnerScaler) hasTagFiltering() bool {
	return len(s.metadata.TagList) > 0 || s.metadata.RunUntagged
}

func (s *gitlabRunnerScaler) canRunnerPickUpJob(jobTags []string) bool {
	if len(jobTags) == 0 {
		return s.metadata.RunUntagged || !s.hasTagFiltering()
	}

	for _, jobTag := range jobTags {
		found := false
		for _, runnerTag := range s.metadata.TagList {
			if strings.EqualFold(jobTag, runnerTag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (s *gitlabRunnerScaler) getGitLabRequest(ctx context.Context, url string) ([]byte, http.Header, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, -1, err
	}

	req.Header.Set("PRIVATE-TOKEN", s.metadata.PersonalAccessToken)

	r, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, -1, err
	}
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, r.StatusCode, err
	}

	if r.StatusCode == http.StatusTooManyRequests {
		retryAfter := r.Header.Get("Retry-After")
		return nil, r.Header, r.StatusCode, fmt.Errorf("GitLab API rate limit exceeded, retry after %s seconds", retryAfter)
	}

	if r.StatusCode != http.StatusOK {
		if remaining := r.Header.Get("RateLimit-Remaining"); remaining != "" {
			if rem, _ := strconv.Atoi(remaining); rem == 0 {
				resetTime, _ := strconv.ParseInt(r.Header.Get("RateLimit-Reset"), 10, 64)

				return nil, r.Header, r.StatusCode, fmt.Errorf("GitLab API rate limit exceeded, resets at %s", time.Unix(resetTime, 0))
			}
		}

		return nil, r.Header, r.StatusCode, fmt.Errorf("GitLab API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	return b, r.Header, r.StatusCode, nil
}

func (s *gitlabRunnerScaler) getGroupProjects(ctx context.Context) ([]gitlabProject, error) {
	var allProjects []gitlabProject
	page := 1

	for {
		url := fmt.Sprintf("%s/api/v4/groups/%s/projects?per_page=%d&page=%d&include_subgroups=%t&simple=true",
			s.metadata.GitLabAPIURL, s.metadata.GroupID, gitlabDefaultPerPage, page, s.metadata.IncludeSubgroups)

		body, _, statusCode, err := s.getGitLabRequest(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("error fetching group projects (status %d): %w", statusCode, err)
		}

		var projects []gitlabProject
		if err := json.Unmarshal(body, &projects); err != nil {
			return nil, fmt.Errorf("error parsing group projects response: %w", err)
		}

		allProjects = append(allProjects, projects...)

		if len(projects) < gitlabDefaultPerPage {
			break
		}

		page++
	}

	return allProjects, nil
}

func (s *gitlabRunnerScaler) getProjectPendingJobCount(ctx context.Context, projectID string) (int64, error) {
	scopes := strings.Split(s.metadata.JobScopes, ",")
	var totalCount int64

	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}

		page := 1
		for {
			url := fmt.Sprintf("%s/api/v4/projects/%s/jobs?per_page=%d&page=%d&scope[]=%s",
				s.metadata.GitLabAPIURL, projectID, gitlabDefaultPerPage, page, scope)

			body, headers, statusCode, err := s.getGitLabRequest(ctx, url)
			if err != nil {
				if statusCode == http.StatusNotFound {
					s.logger.V(1).Info("project not found, skipping", "projectID", projectID)
					return totalCount, nil
				}

				return totalCount, fmt.Errorf("error fetching jobs for project %s (status %d): %w", projectID, statusCode, err)
			}

			if !s.hasTagFiltering() {
				if xTotal := headers.Get("x-total"); xTotal != "" && page == 1 {
					count, err := strconv.ParseInt(xTotal, 10, 64)
					if err == nil {
						totalCount += count
						break
					}
				}
			}

			var jobs []gitlabJob
			if err := json.Unmarshal(body, &jobs); err != nil {
				return totalCount, fmt.Errorf("error parsing jobs response for project %s: %w", projectID, err)
			}

			if s.hasTagFiltering() {
				for _, job := range jobs {
					if s.canRunnerPickUpJob(job.TagList) {
						totalCount++
					}
				}
			} else {
				totalCount += int64(len(jobs))
			}

			if len(jobs) < gitlabDefaultPerPage {
				break
			}

			page++
		}
	}

	return totalCount, nil
}

func (s *gitlabRunnerScaler) GetQueueLength(ctx context.Context) (int64, error) {
	if s.metadata.ProjectID != "" {
		return s.getProjectPendingJobCount(ctx, s.metadata.ProjectID)
	}

	projects, err := s.getGroupProjects(ctx)
	if err != nil {
		return -1, err
	}

	var totalCount int64
	for _, project := range projects {
		count, err := s.getProjectPendingJobCount(ctx, strconv.FormatInt(project.ID, 10))
		if err != nil {
			return -1, err
		}

		totalCount += count
	}

	return totalCount, nil
}

func (s *gitlabRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueLen, err := s.GetQueueLength(ctx)
	if err != nil {
		s.logger.Error(err, "error getting GitLab job queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen > s.metadata.ActivationTargetQueueLength, nil
}

func (s *gitlabRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	metricName := "gitlab-runner"
	if s.metadata.ProjectID != "" {
		metricName = fmt.Sprintf("gitlab-runner-%s", s.metadata.ProjectID)
	} else if s.metadata.GroupID != "" {
		metricName = fmt.Sprintf("gitlab-runner-%s", s.metadata.GroupID)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetQueueLength),
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

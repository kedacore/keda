package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	giteaAdminJobPath = "/api/v1/admin/actions/jobs"
	giteaOrgJobPath   = "/api/v1/orgs/%s/actions/jobs"
	giteaRepoJobPath  = "/api/v1/repos/%s/%s/actions/jobs"
	giteaUserJobPath  = "/api/v1/user/actions/jobs"

	// Gitea clamps `limit` to MAX_RESPONSE_ITEMS (default 50) no matter what is
	// requested, so this is the real page size rather than a preference.
	giteaPageSize = 50

	// Upper bound on pages walked in a single poll when label filtering is on.
	// 20 pages x 50 = 1000 jobs, far above any queue that a scaler decision
	// could still be meaningfully affected by: long before that the workload is
	// pinned at maxReplicaCount. Hitting the cap is logged, never silent.
	giteaMaxPages = 20
)

// giteaJob is one entry of the `jobs` array returned by Gitea's Actions jobs
// endpoints. Only the fields the scaler actually reads are modelled.
type giteaJob struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
}

// giteaJobsResponse is Gitea's envelope. Note this differs from Forgejo, which
// returns a bare array — the two forges diverged on this API.
type giteaJobsResponse struct {
	Jobs       []giteaJob `json:"jobs"`
	TotalCount int64      `json:"total_count"`
}

type giteaRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *giteaRunnerMetadata
	client     *http.Client
	logger     logr.Logger
}

type giteaRunnerMetadata struct {
	TriggerIndex int

	Token   string `keda:"name=token, order=authParams;resolvedEnv"`
	Address string `keda:"name=address, order=triggerMetadata"`

	// Comma-separated runner labels. When empty the scaler counts every
	// pending job, which is both correct and cheaper — see getJobCount.
	Labels string `keda:"name=labels, order=triggerMetadata, optional"`

	// Scope selectors, evaluated in the order repo > org > global > user.
	Global bool   `keda:"name=global, order=triggerMetadata, optional"`
	Owner  string `keda:"name=owner, order=triggerMetadata, optional"`
	Org    string `keda:"name=org, order=triggerMetadata, optional"`
	Repo   string `keda:"name=repo, order=triggerMetadata, optional"`

	// Treat a job carrying no labels at all as matching only a scaler that
	// itself declares no labels. Mirrors the github-runner scaler's option.
	MatchUnlabeledJobsWithUnlabeledRunners bool `keda:"name=matchUnlabeledJobsWithUnlabeledRunners, order=triggerMetadata, default=false"`

	// Jobs one runner replica absorbs.
	TargetJobs int64 `keda:"name=targetJobs, order=triggerMetadata, default=1"`
}

func (m *giteaRunnerMetadata) labelList() []string {
	if m.Labels == "" {
		return nil
	}
	parts := strings.Split(m.Labels, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseGiteaRunnerMetadata(config *scalersconfig.ScalerConfig) (*giteaRunnerMetadata, error) {
	meta := &giteaRunnerMetadata{}
	meta.TriggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing gitea metadata: %w", err)
	}

	meta.Address = strings.TrimSuffix(meta.Address, "/")

	if meta.TargetJobs <= 0 {
		return nil, fmt.Errorf("targetJobs must be a positive number, got %d", meta.TargetJobs)
	}

	if meta.Repo != "" && meta.Owner == "" {
		return nil, fmt.Errorf("owner must be set when repo is set")
	}

	return meta, nil
}

// NewGiteaRunnerScaler creates a new Gitea Runner Scaler
func NewGiteaRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	c := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGiteaRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Gitea Runner metadata: %w", err)
	}

	return &giteaRunnerScaler{
		client:     c,
		metricType: metricType,
		metadata:   meta,
		logger:     InitializeLogger(config, "gitea_runner_scaler"),
	}, nil
}

func (s *giteaRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	count, err := s.getJobCount(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(count))

	return []external_metrics.ExternalMetricValue{metric}, count > 0, nil
}

func (s *giteaRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString("gitea")),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetJobs),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// getJobCount returns the number of jobs that are queued OR already running.
//
// Counting running work as well as queued work is deliberate and load-bearing.
// A "how many jobs are waiting" metric collapses to zero the moment those jobs
// are picked up, so the scaler would immediately tear down the very replicas
// doing the work. Gitea lets both statuses be requested in one call — the
// filter is a genuine OR — so this costs nothing extra.
//
// Two paths, and the difference matters for API load:
//
//   - No labels configured: total_count honours the status filter, so a single
//     request with limit=1 yields the answer. One call per poll, no pagination.
//   - Labels configured: Gitea ignores a `labels` query parameter entirely
//     (verified against 1.27) so filtering must happen client-side, which means
//     actually walking the pages.
func (s *giteaRunnerScaler) getJobCount(ctx context.Context) (int64, error) {
	labels := s.metadata.labelList()

	if len(labels) == 0 && !s.metadata.MatchUnlabeledJobsWithUnlabeledRunners {
		resp, err := s.fetchJobsPage(ctx, 1, 1)
		if err != nil {
			return 0, err
		}
		return resp.TotalCount, nil
	}

	var count int64
	for page := 1; page <= giteaMaxPages; page++ {
		resp, err := s.fetchJobsPage(ctx, page, giteaPageSize)
		if err != nil {
			return 0, err
		}

		for _, job := range resp.Jobs {
			if canGiteaRunnerMatchLabels(job.Labels, labels, s.metadata.MatchUnlabeledJobsWithUnlabeledRunners) {
				count++
			}
		}

		// A short page is the last page.
		if len(resp.Jobs) < giteaPageSize {
			return count, nil
		}

		if page == giteaMaxPages {
			s.logger.V(0).Info(
				"Gitea job pagination hit its page cap; the reported count is a lower bound",
				"pagesWalked", giteaMaxPages,
				"jobsInspected", giteaMaxPages*giteaPageSize,
				"totalCount", resp.TotalCount,
				"matched", count,
			)
		}
	}

	return count, nil
}

func (s *giteaRunnerScaler) fetchJobsPage(ctx context.Context, page, limit int) (*giteaJobsResponse, error) {
	uri, err := s.getJobsURL(page, limit)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.metadata.Token))

	r, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			s.logger.Error(closeErr, "Failed to close response body")
		}
	}()

	if r.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, r.Body)
		return nil, fmt.Errorf("the Gitea REST API returned error. url: %s status: %d", uri.Path, r.StatusCode)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	resp := &giteaJobsResponse{}
	if err := json.Unmarshal(b, resp); err != nil {
		return nil, fmt.Errorf("error unmarshalling Gitea jobs response: %w", err)
	}

	return resp, nil
}

// getJobsURL builds the endpoint for the configured scope. Gitea exposes the
// same jobs resource at four access levels; the narrowest configured one wins.
func (s *giteaRunnerScaler) getJobsURL(page, limit int) (*url.URL, error) {
	var path string

	switch {
	case s.metadata.Owner != "" && s.metadata.Repo != "":
		path = fmt.Sprintf(giteaRepoJobPath, url.PathEscape(s.metadata.Owner), url.PathEscape(s.metadata.Repo))
	case s.metadata.Org != "":
		path = fmt.Sprintf(giteaOrgJobPath, url.PathEscape(s.metadata.Org))
	case s.metadata.Global:
		path = giteaAdminJobPath
	default:
		path = giteaUserJobPath
	}

	u, err := url.Parse(s.metadata.Address + path)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	// Repeated `status` keys are OR-ed by Gitea, and total_count honours them.
	q.Add("status", "queued")
	q.Add("status", "in_progress")
	q.Add("page", fmt.Sprintf("%d", page))
	q.Add("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	return u, nil
}

// canGiteaRunnerMatchLabels reports whether a runner offering runnerLabels is
// able to take a job requesting jobLabels — i.e. every label the job asks for
// is one the runner provides. Mirrors the github-runner scaler's semantics so
// the two behave the same way for users moving between forges.
func canGiteaRunnerMatchLabels(jobLabels, runnerLabels []string, matchUnlabeled bool) bool {
	if matchUnlabeled && len(jobLabels) == 0 {
		return len(runnerLabels) == 0
	}
	for _, jobLabel := range jobLabels {
		if !contains(runnerLabels, jobLabel) {
			return false
		}
	}
	return true
}

func (s *giteaRunnerScaler) Close(_ context.Context) error {
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	return nil
}

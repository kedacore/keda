package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	defaultTargetWorkflowQueueLength = 1
)

var (
	// githubAPIResetTime is the time when the GitHub API rate limit resets
	githubAPIResetTime time.Time
	githubAPIRemaining int
)

type githubRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *githubRunnerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type githubRunnerMetadata struct {
	githubAPIURL                        string
	owner                               string
	organization                        bool
	personalAccessToken                 string
	repo                                *string
	targetWorkflowQueueLength           int64
	activationTargetWorkflowQueueLength int64
	scalerIndex                         int
}

type WorkflowRuns struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

type WorkflowRun struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	NodeID           string  `json:"node_id"`
	HeadBranch       string  `json:"head_branch"`
	HeadSha          string  `json:"head_sha"`
	Path             string  `json:"path"`
	DisplayTitle     string  `json:"display_title"`
	RunNumber        int     `json:"run_number"`
	Event            string  `json:"event"`
	Status           string  `json:"status"`
	Conclusion       *string `json:"conclusion"`
	WorkflowID       int     `json:"workflow_id"`
	CheckSuiteID     int64   `json:"check_suite_id"`
	CheckSuiteNodeID string  `json:"check_suite_node_id"`
	URL              string  `json:"url"`
	HTMLURL          string  `json:"html_url"`
	PullRequests     []struct {
		URL    string `json:"url"`
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Head   struct {
			Ref  string `json:"ref"`
			Sha  string `json:"sha"`
			Repo struct {
				ID   int    `json:"id"`
				URL  string `json:"url"`
				Name string `json:"name"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Ref  string `json:"ref"`
			Sha  string `json:"sha"`
			Repo struct {
				ID   int    `json:"id"`
				URL  string `json:"url"`
				Name string `json:"name"`
			} `json:"repo"`
		} `json:"base"`
	} `json:"pull_requests"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Actor     struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"actor"`
	RunAttempt          int           `json:"run_attempt"`
	ReferencedWorkflows []interface{} `json:"referenced_workflows"`
	RunStartedAt        time.Time     `json:"run_started_at"`
	TriggeringActor     struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"triggering_actor"`
	JobsURL            string  `json:"jobs_url"`
	LogsURL            string  `json:"logs_url"`
	CheckSuiteURL      string  `json:"check_suite_url"`
	ArtifactsURL       string  `json:"artifacts_url"`
	CancelURL          string  `json:"cancel_url"`
	RerunURL           string  `json:"rerun_url"`
	PreviousAttemptURL *string `json:"previous_attempt_url"`
	WorkflowURL        string  `json:"workflow_url"`
	HeadCommit         struct {
		ID        string    `json:"id"`
		TreeID    string    `json:"tree_id"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Committer struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"committer"`
	} `json:"head_commit"`
	Repository     Repo `json:"repository"`
	HeadRepository Repo `json:"head_repository"`
}

type Repos struct {
	Repo []Repo
}

type Repo struct {
	ID       int    `json:"id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"owner"`
	Private          bool        `json:"private"`
	HTMLURL          string      `json:"html_url"`
	Description      string      `json:"description"`
	Fork             bool        `json:"fork"`
	URL              string      `json:"url"`
	ArchiveURL       string      `json:"archive_url"`
	AssigneesURL     string      `json:"assignees_url"`
	BlobsURL         string      `json:"blobs_url"`
	BranchesURL      string      `json:"branches_url"`
	CollaboratorsURL string      `json:"collaborators_url"`
	CommentsURL      string      `json:"comments_url"`
	CommitsURL       string      `json:"commits_url"`
	CompareURL       string      `json:"compare_url"`
	ContentsURL      string      `json:"contents_url"`
	ContributorsURL  string      `json:"contributors_url"`
	DeploymentsURL   string      `json:"deployments_url"`
	DownloadsURL     string      `json:"downloads_url"`
	EventsURL        string      `json:"events_url"`
	ForksURL         string      `json:"forks_url"`
	GitCommitsURL    string      `json:"git_commits_url"`
	GitRefsURL       string      `json:"git_refs_url"`
	GitTagsURL       string      `json:"git_tags_url"`
	GitURL           string      `json:"git_url"`
	IssueCommentURL  string      `json:"issue_comment_url"`
	IssueEventsURL   string      `json:"issue_events_url"`
	IssuesURL        string      `json:"issues_url"`
	KeysURL          string      `json:"keys_url"`
	LabelsURL        string      `json:"labels_url"`
	LanguagesURL     string      `json:"languages_url"`
	MergesURL        string      `json:"merges_url"`
	MilestonesURL    string      `json:"milestones_url"`
	NotificationsURL string      `json:"notifications_url"`
	PullsURL         string      `json:"pulls_url"`
	ReleasesURL      string      `json:"releases_url"`
	SSHURL           string      `json:"ssh_url"`
	StargazersURL    string      `json:"stargazers_url"`
	StatusesURL      string      `json:"statuses_url"`
	SubscribersURL   string      `json:"subscribers_url"`
	SubscriptionURL  string      `json:"subscription_url"`
	TagsURL          string      `json:"tags_url"`
	TeamsURL         string      `json:"teams_url"`
	TreesURL         string      `json:"trees_url"`
	CloneURL         string      `json:"clone_url"`
	MirrorURL        string      `json:"mirror_url"`
	HooksURL         string      `json:"hooks_url"`
	SvnURL           string      `json:"svn_url"`
	Homepage         string      `json:"homepage"`
	Language         interface{} `json:"language"`
	ForksCount       int         `json:"forks_count"`
	StargazersCount  int         `json:"stargazers_count"`
	WatchersCount    int         `json:"watchers_count"`
	Size             int         `json:"size"`
	DefaultBranch    string      `json:"default_branch"`
	OpenIssuesCount  int         `json:"open_issues_count"`
	IsTemplate       bool        `json:"is_template"`
	Topics           []string    `json:"topics"`
	HasIssues        bool        `json:"has_issues"`
	HasProjects      bool        `json:"has_projects"`
	HasWiki          bool        `json:"has_wiki"`
	HasPages         bool        `json:"has_pages"`
	HasDownloads     bool        `json:"has_downloads"`
	Archived         bool        `json:"archived"`
	Disabled         bool        `json:"disabled"`
	Visibility       string      `json:"visibility"`
	PushedAt         time.Time   `json:"pushed_at"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Permissions      struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	AllowRebaseMerge    bool        `json:"allow_rebase_merge"`
	TemplateRepository  interface{} `json:"template_repository"`
	TempCloneToken      string      `json:"temp_clone_token"`
	AllowSquashMerge    bool        `json:"allow_squash_merge"`
	AllowAutoMerge      bool        `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool        `json:"delete_branch_on_merge"`
	AllowMergeCommit    bool        `json:"allow_merge_commit"`
	SubscribersCount    int         `json:"subscribers_count"`
	NetworkCount        int         `json:"network_count"`
	License             struct {
		Key     string `json:"key"`
		Name    string `json:"name"`
		URL     string `json:"url"`
		SpdxID  string `json:"spdx_id"`
		NodeID  string `json:"node_id"`
		HTMLURL string `json:"html_url"`
	} `json:"license"`
	Forks      int `json:"forks"`
	OpenIssues int `json:"open_issues"`
	Watchers   int `json:"watchers"`
}

type Jobs struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

type Job struct {
	ID          int       `json:"id"`
	RunID       int       `json:"run_id"`
	RunURL      string    `json:"run_url"`
	NodeID      string    `json:"node_id"`
	HeadSha     string    `json:"head_sha"`
	URL         string    `json:"url"`
	HtmlURL     string    `json:"html_url"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Name        string    `json:"name"`
	Steps       []struct {
		Name        string    `json:"name"`
		Status      string    `json:"status"`
		Conclusion  string    `json:"conclusion"`
		Number      int       `json:"number"`
		StartedAt   time.Time `json:"started_at"`
		CompletedAt time.Time `json:"completed_at"`
	} `json:"steps"`
	CheckRunURL     string   `json:"check_run_url"`
	Labels          []string `json:"labels"`
	RunnerID        int      `json:"runner_id"`
	RunnerName      string   `json:"runner_name"`
	RunnerGroupID   int      `json:"runner_group_id"`
	RunnerGroupName string   `json:"runner_group_name"`
	WorkflowName    string   `json:"workflow_name"`
	HeadBranch      string   `json:"head_branch"`
}

// NewGitHubRunnerScaler creates a new GitHub Runner Scaler
func NewGitHubRunnerScaler(config *ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGitHubRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing GitHub Runner metadata: %s", err)
	}

	return &githubRunnerScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "github_runner_scaler"),
	}, nil
}

func parseGitHubRunnerMetadata(config *ScalerConfig) (*githubRunnerMetadata, error) {
	meta := githubRunnerMetadata{}
	meta.targetWorkflowQueueLength = defaultTargetWorkflowQueueLength
	meta.organization = false

	if val, ok := config.TriggerMetadata["targetWorkflowQueueLength"]; ok {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing azure pipelines metadata targetPipelinesQueueLength: %w", err)
		}
		meta.targetWorkflowQueueLength = queueLength
	}

	if val, ok := config.TriggerMetadata["organization"]; ok {
		organization, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing azure pipelines metadata organization: %w", err)
		}
		meta.organization = organization
	}

	meta.activationTargetWorkflowQueueLength = 0
	if val, ok := config.TriggerMetadata["activationTargetWorkflowQueueLength"]; ok {
		activationQueueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing azure pipelines metadata activationTargetPipelinesQueueLength: %w", err)
		}
		meta.activationTargetWorkflowQueueLength = activationQueueLength
	}

	if val, ok := config.AuthParams["owner"]; ok && val != "" {
		// Found the owner in a parameter from TriggerAuthentication
		meta.owner = val
	} else if val, ok := config.TriggerMetadata["ownerFromEnv"]; ok && val != "" {
		meta.owner = config.ResolvedEnv[val]
	} else if val, ok := config.TriggerMetadata["owner"]; ok && val != "" {
		meta.owner = val
	} else {
		return nil, fmt.Errorf("no owner given")
	}

	if val, ok := config.TriggerMetadata["repo"]; ok {
		meta.repo = &val
	}

	if val, ok := config.AuthParams["githubApiURL"]; ok && val != "" {
		// Found the githubApiURL in a parameter from TriggerAuthentication
		meta.githubAPIURL = val
	} else if val, ok := config.TriggerMetadata["githubApiURL"]; ok && val != "" {
		meta.githubAPIURL = val
	} else if val, ok := config.TriggerMetadata["githubApiURLFromEnv"]; ok && val != "" {
		meta.githubAPIURL = config.ResolvedEnv[val]
	} else {
		return nil, fmt.Errorf("no githubApiURL given")
	}

	if val, ok := config.AuthParams["personalAccessToken"]; ok && val != "" {
		// Found the personalAccessToken in a parameter from TriggerAuthentication
		meta.personalAccessToken = val
	} else if val, ok := config.TriggerMetadata["personalAccessToken"]; ok && val != "" {
		meta.personalAccessToken = val
	} else if val, ok := config.TriggerMetadata["personalAccessTokenFromEnv"]; ok && val != "" {
		meta.personalAccessToken = config.ResolvedEnv[val]
	} else {
		return nil, fmt.Errorf("no personalAccessToken given")
	}

	return &meta, nil
}

// getUserRepositories returns a list of repositories for a given organization
func (s *githubRunnerScaler) getUserRepositories(ctx context.Context) (*[]string, error) {
	url := fmt.Sprintf("%s/users/%s/repos", s.metadata.githubAPIURL, s.metadata.owner)
	body, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		s.logger.Error(err, "Cannot unmarshal GitHub Workflow Repos API response")
		return nil, err
	}

	var repoList []string
	for _, repo := range repos {
		repoList = append(repoList, repo.Name)
	}

	return &repoList, nil
}

// getOrganizationRepositories returns a list of repositories for a given organization
func (s *githubRunnerScaler) getOrganizationRepositories(ctx context.Context) (*[]string, error) {
	url := fmt.Sprintf("%s/orgs/%s/repos", s.metadata.githubAPIURL, s.metadata.owner)
	body, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		s.logger.Error(err, "Cannot unmarshal GitHub Workflow Repos API response")
		return nil, err
	}

	var repoList []string
	for _, repo := range repos {
		repoList = append(repoList, repo.Name)
	}

	return &repoList, nil
}

func getGithubRequest(ctx context.Context, url string, metadata *githubRunnerMetadata, httpClient *http.Client) ([]byte, error) {
	if githubAPIResetTime.Before(time.Now()) && githubAPIRemaining == 0 {
		return []byte{}, fmt.Errorf("GitHub API rate limit exceeded resets at %s", githubAPIResetTime)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Bearer "+metadata.personalAccessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	r, err := httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	githubAPIRemaining, err := strconv.Atoi(r.Header.Get("X-RateLimit-Remaining"))
	if err != nil {
		return []byte{}, fmt.Errorf("cannot get the rate limit remaining from API Call")
	}

	if githubAPIRemaining == 0 {
		resetTime, _ := strconv.ParseInt(r.Header.Get("X-RateLimit-Reset"), 10, 64)
		githubAPIResetTime = time.Unix(resetTime, 0)
		return []byte{}, fmt.Errorf("github rate limit exceeded, resets at %s", githubAPIResetTime)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return []byte{}, err
	}
	r.Body.Close()

	if r.StatusCode != 200 {
		return []byte{}, fmt.Errorf("the GitHub REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	return b, nil
}

func stripDeadRuns(allWfrs []WorkflowRuns) []WorkflowRun {
	var filtered []WorkflowRun
	for _, wfrs := range allWfrs {
		for _, wfr := range wfrs.WorkflowRuns {
			if wfr.Status == "queued" {
				filtered = append(filtered, wfr)
			}
		}
	}
	return filtered
}

// getWorkflowRunJobs returns a list of jobs for a given workflow run
func (s *githubRunnerScaler) getWorkflowRunJobs(ctx context.Context, workflowRunID int64, repoName string) ([]Job, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d/jobs", s.metadata.githubAPIURL, s.metadata.owner, repoName, workflowRunID)
	body, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}

	var jobs Jobs
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		s.logger.Error(err, "Cannot unmarshal GitHub Workflow Jobs API response")
		return nil, err
	}

	return jobs.Jobs, nil
}

// GetWorkflowQueueLength returns the number of workflows in the queue
func (s *githubRunnerScaler) GetWorkflowQueueLength(ctx context.Context) (int64, error) {
	var repos *[]string
	var err error

	if s.metadata.repo == nil {
		if s.metadata.organization {
			repos, err = s.getOrganizationRepositories(ctx)
			if err != nil {
				return -1, err
			}
		} else {
			repos, err = s.getUserRepositories(ctx)
			if err != nil {
				return -1, err
			}
		}
	} else {
		repos = &[]string{*s.metadata.repo}
	}

	var allWfrs []WorkflowRuns

	for _, repo := range *repos {
		url := fmt.Sprintf("%s/repos/%s/%s/actions/runs", s.metadata.githubAPIURL, s.metadata.owner, repo)
		body, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)

		if err != nil {
			return -1, err
		}

		var wfrs WorkflowRuns
		err = json.Unmarshal(body, &wfrs)
		if err != nil {
			s.logger.Error(err, "Cannot unmarshal GitHub Workflow Runs API response")
			return -1, err
		}
		allWfrs = append(allWfrs, wfrs)
	}

	var queueCount int64

	wfrs := stripDeadRuns(allWfrs)
	for _, wfr := range wfrs {
		jobs, err := s.getWorkflowRunJobs(ctx, wfr.ID, wfr.Repository.Name)
		if err != nil {
			return -1, err
		}
		for _, job := range jobs {
			if job.Status == "queued" {
				queueCount++
			}
		}
	}

	return queueCount, nil
}

func (s *githubRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueLen, err := s.GetWorkflowQueueLength(ctx)

	if err != nil {
		s.logger.Error(err, "error getting workflow queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen > s.metadata.activationTargetWorkflowQueueLength, nil
}

func (s *githubRunnerScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("github-runner-%s", s.metadata.owner))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetWorkflowQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *githubRunnerScaler) Close(ctx context.Context) error {
	return nil
}

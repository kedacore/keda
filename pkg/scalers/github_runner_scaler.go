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

	gha "github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetWorkflowQueueLength = 1
	defaultGithubAPIURL              = "https://api.github.com"
	ORG                              = "org"
	ENT                              = "ent"
	REPO                             = "repo"
)

var reservedLabels = []string{"self-hosted", "linux", "x64"}

type githubRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *githubRunnerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type githubRunnerMetadata struct {
	githubAPIURL              string
	owner                     string
	runnerScope               string
	personalAccessToken       *string
	repos                     []string
	labels                    []string
	targetWorkflowQueueLength int64
	scalerIndex               int
	applicationID             *int64
	installationID            *int64
	applicationKey            *string
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
	HTMLURL     string    `json:"html_url"`
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
		return nil, fmt.Errorf("error parsing GitHub Runner metadata: %w", err)
	}

	if meta.applicationID != nil && meta.installationID != nil && meta.applicationKey != nil {
		httpTrans := kedautil.CreateHTTPTransport(false)
		hc, err := gha.New(httpTrans, *meta.applicationID, *meta.installationID, []byte(*meta.applicationKey))
		if err != nil {
			return nil, fmt.Errorf("error creating GitHub App client: %w, \n appID: %d, instID: %d", err, meta.applicationID, meta.installationID)
		}
		httpClient = &http.Client{Transport: hc}
	}

	return &githubRunnerScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "github_runner_scaler"),
	}, nil
}

// getValueFromMetaOrEnv returns the value of the given key from the metadata or the environment variables
func getValueFromMetaOrEnv(key string, metadata map[string]string, env map[string]string) (string, error) {
	if val, ok := metadata[key]; ok && val != "" {
		return val, nil
	} else if val, ok := metadata[key+"FromEnv"]; ok && val != "" {
		return env[val], nil
	}
	return "", fmt.Errorf("no %s given", key)
}

// getInt64ValueFromMetaOrEnv returns the value of the given key from the metadata or the environment variables
func getInt64ValueFromMetaOrEnv(key string, config *ScalerConfig) (int64, error) {
	sInt, err := getValueFromMetaOrEnv(key, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return -1, fmt.Errorf("error parsing %s: %w", key, err)
	}

	goodInt, err := strconv.ParseInt(sInt, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("error parsing %s: %w", key, err)
	}
	return goodInt, nil
}

func parseGitHubRunnerMetadata(config *ScalerConfig) (*githubRunnerMetadata, error) {
	meta := &githubRunnerMetadata{}
	meta.targetWorkflowQueueLength = defaultTargetWorkflowQueueLength

	if val, err := getValueFromMetaOrEnv("runnerScope", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.runnerScope = val
	} else {
		return nil, err
	}

	if val, err := getValueFromMetaOrEnv("owner", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.owner = val
	} else {
		return nil, err
	}

	if val, err := getInt64ValueFromMetaOrEnv("targetWorkflowQueueLength", config); err == nil && val != -1 {
		meta.targetWorkflowQueueLength = val
	} else {
		meta.targetWorkflowQueueLength = defaultTargetWorkflowQueueLength
	}

	if val, err := getValueFromMetaOrEnv("labels", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.labels = strings.Split(val, ",")
	}

	if val, err := getValueFromMetaOrEnv("repos", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.repos = strings.Split(val, ",")
	}

	if val, err := getValueFromMetaOrEnv("githubApiURL", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.githubAPIURL = val
	} else {
		meta.githubAPIURL = defaultGithubAPIURL
	}

	if val, ok := config.AuthParams["personalAccessToken"]; ok && val != "" {
		// Found the pat token in a parameter from TriggerAuthentication
		meta.personalAccessToken = &val
	}

	if appID, instID, key, err := setupGitHubApp(config); err == nil {
		meta.applicationID = appID
		meta.installationID = instID
		meta.applicationKey = key
	} else {
		return nil, err
	}

	if meta.applicationKey == nil && meta.personalAccessToken == nil {
		return nil, fmt.Errorf("no personalAccessToken or appKey given")
	}

	meta.scalerIndex = config.ScalerIndex

	return meta, nil
}

func setupGitHubApp(config *ScalerConfig) (*int64, *int64, *string, error) {
	var appID *int64
	var instID *int64
	var appKey *string

	if val, err := getInt64ValueFromMetaOrEnv("applicationID", config); err == nil && val != -1 {
		appID = &val
	}

	if val, err := getInt64ValueFromMetaOrEnv("installationID", config); err == nil && val != -1 {
		instID = &val
	}

	if val, ok := config.AuthParams["appKey"]; ok && val != "" {
		appKey = &val
	}

	if (appID != nil || instID != nil || appKey != nil) &&
		(appID == nil || instID == nil || appKey == nil) {
		return nil, nil, nil, fmt.Errorf("applicationID, installationID and applicationKey must be given")
	}

	return appID, instID, appKey, nil
}

// getRepositories returns a list of repositories for a given organization, user or enterprise
func (s *githubRunnerScaler) getRepositories(ctx context.Context) ([]string, error) {
	if s.metadata.repos != nil {
		return s.metadata.repos, nil
	}

	var url string
	switch s.metadata.runnerScope {
	case ORG:
		url = fmt.Sprintf("%s/orgs/%s/repos", s.metadata.githubAPIURL, s.metadata.owner)
	case REPO:
		url = fmt.Sprintf("%s/users/%s/repos", s.metadata.githubAPIURL, s.metadata.owner)
	case ENT:
		url = fmt.Sprintf("%s/orgs/%s/repos", s.metadata.githubAPIURL, s.metadata.owner)
	default:
		return nil, fmt.Errorf("runnerScope %s not supported", s.metadata.runnerScope)
	}
	body, _, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		return nil, err
	}

	var repoList []string
	for _, repo := range repos {
		repoList = append(repoList, repo.Name)
	}

	return repoList, nil
}

func getGithubRequest(ctx context.Context, url string, metadata *githubRunnerMetadata, httpClient *http.Client) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []byte{}, -1, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if metadata.applicationID == nil && metadata.personalAccessToken != nil {
		req.Header.Set("Authorization", "Bearer "+*metadata.personalAccessToken)
	}

	r, err := httpClient.Do(req)
	if err != nil {
		return []byte{}, -1, err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return []byte{}, -1, err
	}
	_ = r.Body.Close()

	if r.StatusCode != 200 {
		if r.Header.Get("X-RateLimit-Remaining") != "" {
			githubAPIRemaining, _ := strconv.Atoi(r.Header.Get("X-RateLimit-Remaining"))

			if githubAPIRemaining == 0 {
				resetTime, _ := strconv.ParseInt(r.Header.Get("X-RateLimit-Reset"), 10, 64)
				return []byte{}, r.StatusCode, fmt.Errorf("GitHub API rate limit exceeded, resets at %s", time.Unix(resetTime, 0))
			}
		}

		return []byte{}, r.StatusCode, fmt.Errorf("the GitHub REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	return b, r.StatusCode, nil
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
	body, _, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}

	var jobs Jobs
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		return nil, err
	}

	return jobs.Jobs, nil
}

// getWorkflowRuns returns a list of workflow runs for a given repository
func (s *githubRunnerScaler) getWorkflowRuns(ctx context.Context, repoName string) (*WorkflowRuns, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs", s.metadata.githubAPIURL, s.metadata.owner, repoName)
	body, statusCode, err := getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil && statusCode == 404 {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var wfrs WorkflowRuns
	err = json.Unmarshal(body, &wfrs)
	if err != nil {
		return nil, err
	}

	return &wfrs, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

// canRunnerMatchLabels check Agent Label array will match runner label array
func canRunnerMatchLabels(jobLabels []string, runnerLabels []string) bool {
	for _, jobLabel := range jobLabels {
		if !contains(runnerLabels, jobLabel) && !contains(reservedLabels, jobLabel) {
			return false
		}
	}
	return true
}

// GetWorkflowQueueLength returns the number of workflow jobs in the queue
func (s *githubRunnerScaler) GetWorkflowQueueLength(ctx context.Context) (int64, error) {
	var repos []string
	var err error

	repos, err = s.getRepositories(ctx)
	if err != nil {
		return -1, err
	}

	var allWfrs []WorkflowRuns

	for _, repo := range repos {
		wfrs, err := s.getWorkflowRuns(ctx, repo)
		if err != nil {
			return -1, err
		}
		if wfrs != nil {
			allWfrs = append(allWfrs, *wfrs)
		}
	}

	var queueCount int64

	wfrs := stripDeadRuns(allWfrs)
	for _, wfr := range wfrs {
		jobs, err := s.getWorkflowRunJobs(ctx, wfr.ID, wfr.Repository.Name)
		if err != nil {
			return -1, err
		}
		for _, job := range jobs {
			if job.Status == "queued" && canRunnerMatchLabels(job.Labels, s.metadata.labels) {
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

	return []external_metrics.ExternalMetricValue{metric}, queueLen >= s.metadata.targetWorkflowQueueLength, nil
}

func (s *githubRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("github-runner-%s", s.metadata.owner))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetWorkflowQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *githubRunnerScaler) Close(_ context.Context) error {
	return nil
}

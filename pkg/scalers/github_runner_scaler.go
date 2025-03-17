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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	metricType    v2.MetricTargetType
	metadata      *githubRunnerMetadata
	httpClient    *http.Client
	logger        logr.Logger
	etags         map[string]string
	previousRepos []string
	previousWfrs  map[string]map[string]*WorkflowRuns
	previousJobs  map[string][]Job
	rateLimits    RateLimits
}

type githubRunnerMetadata struct {
	githubAPIURL              string
	owner                     string
	runnerScope               string
	personalAccessToken       *string
	repos                     []string
	labels                    []string
	noDefaultLabels           bool
	enableEtags               bool
	enableBackoff             bool
	targetWorkflowQueueLength int64
	triggerIndex              int
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

type RateLimits struct {
	Remaining      int       `json:"remaining"`
	ResetTime      time.Time `json:"resetTime"`
	RetryAfterTime time.Time `json:"retryAfterTime"`
}

// NewGitHubRunnerScaler creates a new GitHub Runner Scaler
func NewGitHubRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
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
		hc.BaseURL = meta.githubAPIURL
		httpClient = &http.Client{Transport: hc}
	}

	etags := make(map[string]string)
	previousRepos := []string{}
	previousJobs := make(map[string][]Job)
	previousWfrs := make(map[string]map[string]*WorkflowRuns)
	rateLimits := RateLimits{}

	return &githubRunnerScaler{
		metricType:    metricType,
		metadata:      meta,
		httpClient:    httpClient,
		logger:        InitializeLogger(config, "github_runner_scaler"),
		etags:         etags,
		previousRepos: previousRepos,
		previousJobs:  previousJobs,
		previousWfrs:  previousWfrs,
		rateLimits:    rateLimits,
	}, nil
}

// getValueFromMetaOrEnv returns the value of the given key from the metadata or the environment variables
func getValueFromMetaOrEnv(key string, metadata map[string]string, env map[string]string) (string, error) {
	if val, ok := metadata[key]; ok && val != "" {
		return val, nil
	} else if val, ok := metadata[key+"FromEnv"]; ok && val != "" {
		if envVal, ok := env[val]; ok && envVal != "" {
			return envVal, nil
		}
		return "", fmt.Errorf("%s %s env variable value is empty", key, val)
	}

	return "", fmt.Errorf("no %s given", key)
}

// getInt64ValueFromMetaOrEnv returns the value of the given key from the metadata or the environment variables
func getInt64ValueFromMetaOrEnv(key string, metadata map[string]string, env map[string]string) (int64, error) {
	sInt, err := getValueFromMetaOrEnv(key, metadata, env)
	if err != nil {
		return -1, fmt.Errorf("error parsing %s: %w", key, err)
	}

	goodInt, err := strconv.ParseInt(sInt, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("error parsing %s: %w", key, err)
	}
	return goodInt, nil
}

// getInt64ValueFromMetaOrEnv returns the value of the given key from the metadata or the environment variables
func getBoolValueFromMetaOrEnv(key string, metadata map[string]string, env map[string]string) (bool, error) {
	sBool, err := getValueFromMetaOrEnv(key, metadata, env)
	if err != nil {
		return false, fmt.Errorf("error parsing %s: %w", key, err)
	}

	goodBool, err := strconv.ParseBool(sBool)
	if err != nil {
		return false, fmt.Errorf("error parsing %s: %w", key, err)
	}
	return goodBool, nil
}

func parseGitHubRunnerMetadata(config *scalersconfig.ScalerConfig) (*githubRunnerMetadata, error) {
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

	if val, err := getInt64ValueFromMetaOrEnv("targetWorkflowQueueLength", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != -1 {
		meta.targetWorkflowQueueLength = val
	} else {
		meta.targetWorkflowQueueLength = defaultTargetWorkflowQueueLength
	}

	if val, err := getValueFromMetaOrEnv("labels", config.TriggerMetadata, config.ResolvedEnv); err == nil && val != "" {
		meta.labels = strings.Split(val, ",")
	}

	if val, err := getBoolValueFromMetaOrEnv("noDefaultLabels", config.TriggerMetadata, config.ResolvedEnv); err == nil {
		meta.noDefaultLabels = val
	} else {
		meta.noDefaultLabels = false
	}

	if val, err := getBoolValueFromMetaOrEnv("enableEtags", config.TriggerMetadata, config.ResolvedEnv); err == nil {
		meta.enableEtags = val
	} else {
		meta.enableEtags = false
	}

	if val, err := getBoolValueFromMetaOrEnv("enableBackoff", config.TriggerMetadata, config.ResolvedEnv); err == nil {
		meta.enableBackoff = val
	} else {
		meta.enableBackoff = false
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

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func setupGitHubApp(config *scalersconfig.ScalerConfig) (*int64, *int64, *string, error) {
	var appID *int64
	var instID *int64
	var appKey *string

	appIDVal, appIDErr := getInt64ValueFromMetaOrEnv("applicationID", config.TriggerMetadata, config.ResolvedEnv)
	if appIDErr == nil && appIDVal != -1 {
		appID = &appIDVal
	}

	instIDVal, instIDErr := getInt64ValueFromMetaOrEnv("installationID", config.TriggerMetadata, config.ResolvedEnv)
	if instIDErr == nil && instIDVal != -1 {
		instID = &instIDVal
	}

	if val, ok := config.AuthParams["appKey"]; ok && val != "" {
		appKey = &val
	}

	if (appID != nil || instID != nil || appKey != nil) &&
		(appID == nil || instID == nil || appKey == nil) {
		if appIDErr != nil {
			return nil, nil, nil, appIDErr
		}

		if instIDErr != nil {
			return nil, nil, nil, instIDErr
		}

		return nil, nil, nil, fmt.Errorf("no applicationKey given")
	}

	return appID, instID, appKey, nil
}

// getRepositories returns a list of repositories for a given organization, user or enterprise
func (s *githubRunnerScaler) getRepositories(ctx context.Context) ([]string, error) {
	if s.metadata.repos != nil {
		return s.metadata.repos, nil
	}

	page := 1
	var repoList []string

	for {
		var url string
		switch s.metadata.runnerScope {
		case ORG:
			url = fmt.Sprintf("%s/orgs/%s/repos?page=%s", s.metadata.githubAPIURL, s.metadata.owner, strconv.Itoa(page))
		case REPO:
			url = fmt.Sprintf("%s/users/%s/repos?page=%s", s.metadata.githubAPIURL, s.metadata.owner, strconv.Itoa(page))
		case ENT:
			url = fmt.Sprintf("%s/orgs/%s/repos?page=%s", s.metadata.githubAPIURL, s.metadata.owner, strconv.Itoa(page))
		default:
			return nil, fmt.Errorf("runnerScope %s not supported", s.metadata.runnerScope)
		}

		body, statusCode, err := s.getGithubRequest(ctx, url, s.metadata, s.httpClient)
		if err != nil {
			return nil, err
		}
		if statusCode == 304 && s.metadata.enableEtags {
			if s.previousRepos != nil {
				return s.previousRepos, nil
			}

			return nil, fmt.Errorf("request for repositories returned status: %d %s but previous repositories is not set", statusCode, http.StatusText(statusCode))
		}

		var repos []Repo

		err = json.Unmarshal(body, &repos)
		if err != nil {
			return nil, err
		}

		for _, repo := range repos {
			repoList = append(repoList, repo.Name)
		}

		// GitHub returned less than 30 repos per page, so consider no repos left
		if len(repos) < 30 {
			break
		}

		page++
	}

	if s.metadata.enableEtags {
		s.previousRepos = repoList
	}

	return repoList, nil
}

func (s *githubRunnerScaler) getRateLimits(header http.Header) RateLimits {
	retryAfterTime := time.Time{}

	remaining, _ := strconv.Atoi(header.Get("X-RateLimit-Remaining"))
	reset, _ := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64)
	resetTime := time.Unix(reset, 0)

	if header.Get("retry-after") != "" {
		retryAfter, _ := strconv.Atoi(header.Get("retry-after"))
		retryAfterTime = time.Now().Add(time.Duration(retryAfter) * time.Second)
	}

	return RateLimits{
		Remaining:      remaining,
		ResetTime:      resetTime,
		RetryAfterTime: retryAfterTime,
	}
}

func (s *githubRunnerScaler) getGithubRequest(ctx context.Context, url string, metadata *githubRunnerMetadata, httpClient *http.Client) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []byte{}, -1, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if metadata.applicationID == nil && metadata.personalAccessToken != nil {
		req.Header.Set("Authorization", "Bearer "+*metadata.personalAccessToken)
	}

	if s.metadata.enableEtags {
		if etag, found := s.etags[url]; found {
			req.Header.Set("If-None-Match", etag)
		}
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

	var rateLimits RateLimits
	if r.Header.Get("X-RateLimit-Remaining") != "" {
		rateLimits := s.getRateLimits(r.Header)
		s.logger.V(0).Info(fmt.Sprintf("GitHub API rate limits: remaining %d, reset at %s, retry after %s", rateLimits.Remaining, rateLimits.ResetTime, rateLimits.RetryAfterTime))

		if s.metadata.enableBackoff {
			s.rateLimits = rateLimits
		}
	}

	if r.StatusCode != 200 {

		if r.StatusCode == 304 && s.metadata.enableEtags {
			s.logger.V(1).Info(fmt.Sprintf("The github rest api for the url: %s returned status %d %s", url, r.StatusCode, http.StatusText(r.StatusCode)))
			return []byte{}, r.StatusCode, nil
		}

		if rateLimits.Remaining == 0 && !rateLimits.ResetTime.IsZero() {
			return []byte{}, r.StatusCode, fmt.Errorf("GitHub API rate limit exceeded, reset time %s", rateLimits.ResetTime)
		}

		if time.Now().Before(rateLimits.RetryAfterTime) && !rateLimits.RetryAfterTime.IsZero() {
			return []byte{}, r.StatusCode, fmt.Errorf("GitHub API rate limit exceeded, retry after %s", rateLimits.RetryAfterTime)
		}

		return []byte{}, r.StatusCode, fmt.Errorf("the GitHub REST API returned error. url: %s status: %d response: %s", url, r.StatusCode, string(b))
	}

	if s.metadata.enableEtags {
		if etag := r.Header.Get("ETag"); etag != "" {
			s.etags[url] = etag
		}
	}

	return b, r.StatusCode, nil
}

func stripDeadRuns(allWfrs []WorkflowRuns) []WorkflowRun {
	var filtered []WorkflowRun
	for _, wfrs := range allWfrs {
		for _, wfr := range wfrs.WorkflowRuns {
			if wfr.Status == "queued" || wfr.Status == "in_progress" {
				filtered = append(filtered, wfr)
			}
		}
	}
	return filtered
}

// getWorkflowRunJobs returns a list of jobs for a given workflow run
func (s *githubRunnerScaler) getWorkflowRunJobs(ctx context.Context, workflowRunID int64, repoName string) ([]Job, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d/jobs?per_page=100", s.metadata.githubAPIURL, s.metadata.owner, repoName, workflowRunID)
	body, statusCode, err := s.getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil {
		return nil, err
	}
	if statusCode == 304 && s.metadata.enableEtags {
		if s.previousJobs[repoName] != nil {
			return s.previousJobs[repoName], nil
		}

		return nil, fmt.Errorf("request for jobs returned status: %d %s but previous jobs is not set", statusCode, http.StatusText(statusCode))
	}

	var jobs Jobs
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		return nil, err
	}

	if s.metadata.enableEtags {
		s.previousJobs[repoName] = jobs.Jobs
	}

	return jobs.Jobs, nil
}

// getWorkflowRuns returns a list of workflow runs for a given repository
func (s *githubRunnerScaler) getWorkflowRuns(ctx context.Context, repoName string, status string) (*WorkflowRuns, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs?status=%s&per_page=100", s.metadata.githubAPIURL, s.metadata.owner, repoName, status)
	body, statusCode, err := s.getGithubRequest(ctx, url, s.metadata, s.httpClient)
	if err != nil && statusCode == 404 {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if statusCode == 304 && s.metadata.enableEtags {
		if s.previousWfrs[repoName][status] != nil {
			return s.previousWfrs[repoName][status], nil
		}

		return nil, fmt.Errorf("request for workflow runs returned status: %d %s but previous workflow runs is not set. Repo: %s, Status: %s", statusCode, http.StatusText(statusCode), repoName, status)
	}

	var wfrs WorkflowRuns
	err = json.Unmarshal(body, &wfrs)
	if err != nil {
		return nil, err
	}

	if s.metadata.enableEtags {
		if _, repoFound := s.previousWfrs[repoName]; !repoFound {
			s.previousWfrs[repoName] = map[string]*WorkflowRuns{status: &wfrs}
		} else {
			s.previousWfrs[repoName][status] = &wfrs
		}
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
func canRunnerMatchLabels(jobLabels []string, runnerLabels []string, noDefaultLabels bool) bool {
	allLabels := runnerLabels
	if !noDefaultLabels {
		allLabels = append(allLabels, reservedLabels...)
	}
	for _, jobLabel := range jobLabels {
		if !contains(allLabels, jobLabel) {
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
		wfrsQueued, err := s.getWorkflowRuns(ctx, repo, "queued")
		if err != nil {
			return -1, err
		}
		if wfrsQueued != nil {
			allWfrs = append(allWfrs, *wfrsQueued)
		}
		wfrsInProgress, err := s.getWorkflowRuns(ctx, repo, "in_progress")
		if err != nil {
			return -1, err
		}
		if wfrsInProgress != nil {
			allWfrs = append(allWfrs, *wfrsInProgress)
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
			if (job.Status == "queued" || job.Status == "in_progress") && canRunnerMatchLabels(job.Labels, s.metadata.labels, s.metadata.noDefaultLabels) {
				queueCount++
			}
		}
	}

	return queueCount, nil
}

func (s *githubRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	if s.metadata.enableBackoff {
		if s.rateLimits.Remaining == 0 && !s.rateLimits.ResetTime.IsZero() && time.Now().Before(s.rateLimits.ResetTime) {
			reset := time.Until(s.rateLimits.ResetTime)
			s.logger.V(0).Info(fmt.Sprintf("Rate limit exceeded, resets at %s, waiting for %s", s.rateLimits.ResetTime, reset))

			// Use context-aware delay
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err() // Return if the context is canceled
			case <-time.After(reset):
				// Wait for reset time, then proceed
			}
		}

		if !s.rateLimits.RetryAfterTime.IsZero() && time.Now().Before(s.rateLimits.RetryAfterTime) {
			retry := time.Until(s.rateLimits.RetryAfterTime)
			s.logger.V(0).Info(fmt.Sprintf("Rate limit exceeded, retry after %s, waiting for %s", s.rateLimits.ResetTime, retry))

			// Use context-aware delay
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err() // Return if the context is canceled
			case <-time.After(retry):
				// Wait for retry time, then proceed
			}
		}

	}

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
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("github-runner-%s", s.metadata.owner))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetWorkflowQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *githubRunnerScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

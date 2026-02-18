package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func writeJSON(w http.ResponseWriter, body string) {
	io.Copy(w, strings.NewReader(body)) //nolint:errcheck
}

var testGitLabJobsPendingResponse = `[{"id":1001,"status":"pending","tag_list":["k8s","docker"]},{"id":1002,"status":"pending","tag_list":["k8s"]}]`
var testGitLabJobsPendingSingleResponse = `[{"id":1001,"status":"pending","tag_list":["k8s","docker"]}]`
var testGitLabJobsEmptyResponse = `[]`
var testGitLabJobsUntaggedResponse = `[{"id":1003,"status":"pending","tag_list":[]}]`
var testGitLabJobsMixedTagsResponse = `[{"id":1001,"status":"pending","tag_list":["k8s","docker"]},{"id":1002,"status":"pending","tag_list":[]},{"id":1003,"status":"pending","tag_list":["gpu"]}]`

var testGitLabGroupProjectsResponse = `[{"id":100},{"id":200},{"id":300}]`
var testGitLabGroupProjectsSingleResponse = `[{"id":100}]`

type parseGitLabRunnerMetadataTestData struct {
	testName      string
	metadata      map[string]string
	authParams    map[string]string
	resolvedEnv   map[string]string
	isError       bool
	expectedError string
}

var testGitLabRunnerAuthParams = map[string]string{
	"personalAccessToken": "glpat-xxxxxxxxxxxxxxxxxxxx",
}

var testGitLabRunnerResolvedEnv = map[string]string{
	"GITLAB_API_URL": "https://gitlab.example.com",
	"ACCESS_TOKEN":   "glpat-envtoken",
	"PROJECT_ID":     "54321",
	"GROUP_ID":       "98765",
}

var testGitLabRunnerMetadata = []parseGitLabRunnerMetadataTestData{
	{
		"empty metadata",
		map[string]string{},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: one of projectID or groupID must be provided",
	},
	{
		"valid with projectID",
		map[string]string{"projectID": "12345"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with groupID",
		map[string]string{"groupID": "67890"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with custom API URL",
		map[string]string{"gitlabAPIURL": "https://gitlab.example.com", "projectID": "12345"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with custom target queue length",
		map[string]string{"projectID": "12345", "targetQueueLength": "5"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with custom job scopes",
		map[string]string{"projectID": "12345", "jobScopes": "pending,created"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"both projectID and groupID",
		map[string]string{"projectID": "12345", "groupID": "67890"},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: only one of projectID or groupID can be provided, not both",
	},
	{
		"missing personalAccessToken",
		map[string]string{"projectID": "12345"},
		map[string]string{},
		nil,
		true,
		"error parsing gitlab runner metadata: missing required parameter \"personalAccessToken\" in [authParams resolvedEnv]",
	},
	{
		"invalid targetQueueLength",
		map[string]string{"projectID": "12345", "targetQueueLength": "abc"},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: unable to set param \"targetQueueLength\" value \"abc\": unable to unmarshal to field type int64: invalid character 'a' looking for beginning of value\ntargetQueueLength must be at least 1",
	},
	{
		"valid with includeSubgroups false",
		map[string]string{"groupID": "67890", "includeSubgroups": "false"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"trailing slash on API URL",
		map[string]string{"gitlabAPIURL": "https://gitlab.example.com/", "projectID": "12345"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"empty projectID",
		map[string]string{"projectID": ""},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: one of projectID or groupID must be provided",
	},
	{
		"empty groupID",
		map[string]string{"groupID": ""},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: one of projectID or groupID must be provided",
	},
	{
		"valid with tagList",
		map[string]string{"projectID": "12345", "tagList": "k8s,docker"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with runUntagged",
		map[string]string{"projectID": "12345", "runUntagged": "true"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"valid with tagList and runUntagged",
		map[string]string{"projectID": "12345", "tagList": "k8s", "runUntagged": "true"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
	{
		"formed from resolved env",
		map[string]string{"projectIDFromEnv": "PROJECT_ID"},
		testGitLabRunnerAuthParams,
		testGitLabRunnerResolvedEnv,
		false,
		"",
	},
	{
		"API URL from resolved env",
		map[string]string{"gitlabAPIURLFromEnv": "GITLAB_API_URL", "projectID": "12345"},
		testGitLabRunnerAuthParams,
		testGitLabRunnerResolvedEnv,
		false,
		"",
	},
	{
		"missing env var reference",
		map[string]string{"projectIDFromEnv": "NONEXISTENT"},
		testGitLabRunnerAuthParams,
		testGitLabRunnerResolvedEnv,
		true,
		"error parsing gitlab runner metadata: one of projectID or groupID must be provided",
	},
	{
		"targetQueueLength zero",
		map[string]string{"projectID": "12345", "targetQueueLength": "0"},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: targetQueueLength must be at least 1",
	},
	{
		"targetQueueLength negative",
		map[string]string{"projectID": "12345", "targetQueueLength": "-1"},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: targetQueueLength must be at least 1",
	},
	{
		"activationTargetQueueLength negative",
		map[string]string{"projectID": "12345", "activationTargetQueueLength": "-1"},
		testGitLabRunnerAuthParams,
		nil,
		true,
		"error parsing gitlab runner metadata: activationTargetQueueLength must be at least 0",
	},
	{
		"valid with unsafeSsl",
		map[string]string{"projectID": "12345", "unsafeSsl": "true"},
		testGitLabRunnerAuthParams,
		nil,
		false,
		"",
	},
}

func TestGitLabRunnerParseMetadata(t *testing.T) {
	for _, testData := range testGitLabRunnerMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			_, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
				ResolvedEnv:     testData.resolvedEnv,
			})

			if testData.isError && err == nil {
				t.Fatal("expected error but got none")
			}
			if testData.isError && err != nil && err.Error() != testData.expectedError {
				t.Fatalf("expected error:\n%s\nbut got:\n%s", testData.expectedError, err.Error())
			}
			if !testData.isError && err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}
		})
	}
}

func TestGitLabRunnerParseMetadata_Defaults(t *testing.T) {
	meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"projectID": "12345"},
		AuthParams:      testGitLabRunnerAuthParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if meta.GitLabAPIURL != "https://gitlab.com" {
		t.Errorf("expected default gitlabAPIURL 'https://gitlab.com', got '%s'", meta.GitLabAPIURL)
	}
	if meta.TargetQueueLength != 1 {
		t.Errorf("expected default targetQueueLength 1, got %d", meta.TargetQueueLength)
	}
	if meta.JobScopes != "pending" {
		t.Errorf("expected default jobScopes 'pending', got '%s'", meta.JobScopes)
	}
	if meta.ActivationTargetQueueLength != 0 {
		t.Errorf("expected default activationTargetQueueLength 0, got %d", meta.ActivationTargetQueueLength)
	}
	if !meta.IncludeSubgroups {
		t.Error("expected default includeSubgroups to be true")
	}
	if meta.RunUntagged {
		t.Error("expected default runUntagged to be false")
	}
	if len(meta.TagList) != 0 {
		t.Errorf("expected default tagList to be empty, got %v", meta.TagList)
	}
	if meta.UnsafeSsl {
		t.Error("expected default unsafeSsl to be false")
	}
}

func TestGitLabRunnerParseMetadata_TrailingSlashTrimmed(t *testing.T) {
	meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"gitlabAPIURL": "https://gitlab.example.com/", "projectID": "12345"},
		AuthParams:      testGitLabRunnerAuthParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if meta.GitLabAPIURL != "https://gitlab.example.com" {
		t.Errorf("expected trailing slash trimmed, got '%s'", meta.GitLabAPIURL)
	}
}

func TestGitLabRunnerParseMetadata_TagListParsed(t *testing.T) {
	meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"projectID": "12345", "tagList": "k8s,docker,linux"},
		AuthParams:      testGitLabRunnerAuthParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(meta.TagList) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(meta.TagList), meta.TagList)
	}
	if meta.TagList[0] != "k8s" || meta.TagList[1] != "docker" || meta.TagList[2] != "linux" {
		t.Errorf("unexpected tag list: %v", meta.TagList)
	}
}

func TestGitLabRunnerParseMetadata_ResolvedEnv(t *testing.T) {
	meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"projectIDFromEnv": "PROJECT_ID"},
		AuthParams:      testGitLabRunnerAuthParams,
		ResolvedEnv:     testGitLabRunnerResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if meta.ProjectID != "54321" {
		t.Errorf("expected projectID '54321' from resolved env, got '%s'", meta.ProjectID)
	}
}

func TestNewGitLabRunnerScaler(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"projectID": "12345"},
		AuthParams:      testGitLabRunnerAuthParams,
	}

	scaler, err := NewGitLabRunnerScaler(config)
	if err != nil {
		t.Fatalf("unexpected error creating scaler: %s", err)
	}

	if scaler == nil {
		t.Fatal("expected non-nil scaler")
	}

	err = scaler.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error closing scaler: %s", err)
	}
}

func TestNewGitLabRunnerScaler_InvalidMetadata(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{},
		AuthParams:      testGitLabRunnerAuthParams,
	}

	_, err := NewGitLabRunnerScaler(config)
	if err == nil {
		t.Fatal("expected error for invalid metadata")
	}
}

func getGitLabTestMetadata(url string) *gitlabRunnerMetadata {
	return &gitlabRunnerMetadata{
		GitLabAPIURL:                url,
		PersonalAccessToken:         "glpat-test",
		TargetQueueLength:           1,
		ActivationTargetQueueLength: 0,
		JobScopes:                   "pending",
		IncludeSubgroups:            true,
	}
}

func gitlabAPIStubHandler(projectJobsResponse string, groupProjectsResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, `{"message":"401 Unauthorized"}`)
			return
		}

		if strings.Contains(r.URL.Path, "/groups/") && strings.HasSuffix(r.URL.Path, "/projects") {
			writeJSON(w, groupProjectsResponse)
			return
		}

		if strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/jobs") {
			jobCount := strings.Count(projectJobsResponse, `"id"`)
			w.Header().Set("x-total", fmt.Sprintf("%d", jobCount))
			writeJSON(w, projectJobsResponse)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, `{"message":"404 Not Found"}`)
	}))
}

func gitlabAPIStubHandlerWithXTotal(xTotal string, projectJobsResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/jobs") {
			if xTotal != "" {
				w.Header().Set("x-total", xTotal)
			}
			writeJSON(w, projectJobsResponse)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func gitlabAPIStubHandler404() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, `{"message":"404 Not Found"}`)
	}))
}

func gitlabAPIStubHandler500() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, `{"message":"500 Internal Server Error"}`)
	}))
}

func gitlabAPIStubHandlerRateLimited() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, `{"message":"429 Too Many Requests"}`)
	}))
}

func gitlabAPIStubHandlerRateLimitExhausted() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit-Remaining", "0")
		w.Header().Set("RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, `{"message":"403 Forbidden"}`)
	}))
}

func gitlabAPIStubHandlerMultiPage(projectsPage1, projectsPage2, jobsResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.HasSuffix(r.URL.Path, "/projects") {
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				writeJSON(w, projectsPage1)
			} else {
				writeJSON(w, projectsPage2)
			}
			return
		}

		if strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/jobs") {
			jobCount := strings.Count(jobsResponse, `"id"`)
			w.Header().Set("x-total", fmt.Sprintf("%d", jobCount))
			writeJSON(w, jobsResponse)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func gitlabAPIStubHandlerMultiScope() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/jobs") {
			scope := r.URL.Query().Get("scope[]")
			switch scope {
			case "pending":
				w.Header().Set("x-total", "2")
				writeJSON(w, testGitLabJobsPendingResponse)
			case "created":
				w.Header().Set("x-total", "1")
				writeJSON(w, testGitLabJobsPendingSingleResponse)
			default:
				w.Header().Set("x-total", "0")
				writeJSON(w, testGitLabJobsEmptyResponse)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestGitLabRunnerScaler_QueueLength_SingleProject(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 2 {
		t.Fatalf("expected queue length 2, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_SingleProject_Empty(t *testing.T) {
	apiStub := gitlabAPIStubHandlerWithXTotal("0", testGitLabJobsEmptyResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_GroupScope(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, testGitLabGroupProjectsSingleResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 2 {
		t.Fatalf("expected queue length 2, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_GroupScope_MultipleProjects(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingSingleResponse, testGitLabGroupProjectsResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// 3 projects * 1 job each = 3
	if queueLen != 3 {
		t.Fatalf("expected queue length 3, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_MultipleScopes(t *testing.T) {
	apiStub := gitlabAPIStubHandlerMultiScope()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.JobScopes = "pending,created"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// 2 pending + 1 created = 3
	if queueLen != 3 {
		t.Fatalf("expected queue length 3, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_XTotalHeader(t *testing.T) {
	apiStub := gitlabAPIStubHandlerWithXTotal("42", testGitLabJobsEmptyResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 42 {
		t.Fatalf("expected queue length 42 from x-total header, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_NoXTotalHeader_FallsBackToBodyParsing(t *testing.T) {
	apiStub := gitlabAPIStubHandlerWithXTotal("", testGitLabJobsPendingResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 2 {
		t.Fatalf("expected queue length 2 from body parsing, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_QueueLength_XTotalIgnoredWithTagFiltering(t *testing.T) {
	apiStub := gitlabAPIStubHandlerWithXTotal("42", testGitLabJobsPendingResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"k8s", "docker"}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Only job 1001 has tags ["k8s","docker"] which are all in runner tags; job 1002 has ["k8s"] which is also a subset
	if queueLen != 2 {
		t.Fatalf("expected queue length 2 with tag filtering (not 42 from x-total), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_404_ProjectNotFound(t *testing.T) {
	apiStub := gitlabAPIStubHandler404()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "99999"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("expected 404 to be handled gracefully, got error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0 for 404 project, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_500_ServerError(t *testing.T) {
	apiStub := gitlabAPIStubHandler500()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected error to contain '500', got: %s", err.Error())
	}
}

func TestGitLabRunnerScaler_RateLimited_429(t *testing.T) {
	apiStub := gitlabAPIStubHandlerRateLimited()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "30") {
		t.Fatalf("expected retry-after value in error, got: %s", err.Error())
	}
}

func TestGitLabRunnerScaler_RateLimited_RateLimitExhausted(t *testing.T) {
	apiStub := gitlabAPIStubHandlerRateLimitExhausted()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for rate limit exhausted response")
	}

	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit error, got: %s", err.Error())
	}
}

func TestGitLabRunnerScaler_BadConnection(t *testing.T) {
	// Start a server and immediately close it to get a guaranteed-unused address
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closedServer.URL
	closedServer.Close()

	meta := getGitLabTestMetadata(closedURL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for bad connection")
	}
}

func TestGitLabRunnerScaler_BadURL(t *testing.T) {
	meta := getGitLabTestMetadata(string([]byte{199, 199, 199}))
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func TestGitLabRunnerScaler_GroupScope_404OnGroupProjects(t *testing.T) {
	apiStub := gitlabAPIStubHandler404()
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "99999"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error when group projects endpoint returns 404")
	}
}

func TestGitLabRunnerScaler_GroupScope_Pagination(t *testing.T) {
	page1Projects := generateGitLabProjects(gitlabDefaultPerPage)
	page2Projects := `[{"id":999}]`

	apiStub := gitlabAPIStubHandlerMultiPage(page1Projects, page2Projects, testGitLabJobsEmptyResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_GroupScope_IncludeSubgroups(t *testing.T) {
	var capturedURL string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.HasSuffix(r.URL.Path, "/projects") {
			capturedURL = r.URL.String()
			writeJSON(w, testGitLabGroupProjectsSingleResponse)
			return
		}
		if strings.Contains(r.URL.Path, "/jobs") {
			w.Header().Set("x-total", "0")
			writeJSON(w, testGitLabJobsEmptyResponse)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"
	meta.IncludeSubgroups = true

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(capturedURL, "include_subgroups=true") {
		t.Fatalf("expected include_subgroups=true in URL, got: %s", capturedURL)
	}
}

func TestGitLabRunnerScaler_GroupScope_ExcludeSubgroups(t *testing.T) {
	var capturedURL string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.HasSuffix(r.URL.Path, "/projects") {
			capturedURL = r.URL.String()
			writeJSON(w, testGitLabGroupProjectsSingleResponse)
			return
		}
		if strings.Contains(r.URL.Path, "/jobs") {
			w.Header().Set("x-total", "0")
			writeJSON(w, testGitLabJobsEmptyResponse)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"
	meta.IncludeSubgroups = false

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(capturedURL, "include_subgroups=false") {
		t.Fatalf("expected include_subgroups=false in URL, got: %s", capturedURL)
	}
}

func TestGitLabRunnerScaler_AuthHeaderSent(t *testing.T) {
	var capturedToken string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken = r.Header.Get("PRIVATE-TOKEN")
		w.Header().Set("x-total", "0")
		writeJSON(w, testGitLabJobsEmptyResponse)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.PersonalAccessToken = "glpat-mytoken"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if capturedToken != "glpat-mytoken" {
		t.Fatalf("expected PRIVATE-TOKEN 'glpat-mytoken', got '%s'", capturedToken)
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_Active(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TargetQueueLength = 1

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !isActive {
		t.Fatal("expected scaler to be active")
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_Inactive(t *testing.T) {
	apiStub := gitlabAPIStubHandlerWithXTotal("0", testGitLabJobsEmptyResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TargetQueueLength = 1

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if isActive {
		t.Fatal("expected scaler to be inactive when queue is empty")
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_Error(t *testing.T) {
	meta := getGitLabTestMetadata("http://127.0.0.1:9999")
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err == nil {
		t.Fatal("expected error")
	}

	if isActive {
		t.Fatal("expected scaler to be inactive on error")
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_HighActivationThreshold(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.ActivationTargetQueueLength = 10

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if isActive {
		t.Fatal("expected scaler to be inactive when queue (2) <= activation threshold (10)")
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_ActivationBoundary(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.ActivationTargetQueueLength = 2

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if isActive {
		t.Fatal("expected scaler to be inactive when queue (2) == activation threshold (2), uses strict greater-than")
	}
}

func TestGitLabRunnerScaler_GetMetricsAndActivity_ActivationExceeded(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.ActivationTargetQueueLength = 1

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !isActive {
		t.Fatal("expected scaler to be active when queue (2) > activation threshold (1)")
	}
}

func TestGitLabRunnerParseMetadata_ActivationTargetQueueLength(t *testing.T) {
	meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"projectID": "12345", "activationTargetQueueLength": "5"},
		AuthParams:      testGitLabRunnerAuthParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if meta.ActivationTargetQueueLength != 5 {
		t.Errorf("expected activationTargetQueueLength 5, got %d", meta.ActivationTargetQueueLength)
	}
}

type gitlabRunnerMetricIdentifier struct {
	metadata     map[string]string
	triggerIndex int
	name         string
}

var gitlabRunnerMetricIdentifiers = []gitlabRunnerMetricIdentifier{
	{map[string]string{"projectID": "12345"}, 0, "s0-gitlab-runner-12345"},
	{map[string]string{"projectID": "12345"}, 1, "s1-gitlab-runner-12345"},
	{map[string]string{"groupID": "67890"}, 0, "s0-gitlab-runner-67890"},
	{map[string]string{"groupID": "67890"}, 2, "s2-gitlab-runner-67890"},
}

func TestGitLabRunnerGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gitlabRunnerMetricIdentifiers {
		t.Run(fmt.Sprintf("index_%d_%v", testData.triggerIndex, testData.metadata), func(t *testing.T) {
			meta, err := parseGitLabRunnerMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testGitLabRunnerAuthParams,
				TriggerIndex:    testData.triggerIndex,
			})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			scaler := gitlabRunnerScaler{
				metadata:   meta,
				httpClient: http.DefaultClient,
			}

			metricSpec := scaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			if metricName != testData.name {
				t.Errorf("Wrong External metric source name: got '%s', want '%s'", metricName, testData.name)
			}
		})
	}
}

func TestGitLabRunnerScaler_Close(t *testing.T) {
	scaler := gitlabRunnerScaler{
		httpClient: http.DefaultClient,
	}

	err := scaler.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on close: %s", err)
	}
}

func TestGitLabRunnerScaler_Close_NilClient(t *testing.T) {
	scaler := gitlabRunnerScaler{
		httpClient: nil,
	}

	err := scaler.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on close with nil client: %s", err)
	}
}

func TestGitLabRunnerScaler_ScopeParameter(t *testing.T) {
	var capturedScope string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedScope = r.URL.Query().Get("scope[]")
		w.Header().Set("x-total", "0")
		writeJSON(w, testGitLabJobsEmptyResponse)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.JobScopes = "running"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if capturedScope != "running" {
		t.Fatalf("expected scope 'running', got '%s'", capturedScope)
	}
}

func TestGitLabRunnerScaler_GroupScope_EmptyGroup(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsEmptyResponse, testGitLabJobsEmptyResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0 for empty group, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_InvalidJSONResponse(t *testing.T) {
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `not valid json`)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGitLabRunnerScaler_GroupScope_InvalidJSONProjects(t *testing.T) {
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.HasSuffix(r.URL.Path, "/projects") {
			writeJSON(w, `not valid json`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON projects response")
	}
}

func TestGitLabRunnerScaler_URLConstruction_Project(t *testing.T) {
	var capturedURL string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("x-total", "0")
		writeJSON(w, testGitLabJobsEmptyResponse)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.JobScopes = "pending"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(capturedURL, "/api/v4/projects/12345/jobs") {
		t.Fatalf("expected URL to contain '/api/v4/projects/12345/jobs', got: %s", capturedURL)
	}
	if !strings.Contains(capturedURL, "scope[]=pending") {
		t.Fatalf("expected URL to contain 'scope[]=pending', got: %s", capturedURL)
	}
}

func TestGitLabRunnerScaler_URLConstruction_Group(t *testing.T) {
	var capturedGroupURL string
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") {
			capturedGroupURL = r.URL.String()
			writeJSON(w, testGitLabJobsEmptyResponse)
			return
		}
		w.Header().Set("x-total", "0")
		writeJSON(w, testGitLabJobsEmptyResponse)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(capturedGroupURL, "/api/v4/groups/67890/projects") {
		t.Fatalf("expected URL to contain '/api/v4/groups/67890/projects', got: %s", capturedGroupURL)
	}
	if !strings.Contains(capturedGroupURL, "simple=true") {
		t.Fatalf("expected URL to contain 'simple=true', got: %s", capturedGroupURL)
	}
}

// Tag filtering tests

func TestGitLabRunnerScaler_CanRunnerPickUpJob_MatchingTags(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"k8s", "docker"}}
	scaler := gitlabRunnerScaler{metadata: meta}

	if !scaler.canRunnerPickUpJob([]string{"k8s"}) {
		t.Error("expected runner with tags [k8s, docker] to pick up job with tags [k8s]")
	}
	if !scaler.canRunnerPickUpJob([]string{"k8s", "docker"}) {
		t.Error("expected runner with tags [k8s, docker] to pick up job with tags [k8s, docker]")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_NonMatchingTags(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"k8s"}}
	scaler := gitlabRunnerScaler{metadata: meta}

	if scaler.canRunnerPickUpJob([]string{"k8s", "docker"}) {
		t.Error("expected runner with tags [k8s] to NOT pick up job requiring [k8s, docker]")
	}
	if scaler.canRunnerPickUpJob([]string{"gpu"}) {
		t.Error("expected runner with tags [k8s] to NOT pick up job requiring [gpu]")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_CaseInsensitive(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"K8S", "Docker"}}
	scaler := gitlabRunnerScaler{metadata: meta}

	if !scaler.canRunnerPickUpJob([]string{"k8s", "docker"}) {
		t.Error("expected case-insensitive tag matching")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_UntaggedJob_RunUntaggedTrue(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"k8s"}, RunUntagged: true}
	scaler := gitlabRunnerScaler{metadata: meta}

	if !scaler.canRunnerPickUpJob([]string{}) {
		t.Error("expected runner with runUntagged=true to pick up untagged job")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_UntaggedJob_RunUntaggedFalse(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"k8s"}, RunUntagged: false}
	scaler := gitlabRunnerScaler{metadata: meta}

	if scaler.canRunnerPickUpJob([]string{}) {
		t.Error("expected runner with runUntagged=false and tagList set to NOT pick up untagged job")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_RunUntaggedOnly(t *testing.T) {
	meta := &gitlabRunnerMetadata{RunUntagged: true}
	scaler := gitlabRunnerScaler{metadata: meta}

	if !scaler.canRunnerPickUpJob([]string{}) {
		t.Error("expected runner with only runUntagged=true to pick up untagged job")
	}
	if scaler.canRunnerPickUpJob([]string{"k8s"}) {
		t.Error("expected runner with only runUntagged=true (no tags) to NOT pick up tagged job")
	}
}

func TestGitLabRunnerScaler_CanRunnerPickUpJob_ExtraRunnerTags(t *testing.T) {
	meta := &gitlabRunnerMetadata{TagList: []string{"k8s", "docker", "linux", "amd64"}}
	scaler := gitlabRunnerScaler{metadata: meta}

	if !scaler.canRunnerPickUpJob([]string{"k8s", "docker"}) {
		t.Error("expected runner with extra tags to pick up job requiring subset of tags")
	}
}

func TestGitLabRunnerScaler_TagFiltering_MatchingSubset(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"k8s", "docker"}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 2 {
		t.Fatalf("expected queue length 2 (both jobs match), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_PartialMatch(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsMixedTagsResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"k8s", "docker"}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 1 {
		t.Fatalf("expected queue length 1 (only tagged matching job), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_WithRunUntagged(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsMixedTagsResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"k8s", "docker"}
	meta.RunUntagged = true

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 2 {
		t.Fatalf("expected queue length 2 (tagged match + untagged), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_NoMatchingJobs(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsPendingResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"gpu"}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0 (no matching jobs), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_UntaggedJobsOnly(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsUntaggedResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.RunUntagged = true

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 1 {
		t.Fatalf("expected queue length 1 (untagged job picked up), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_RunUntaggedFalse_UntaggedJobsSkipped(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsUntaggedResponse, "")
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"
	meta.TagList = []string{"k8s"}
	meta.RunUntagged = false

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != 0 {
		t.Fatalf("expected queue length 0 (untagged job skipped when runUntagged=false), got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_TagFiltering_GroupScope(t *testing.T) {
	apiStub := gitlabAPIStubHandler(testGitLabJobsMixedTagsResponse, testGitLabGroupProjectsSingleResponse)
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.GroupID = "67890"
	meta.TagList = []string{"k8s", "docker"}
	meta.RunUntagged = true

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Mixed response: ["k8s","docker"] matches, [] matches (runUntagged), ["gpu"] doesn't match
	if queueLen != 2 {
		t.Fatalf("expected queue length 2, got %d", queueLen)
	}
}

func TestGitLabRunnerScaler_HasTagFiltering(t *testing.T) {
	tests := []struct {
		name     string
		tagList  []string
		runUntag bool
		expected bool
	}{
		{"no filtering", nil, false, false},
		{"empty tags no untagged", []string{}, false, false},
		{"with tags", []string{"k8s"}, false, true},
		{"runUntagged only", nil, true, true},
		{"both", []string{"k8s"}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gitlabRunnerMetadata{TagList: tt.tagList, RunUntagged: tt.runUntag}
			scaler := gitlabRunnerScaler{metadata: meta}

			if scaler.hasTagFiltering() != tt.expected {
				t.Errorf("hasTagFiltering() = %v, want %v", scaler.hasTagFiltering(), tt.expected)
			}
		})
	}
}

func TestGitLabRunnerScaler_JobPagination(t *testing.T) {
	page1Jobs := generateGitLabJobs(gitlabDefaultPerPage, nil)
	page2Jobs := generateGitLabJobs(3, nil)

	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/jobs") {
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				writeJSON(w, page1Jobs)
			} else {
				writeJSON(w, page2Jobs)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	queueLen, err := scaler.GetQueueLength(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if queueLen != int64(gitlabDefaultPerPage+3) {
		t.Fatalf("expected queue length %d, got %d", gitlabDefaultPerPage+3, queueLen)
	}
}

func TestGitLabRunnerScaler_ContextCancellation(t *testing.T) {
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-total", "0")
		writeJSON(w, testGitLabJobsEmptyResponse)
	}))
	defer apiStub.Close()

	meta := getGitLabTestMetadata(apiStub.URL)
	meta.ProjectID = "12345"

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := scaler.GetQueueLength(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func generateGitLabJobs(count int, tags []string) string {
	var jobs []gitlabJob
	for i := 0; i < count; i++ {
		jobs = append(jobs, gitlabJob{ID: int64(i + 1), TagList: tags})
	}

	result, _ := json.Marshal(jobs)

	return string(result)
}

func generateGitLabProjects(count int) string {
	var projects []gitlabProject
	for i := 0; i < count; i++ {
		projects = append(projects, gitlabProject{ID: int64(i + 1)})
	}

	result, _ := json.Marshal(projects)

	return string(result)
}

package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testGiteaRunnerAuthParams = map[string]string{
	"token": "some-token",
}

type parseGiteaRunnerMetadataTestData struct {
	testName      string
	metadata      map[string]string
	authParams    map[string]string
	isError       bool
	expectedError string
	address       string
	targetJobs    int64
}

var testGiteaRunnerMetadata = []parseGiteaRunnerMetadataTestData{
	{"no address given", map[string]string{
		"global": "true",
	}, testGiteaRunnerAuthParams, true, "missing required parameter \"address\"", "", 0},
	{"no token given", map[string]string{
		"address": "https://gitea.example.com", "global": "true",
	}, map[string]string{}, true, "missing required parameter \"token\"", "", 0},
	{"repo without owner", map[string]string{
		"address": "https://gitea.example.com", "repo": "my-repo",
	}, testGiteaRunnerAuthParams, true, "owner must be set when repo is set", "", 0},
	{"zero targetJobs", map[string]string{
		"address": "https://gitea.example.com", "global": "true", "targetJobs": "0",
	}, testGiteaRunnerAuthParams, true, "targetJobs must be a positive number", "", 0},
	{"negative targetJobs", map[string]string{
		"address": "https://gitea.example.com", "global": "true", "targetJobs": "-3",
	}, testGiteaRunnerAuthParams, true, "targetJobs must be a positive number", "", 0},
	{"minimal global", map[string]string{
		"address": "https://gitea.example.com", "global": "true",
	}, testGiteaRunnerAuthParams, false, "", "https://gitea.example.com", 1},
	{"trailing slash is trimmed", map[string]string{
		"address": "https://gitea.example.com/", "global": "true",
	}, testGiteaRunnerAuthParams, false, "", "https://gitea.example.com", 1},
	{"labels are optional", map[string]string{
		"address": "https://gitea.example.com", "global": "true", "labels": "ubuntu-latest,self-hosted",
	}, testGiteaRunnerAuthParams, false, "", "https://gitea.example.com", 1},
	{"explicit targetJobs", map[string]string{
		"address": "https://gitea.example.com", "global": "true", "targetJobs": "4",
	}, testGiteaRunnerAuthParams, false, "", "https://gitea.example.com", 4},
	{"repo scope", map[string]string{
		"address": "https://gitea.example.com", "owner": "someone", "repo": "my-repo",
	}, testGiteaRunnerAuthParams, false, "", "https://gitea.example.com", 1},
}

func TestGiteaRunnerParseMetadata(t *testing.T) {
	for _, testData := range testGiteaRunnerMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			meta, err := parseGiteaRunnerMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			})

			if testData.isError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testData.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testData.address, meta.Address)
			assert.Equal(t, testData.targetJobs, meta.TargetJobs)
		})
	}
}

func TestGiteaRunnerLabelList(t *testing.T) {
	tests := []struct {
		name     string
		labels   string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "ubuntu-latest", []string{"ubuntu-latest"}},
		{"multiple", "ubuntu-latest,self-hosted", []string{"ubuntu-latest", "self-hosted"}},
		{"whitespace is trimmed", " ubuntu-latest , self-hosted ", []string{"ubuntu-latest", "self-hosted"}},
		{"empty entries are dropped", "ubuntu-latest,,self-hosted,", []string{"ubuntu-latest", "self-hosted"}},
		{"only separators", ",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &giteaRunnerMetadata{Labels: tt.labels}
			assert.Equal(t, tt.expected, meta.labelList())
		})
	}
}

func TestGiteaRunnerCanMatchLabels(t *testing.T) {
	tests := []struct {
		name           string
		jobLabels      []string
		runnerLabels   []string
		matchUnlabeled bool
		expected       bool
	}{
		{"exact match", []string{"ubuntu-latest"}, []string{"ubuntu-latest"}, false, true},
		{"runner offers a superset", []string{"ubuntu-latest"}, []string{"ubuntu-latest", "self-hosted"}, false, true},
		{"job needs a label the runner lacks", []string{"windows"}, []string{"ubuntu-latest"}, false, false},
		{"job needs two, runner has one", []string{"ubuntu-latest", "gpu"}, []string{"ubuntu-latest"}, false, false},
		{"unlabeled job matches any runner by default", []string{}, []string{"ubuntu-latest"}, false, true},
		{"unlabeled job with strict matching needs an unlabeled runner", []string{}, []string{"ubuntu-latest"}, true, false},
		{"unlabeled job and unlabeled runner with strict matching", []string{}, []string{}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canGiteaRunnerMatchLabels(tt.jobLabels, tt.runnerLabels, tt.matchUnlabeled)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGiteaRunnerJobsURL(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *giteaRunnerMetadata
		expectedPath string
	}{
		{"global uses the admin endpoint", &giteaRunnerMetadata{Address: "https://g.example.com", Global: true}, "/api/v1/admin/actions/jobs"},
		{"org scope", &giteaRunnerMetadata{Address: "https://g.example.com", Org: "my-org"}, "/api/v1/orgs/my-org/actions/jobs"},
		{"repo scope", &giteaRunnerMetadata{Address: "https://g.example.com", Owner: "someone", Repo: "my-repo"}, "/api/v1/repos/someone/my-repo/actions/jobs"},
		{"user scope is the fallback", &giteaRunnerMetadata{Address: "https://g.example.com"}, "/api/v1/user/actions/jobs"},
		{"repo scope beats org scope", &giteaRunnerMetadata{Address: "https://g.example.com", Org: "my-org", Owner: "someone", Repo: "my-repo"}, "/api/v1/repos/someone/my-repo/actions/jobs"},
		{"org scope beats global", &giteaRunnerMetadata{Address: "https://g.example.com", Global: true, Org: "my-org"}, "/api/v1/orgs/my-org/actions/jobs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &giteaRunnerScaler{metadata: tt.metadata}
			u, err := s.getJobsURL(1, 50)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPath, u.Path)

			// Both statuses must be requested, or the scaler tears down replicas
			// the moment their jobs start running.
			assert.ElementsMatch(t, []string{"queued", "in_progress"}, u.Query()["status"])
			assert.Equal(t, "1", u.Query().Get("page"))
			assert.Equal(t, "50", u.Query().Get("limit"))
		})
	}
}

// newGiteaTestScaler wires a scaler to a test server and returns a pointer to
// the request counter, so tests can assert on API call volume rather than just
// the resulting number.
func newGiteaTestScaler(t *testing.T, meta *giteaRunnerMetadata, handler http.HandlerFunc) (*giteaRunnerScaler, *int) {
	t.Helper()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	meta.Address = srv.URL
	return &giteaRunnerScaler{
		metadata: meta,
		client:   srv.Client(),
		logger:   logr.Discard(),
	}, &requests
}

// writeGiteaJobs encodes a jobs response straight onto the ResponseWriter.
// json.NewEncoder rather than w.Write to match the idiom used by the other
// forge scaler tests.
func writeGiteaJobs(t *testing.T, w http.ResponseWriter, jobs []giteaJob, total int64) {
	t.Helper()
	err := json.NewEncoder(w).Encode(giteaJobsResponse{Jobs: jobs, TotalCount: total})
	require.NoError(t, err)
}

// The headline property of this scaler: with no label filter it answers in a
// SINGLE request by reading total_count, and never walks pages.
func TestGiteaRunnerCountUsesTotalCountWithoutLabels(t *testing.T) {
	s, requests := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, TargetJobs: 1},
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "1", r.URL.Query().Get("limit"), "fast path should request a single row")
			writeGiteaJobs(t, w, []giteaJob{{ID: 1}}, 137)
		})

	count, err := s.getJobCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(137), count, "count should come from total_count, not len(jobs)")
	assert.Equal(t, 1, *requests, "no-label path must cost exactly one request")
}

func TestGiteaRunnerCountFiltersByLabel(t *testing.T) {
	jobs := []giteaJob{
		{ID: 1, Status: "queued", Labels: []string{"ubuntu-latest"}},
		{ID: 2, Status: "in_progress", Labels: []string{"ubuntu-latest"}},
		{ID: 3, Status: "queued", Labels: []string{"windows"}},
		{ID: 4, Status: "queued", Labels: []string{"ubuntu-latest", "gpu"}},
	}

	s, requests := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, Labels: "ubuntu-latest", TargetJobs: 1},
		func(w http.ResponseWriter, _ *http.Request) {
			writeGiteaJobs(t, w, jobs, int64(len(jobs)))
		})

	count, err := s.getJobCount(context.Background())
	require.NoError(t, err)
	// Jobs 1 and 2 match. Job 3 wants windows; job 4 also wants gpu.
	assert.Equal(t, int64(2), count)
	assert.Equal(t, 1, *requests, "a short page terminates pagination immediately")
}

func TestGiteaRunnerCountPaginates(t *testing.T) {
	full := make([]giteaJob, giteaPageSize)
	for i := range full {
		full[i] = giteaJob{ID: int64(i), Status: "queued", Labels: []string{"ubuntu-latest"}}
	}
	lastPage := []giteaJob{{ID: 999, Status: "queued", Labels: []string{"ubuntu-latest"}}}

	s, requests := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, Labels: "ubuntu-latest", TargetJobs: 1},
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "1" {
				writeGiteaJobs(t, w, full, 51)
				return
			}
			writeGiteaJobs(t, w, lastPage, 51)
		})

	count, err := s.getJobCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(giteaPageSize+1), count)
	assert.Equal(t, 2, *requests)
}

// A server that always returns a full page must not loop forever.
func TestGiteaRunnerCountRespectsPageCap(t *testing.T) {
	full := make([]giteaJob, giteaPageSize)
	for i := range full {
		full[i] = giteaJob{ID: int64(i), Status: "queued", Labels: []string{"ubuntu-latest"}}
	}

	s, requests := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, Labels: "ubuntu-latest", TargetJobs: 1},
		func(w http.ResponseWriter, _ *http.Request) {
			writeGiteaJobs(t, w, full, 100000)
		})

	count, err := s.getJobCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(giteaMaxPages*giteaPageSize), count)
	assert.Equal(t, giteaMaxPages, *requests, "pagination must stop at the page cap")
}

func TestGiteaRunnerCountErrors(t *testing.T) {
	t.Run("non-200 is an error", func(t *testing.T) {
		s, _ := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, TargetJobs: 1},
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			})

		_, err := s.getJobCount(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("malformed JSON is an error", func(t *testing.T) {
		s, _ := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, TargetJobs: 1},
			func(w http.ResponseWriter, _ *http.Request) {
				require.NoError(t, json.NewEncoder(w).Encode("this is not an object"))
			})

		_, err := s.getJobCount(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshalling")
	})
}

func TestGiteaRunnerSendsTokenHeader(t *testing.T) {
	s, _ := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, Token: "s3cret", TargetJobs: 1},
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "token s3cret", r.Header.Get("Authorization"))
			writeGiteaJobs(t, w, nil, 0)
		})

	_, err := s.getJobCount(context.Background())
	require.NoError(t, err)
}

func TestGiteaRunnerGetMetricsAndActivity(t *testing.T) {
	tests := []struct {
		name           string
		total          int64
		expectedValue  int64
		expectedActive bool
	}{
		{"idle", 0, 0, false},
		{"one queued job activates", 1, 1, true},
		{"many queued jobs", 42, 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newGiteaTestScaler(t, &giteaRunnerMetadata{Global: true, TargetJobs: 1},
				func(w http.ResponseWriter, _ *http.Request) {
					writeGiteaJobs(t, w, nil, tt.total)
				})

			metrics, active, err := s.GetMetricsAndActivity(context.Background(), "gitea")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedActive, active)
			require.Len(t, metrics, 1)
			assert.Equal(t, tt.expectedValue, metrics[0].Value.Value())
		})
	}
}

func TestGiteaRunnerGetMetricSpecForScaling(t *testing.T) {
	meta, err := parseGiteaRunnerMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"address": "https://gitea.example.com", "global": "true", "targetJobs": "2",
		},
		AuthParams:   testGiteaRunnerAuthParams,
		TriggerIndex: 3,
	})
	require.NoError(t, err)

	s := &giteaRunnerScaler{metadata: meta, metricType: "AverageValue", logger: logr.Discard()}
	spec := s.GetMetricSpecForScaling(context.Background())

	require.Len(t, spec, 1)
	assert.Equal(t, fmt.Sprintf("s%d-gitea", 3), spec[0].External.Metric.Name)
	assert.Equal(t, int64(2), spec[0].External.Target.AverageValue.Value())
}

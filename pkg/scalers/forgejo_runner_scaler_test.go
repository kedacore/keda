package scalers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/forgejo"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseForgejoRunnerMetadataTestData struct {
	testName      string
	metadata      map[string]string
	isError       bool
	expectedError string
	global        bool
}

var testForgejoRunnerMetadata = []parseForgejoRunnerMetadataTestData{
	{"no address given", map[string]string{
		"token": "some-token", "name": "some-name", "labels": "some-label", "metric_path": "path",
	}, true, "no address given", false},
	{"no token given", map[string]string{
		"address": "https://api.github.com", "name": "some-name", "labels": "some-label", "metric_path": "path",
	}, true, "no token given", false},
	{"no label given", map[string]string{
		"address": "https://api.github.com", "token": "some-token", "name": "some-name", "metric_path": "path",
	}, true, "no labels given", false},
	{"no name given use default", map[string]string{
		"address": "https://api.github.com", "token": "some-token", "labels": "some-label", "metric_path": "path",
	}, false, "", false},
	{"no metric path given use default", map[string]string{
		"address": "https://api.github.com", "token": "some-token", "name": "some-name", "labels": "some-label",
	}, false, "", false},
	// properly formed
	{"properly formed", map[string]string{
		"address":     "https://api.github.com",
		"token":       "some-token",
		"name":        "some-name",
		"labels":      "some-labels",
		"metric_path": "path",
	}, false, "", false},
	{"properly formed with global", map[string]string{
		"address":     "https://api.github.com",
		"token":       "some-token",
		"name":        "some-name",
		"labels":      "some-labels",
		"global":      "true",
		"metric_path": "path",
	}, false, "", true},
}

func TestForgejoRunnerParseMetadata(t *testing.T) {
	for _, testData := range testForgejoRunnerMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			got, err := parseForgejoRunnerMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testAuthParams})

			if testData.isError && err == nil {
				t.Fatal("expected error but got none")
			}
			if testData.isError && err != nil && err.Error() != testData.expectedError {
				t.Fatal("expected error " + testData.expectedError + " but got error " + err.Error())
			}
			if !testData.isError && err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			if got != nil {
				assert.Equal(t, testData.global, got.RunnerMeta.Global)
			}
		})
	}
}

func Test_forgejoRunnerScaler_getGlobalRunnerJobsUrl(t *testing.T) {
	type fields struct {
		metadata *ForgejoRunnerConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "a global value should return an admin job url",
			fields: fields{
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Global:  true,
						Labels:  "ubuntu-latest",
						Address: "http://localhost",
					},
				},
			},
			want:    "http://localhost/api/v1/admin/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "a false global value should return user jobs url",
			fields: fields{
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Global:  false,
						Labels:  "ubuntu-latest",
						Address: "http://localhost",
					},
				},
			},
			want:    "http://localhost/api/v1/user/actions/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "when an org parameter is present should return the org jobs url",
			fields: fields{
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Labels:  "ubuntu-latest",
						Address: "http://localhost",
						Org:     "my-org",
					},
				},
			},
			want:    "http://localhost/api/v1/orgs/my-org/actions/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "when an repo and owner parameters is present should return the repo jobs url",
			fields: fields{
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Labels:  "ubuntu-latest",
						Address: "http://localhost",
						Owner:   "owner",
						Repo:    "my-repo",
					},
				},
			},
			want:    "http://localhost/api/v1/repos/owner/my-repo/actions/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &forgejoRunnerScaler{
				metadata: tt.fields.metadata,
			}
			got, err := s.getRunnerJobURL()
			if !tt.wantErr(t, err, "getRunnerJobURL()") {
				return
			}
			assert.Equalf(t, tt.want, got.String(), "getRunnerJobURL()")
		})
	}
}

func Test_forgejoRunnerScaler_getJobsList(t *testing.T) {
	jobsList := forgejo.JobsListResponse{
		Jobs: []forgejo.Job{
			{ID: 1},
			{ID: 2},
		},
	}
	repoJobList := forgejo.JobsListResponse{
		Jobs: []forgejo.Job{
			{ID: 3},
			{ID: 4},
		},
	}
	forgejoServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/api/v1/repos/owner/my-repo/actions/runners/jobs" {
			body, _ := json.Marshal(repoJobList)
			_, _ = rw.Write(body)
			return
		}
		body, _ := json.Marshal(jobsList)
		_, _ = rw.Write(body)
	}))
	// Close the server when test finishes
	defer forgejoServer.Close()

	type fields struct {
		metricType v2.MetricTargetType
		metadata   *ForgejoRunnerConfig
		client     *http.Client
		logger     logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    forgejo.JobsListResponse
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "a global metadata should return a valid list of global jobs",
			fields: fields{
				metricType: v2.AverageValueMetricType,
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Labels:  "ubuntu-latest",
						Token:   "my-token",
						Address: forgejoServer.URL,
						Global:  true,
					},
				},
				client: forgejoServer.Client(),
				logger: logr.Logger{},
			},

			want: forgejo.JobsListResponse{
				Jobs: []forgejo.Job{
					{ID: 1},
					{ID: 2},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "a repo level metadata should return a valid list of repo jobs",
			fields: fields{
				metricType: v2.AverageValueMetricType,
				metadata: &ForgejoRunnerConfig{
					RunnerMeta: forgejoRunnerMetadata{
						Labels:  "ubuntu-latest",
						Token:   "my-token",
						Address: forgejoServer.URL,
						Owner:   "owner",
						Repo:    "my-repo",
					},
				},
				client: forgejoServer.Client(),
				logger: logr.Logger{},
			},

			want: forgejo.JobsListResponse{
				Jobs: []forgejo.Job{
					{ID: 3},
					{ID: 4},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &forgejoRunnerScaler{
				metricType: tt.fields.metricType,
				metadata:   tt.fields.metadata,
				client:     tt.fields.client,
				logger:     tt.fields.logger,
			}
			got, err := s.getJobsList(context.Background())
			if !tt.wantErr(t, err, "getJobsList()") {
				return
			}
			assert.Equalf(t, tt.want, got, "getJobsList()")
		})
	}
}

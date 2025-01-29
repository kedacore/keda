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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseForgejoRunnerMetadataTestData struct {
	testName      string
	metadata      map[string]string
	isError       bool
	expectedError string
	global        bool
	address       string
}

var testForgejoRunnerMetadata = []parseForgejoRunnerMetadataTestData{
	{"no address given", map[string]string{
		"token": "some-token", "name": "some-name", "labels": "some-label",
	}, true, "error parsing forgejo metadata: missing required parameter \"address\" in [triggerMetadata]", false, ""},
	{"no token given", map[string]string{
		"address": "https://code.forgejo.org", "name": "some-name", "labels": "some-label",
	}, true, "error parsing forgejo metadata: missing required parameter \"token\" in [authParams triggerMetadata]", false, ""},
	{"no label given", map[string]string{
		"address": "https://code.forgejo.org", "token": "some-token", "name": "some-name",
	}, true, "error parsing forgejo metadata: missing required parameter \"labels\" in [triggerMetadata]", false, ""},
	{"no name given use default", map[string]string{
		"address": "https://code.forgejo.org", "token": "some-token", "labels": "some-label",
	}, false, "", false, "https://code.forgejo.org"},
	{"no metric path given use default", map[string]string{
		"address": "https://code.forgejo.org", "token": "some-token", "name": "some-name", "labels": "some-label",
	}, false, "", false, "https://code.forgejo.org"},
	// properly formed
	{"properly formed", map[string]string{
		"address": "https://code.forgejo.org",
		"token":   "some-token",
		"name":    "some-name",
		"labels":  "some-labels",
	}, false, "", false, "https://code.forgejo.org"},
	{"properly formed with global", map[string]string{
		"address": "https://code.forgejo.org",
		"token":   "some-token",
		"name":    "some-name",
		"labels":  "some-labels",
		"global":  "true",
	}, false, "", true, "https://code.forgejo.org"},
	{"properly formed with address with an extra slash", map[string]string{
		"address": "https://code.forgejo.org/",
		"token":   "some-token",
		"name":    "some-name",
		"labels":  "some-labels",
		"global":  "true",
	}, false, "", true, "https://code.forgejo.org"},
	{"properly formed with global", map[string]string{
		"address": "https://code.forgejo.org",
		"token":   "some-token",
		"name":    "some-name",
		"labels":  "some-labels",
		"owner":   "owner",
		"repo":    "my-repo",
	}, false, "", false, "https://code.forgejo.org"},
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
				assert.Equal(t, testData.global, got.Global)
				assert.Equal(t, testData.address, got.Address)
			}
		})
	}
}

func Test_forgejoRunnerScaler_getGlobalRunnerJobsUrl(t *testing.T) {
	type fields struct {
		metadata *forgejoRunnerMetadata
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
				metadata: &forgejoRunnerMetadata{
					Global:  true,
					Labels:  "ubuntu-latest",
					Address: "http://localhost",
				},
			},
			want:    "http://localhost/api/v1/admin/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "a false global value should return user jobs url",
			fields: fields{
				metadata: &forgejoRunnerMetadata{
					Global:  false,
					Labels:  "ubuntu-latest",
					Address: "http://localhost",
				},
			},
			want:    "http://localhost/api/v1/user/actions/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "when an org parameter is present should return the org jobs url",
			fields: fields{
				metadata: &forgejoRunnerMetadata{
					Labels:  "ubuntu-latest",
					Address: "http://localhost",
					Org:     "my-org",
				},
			},
			want:    "http://localhost/api/v1/orgs/my-org/actions/runners/jobs?labels=ubuntu-latest",
			wantErr: assert.NoError,
		},
		{
			name: "when an repo and owner parameters is present should return the repo jobs url",
			fields: fields{
				metadata: &forgejoRunnerMetadata{
					Labels:  "ubuntu-latest",
					Address: "http://localhost",
					Owner:   "owner",
					Repo:    "my-repo",
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
	jobsList := JobsListResponse{
		Jobs: []ForgejoJob{
			{ID: 1},
			{ID: 2},
		},
	}
	repoJobList := JobsListResponse{
		Jobs: []ForgejoJob{
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
		metadata   *forgejoRunnerMetadata
		client     *http.Client
		logger     logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    JobsListResponse
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "a global metadata should return a valid list of global jobs",
			fields: fields{
				metricType: v2.AverageValueMetricType,
				metadata: &forgejoRunnerMetadata{
					Labels:  "ubuntu-latest",
					Token:   "my-token",
					Address: forgejoServer.URL,
					Global:  true,
				},
				client: forgejoServer.Client(),
				logger: logr.Logger{},
			},

			want: JobsListResponse{
				Jobs: []ForgejoJob{
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
				metadata: &forgejoRunnerMetadata{
					Labels:  "ubuntu-latest",
					Token:   "my-token",
					Address: forgejoServer.URL,
					Owner:   "owner",
					Repo:    "my-repo",
				},
				client: forgejoServer.Client(),
				logger: logr.Logger{},
			},

			want: JobsListResponse{
				Jobs: []ForgejoJob{
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

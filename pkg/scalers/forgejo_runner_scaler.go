package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultForgejoJobsLen = 1

	adminJobMetricPath = "/api/v1/admin/runners/jobs"
	orgJobMetricPath   = "/api/v1/orgs/%s/actions/runners/jobs"
	repoJobMetricPath  = "/api/v1/repos/%s/%s/actions/runners/jobs"
	userJobMetricPath  = "/api/v1/user/actions/runners/jobs"
)

type ForgejoJob struct {
	ID int64 `json:"id"`
	// the repository id
	RepoID int64 `json:"repo_id"`
	// the owner id
	OwnerID int64 `json:"owner_id"`
	// the action run job name
	Name string `json:"name"`
	// the action run job needed ids
	Needs []string `json:"needs"`
	// the action run job labels to run on
	RunsOn []string `json:"runs_on"`
	// the action run job latest task id
	TaskID int64 `json:"task_id"`
	// the action run job status
	Status string `json:"status"`
}

type forgejoRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *forgejoRunnerMetadata
	client     *http.Client
	logger     logr.Logger
}

type forgejoRunnerMetadata struct {
	triggerIndex int

	Token   string `keda:"name=token, order=authParams;triggerMetadata"`
	Address string `keda:"name=address, order=triggerMetadata"`
	Labels  string `keda:"name=labels, order=triggerMetadata"` // comma separated runner labels
	Global  bool   `keda:"name=global, order=triggerMetadata, optional"`
	Owner   string `keda:"name=owner, order=triggerMetadata, optional"`
	Org     string `keda:"name=org, order=triggerMetadata, optional"`
	Repo    string `keda:"name=repo, order=triggerMetadata, optional"`
}

func parseForgejoRunnerMetadata(config *scalersconfig.ScalerConfig) (*forgejoRunnerMetadata, error) {
	meta := &forgejoRunnerMetadata{}
	meta.triggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing forgejo metadata: %w", err)
	}

	if meta.Address[len(meta.Address)-1:] == "/" {
		meta.Address = meta.Address[:len(meta.Address)-1]
	}

	return meta, nil
}

// NewForgejoRunnerScaler creates a new Forgejo Runner Scaler
func NewForgejoRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	c := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseForgejoRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Forgejo Runner metadata: %w", err)
	}

	logger := InitializeLogger(config, "forgejo_runner_scaler")

	return &forgejoRunnerScaler{
		client:     c,
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func (s *forgejoRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	jobList, err := s.getJobsList(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(len(jobList)))

	metric.Value.Add(resource.Quantity{})

	return []external_metrics.ExternalMetricValue{metric}, true, nil
}

func (s *forgejoRunnerScaler) getJobsList(ctx context.Context) ([]ForgejoJob, error) {
	var jobList []ForgejoJob

	uri, err := s.getRunnerJobURL()
	if err != nil {
		return jobList, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return jobList, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.metadata.Token))

	r, err := s.client.Do(req)
	if err != nil {
		return jobList, err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return jobList, err
	}
	_ = r.Body.Close()

	if r.StatusCode != 200 {
		return jobList,
			fmt.Errorf("the Forgejo REST API returned error. url: %s status: %d response: %s",
				s.metadata.Address,
				r.StatusCode,
				string(b),
			)
	}

	err = json.Unmarshal(b, &jobList)
	return jobList, err
}

func (s *forgejoRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("forgejo")),
		},
		Target: GetMetricTarget(s.metricType, defaultForgejoJobsLen),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *forgejoRunnerScaler) getRunnerJobURL() (*url.URL, error) {
	if s.metadata.Owner != "" && s.metadata.Repo != "" {
		return s.getRepoRunnerJobURL()
	}
	if s.metadata.Org != "" {
		return s.getOrgRunnerJobURL()
	}

	if s.metadata.Global {
		return s.getGlobalRunnerJobsURL()
	}

	return s.getUserRunnerJobsURL()
}

func (s *forgejoRunnerScaler) getGlobalRunnerJobsURL() (*url.URL, error) {
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.Address,
			adminJobMetricPath,
			s.metadata.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getUserRunnerJobsURL() (*url.URL, error) {
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.Address,
			userJobMetricPath,
			s.metadata.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getOrgRunnerJobURL() (*url.URL, error) {
	orgJobPath := fmt.Sprintf(orgJobMetricPath, s.metadata.Org)
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.Address,
			orgJobPath,
			s.metadata.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getRepoRunnerJobURL() (*url.URL, error) {
	repoJobPath := fmt.Sprintf(repoJobMetricPath, s.metadata.Owner, s.metadata.Repo)
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.Address,
			repoJobPath,
			s.metadata.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) Close(_ context.Context) error {
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	return nil
}

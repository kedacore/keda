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

	"github.com/kedacore/keda/v2/pkg/scalers/forgejo"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultForgejoJobsLen = 1

	adminJobMetricPath = "/api/v1/admin/runners/jobs"
	orgJobMetricPath   = "/api/v1/orgs/%s/actions/runners/jobs"     // /api/v1/orgs/{org}/actions/runners/jobs
	repoJobMetricPath  = "/api/v1/repos/%s/%s/actions/runners/jobs" // {address}/api/v1/repos/{owner}/{repo}/actions/runners/jobs
	UserJobMetricPath  = "/api/v1/user/actions/runners/jobs"
)

type forgejoRunnerMetadata struct {
	Token      string
	Address    string
	MetricPath string
	Labels     string // comma separated runner labels
	Global     bool
	Owner      string
	Org        string
	Repo       string
}

// ForgejoRunnerConfig represents the overall configuration.
type ForgejoRunnerConfig struct {
	RunnerMeta forgejoRunnerMetadata `yaml:"runner"` // Runner represents the configuration for the runner.
}

func parseForgejoRunnerMetadata(config *scalersconfig.ScalerConfig) (*ForgejoRunnerConfig, error) {
	meta := &ForgejoRunnerConfig{}

	if val, ok := config.TriggerMetadata["token"]; ok && val != "" {
		meta.RunnerMeta.Token = val
	} else {
		return nil, fmt.Errorf("no token given")
	}

	if val, ok := config.TriggerMetadata["address"]; ok && val != "" {
		meta.RunnerMeta.Address = val
	} else {
		return nil, fmt.Errorf("no address given")
	}

	if val, ok := config.TriggerMetadata["labels"]; ok && val != "" {
		meta.RunnerMeta.Labels = val
	} else {
		return nil, fmt.Errorf("no labels given")
	}

	global := false
	if val, ok := config.TriggerMetadata["global"]; ok && val == stringTrue {
		global = true
	}

	meta.RunnerMeta.Global = global

	return meta, nil
}

type forgejoRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *ForgejoRunnerConfig
	client     *http.Client
	logger     logr.Logger
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

	metric := GenerateMetricInMili(metricName, float64(len(jobList.Jobs)))

	metric.Value.Add(resource.Quantity{})

	return []external_metrics.ExternalMetricValue{metric}, true, nil
}

func (s *forgejoRunnerScaler) getJobsList(ctx context.Context) (forgejo.JobsListResponse, error) {
	jobList := forgejo.JobsListResponse{}

	uri, err := s.getRunnerJobURL()
	if err != nil {
		return jobList, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return jobList, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.metadata.RunnerMeta.Token))

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
				s.metadata.RunnerMeta.Address,
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
			Name: GenerateMetricNameWithIndex(
				1,
				kedautil.NormalizeString(fmt.Sprintf("forgejo-runner-%s", s.metadata.RunnerMeta.Address)),
			),
		},
		Target: GetMetricTarget(s.metricType, defaultForgejoJobsLen),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *forgejoRunnerScaler) getRunnerJobURL() (*url.URL, error) {
	if s.metadata.RunnerMeta.Owner != "" && s.metadata.RunnerMeta.Repo != "" {
		return s.getRepoRunnerJobURL()
	}
	if s.metadata.RunnerMeta.Org != "" {
		return s.getOrgRunnerJobURL()
	}

	if s.metadata.RunnerMeta.Global {
		return s.getGlobalRunnerJobsURL()
	}

	return s.getUserRunnerJobsURL()
}

func (s *forgejoRunnerScaler) getGlobalRunnerJobsURL() (*url.URL, error) {
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.RunnerMeta.Address,
			adminJobMetricPath,
			s.metadata.RunnerMeta.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getUserRunnerJobsURL() (*url.URL, error) {
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.RunnerMeta.Address,
			UserJobMetricPath,
			s.metadata.RunnerMeta.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getOrgRunnerJobURL() (*url.URL, error) {
	orgJobPath := fmt.Sprintf(orgJobMetricPath, s.metadata.RunnerMeta.Org)
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.RunnerMeta.Address,
			orgJobPath,
			s.metadata.RunnerMeta.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) getRepoRunnerJobURL() (*url.URL, error) {
	repoJobPath := fmt.Sprintf(repoJobMetricPath, s.metadata.RunnerMeta.Owner, s.metadata.RunnerMeta.Repo)
	return url.Parse(
		fmt.Sprintf(
			"%s%s?labels=%s",
			s.metadata.RunnerMeta.Address,
			repoJobPath,
			s.metadata.RunnerMeta.Labels,
		),
	)
}

func (s *forgejoRunnerScaler) Close(_ context.Context) error {
	return nil
}

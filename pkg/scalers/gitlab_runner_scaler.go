package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetPipelineQueueLength = 1
	defaultGitlabAPIURL              = "https://gitlab.com"
)

type gitlabRunnerScaler struct {
	metricType v2.MetricTargetType
	metadata   *gitlabRunnerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type gitlabRunnerMetadata struct {
	gitlabAPIURL        *url.URL
	personalAccessToken string
	projectID           string

	targetWorkflowQueueLength int64
	triggerIndex              int
}

// NewGitLabRunnerScaler creates a new GitLab Runner Scaler
func NewGitLabRunnerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGitLabRunnerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing GitLab Runner metadata: %w", err)
	}

	return &gitlabRunnerScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "gitlab_runner_scaler"),
	}, nil
}

func parseGitLabRunnerMetadata(config *scalersconfig.ScalerConfig) (*gitlabRunnerMetadata, error) {
	meta := &gitlabRunnerMetadata{}
	meta.targetWorkflowQueueLength = defaultTargetWorkflowQueueLength

	// Get the GitLab API URL
	gitlabAPIURLValue, err := getValueFromMetaOrEnv("gitlabAPIURL", config.TriggerMetadata, config.ResolvedEnv)
	if err != nil || gitlabAPIURLValue == "" {
		gitlabAPIURLValue = defaultGitlabAPIURL
	}

	gitlabAPIURL, err := url.Parse(gitlabAPIURLValue)
	if err != nil {
		return nil, fmt.Errorf("parsing gitlabAPIURL: %w", err)
	}
	meta.gitlabAPIURL = gitlabAPIURL

	// Get the projectID
	projectIDValue, err := getValueFromMetaOrEnv("projectID", config.TriggerMetadata, config.ResolvedEnv)
	if err != nil || projectIDValue == "" {
		return nil, err
	}
	meta.projectID = projectIDValue

	// Get the targetWorkflowQueueLength
	targetWorkflowQueueLength, err := getInt64ValueFromMetaOrEnv("targetWorkflowQueueLength", config)
	if err != nil || targetWorkflowQueueLength == 0 {
		meta.targetWorkflowQueueLength = defaultTargetPipelineQueueLength
	}
	meta.targetWorkflowQueueLength = targetWorkflowQueueLength

	// Get the personalAccessToken
	personalAccessToken, ok := config.AuthParams["personalAccessToken"]
	if !ok || personalAccessToken == "" {
		return nil, errors.New("no personalAccessToken provided")
	}

	meta.personalAccessToken = personalAccessToken
	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func (s *gitlabRunnerScaler) GetWorkflowQueueLength(context.Context) (int64, error) {
	return 0, nil
}

func (s *gitlabRunnerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueLen, err := s.GetWorkflowQueueLength(ctx)

	if err != nil {
		s.logger.Error(err, "error getting workflow queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queueLen))

	return []external_metrics.ExternalMetricValue{metric}, queueLen >= s.metadata.targetWorkflowQueueLength, nil
}

func (s *gitlabRunnerScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("gitlab-runner-%s", s.metadata.projectID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetWorkflowQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *gitlabRunnerScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

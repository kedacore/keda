package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type solrScaler struct {
	metricType v2.MetricTargetType
	metadata   *solrMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type solrMetadata struct {
	triggerIndex int

	Host                       string  `keda:"name=host, order=triggerMetadata"`
	Collection                 string  `keda:"name=collection,                 order=triggerMetadata"`
	TargetQueryValue           float64 `keda:"name=targetQueryValue,           order=triggerMetadata"`
	ActivationTargetQueryValue float64 `keda:"name=activationTargetQueryValue, order=triggerMetadata, default=0"`
	Query                      string  `keda:"name=query,                      order=triggerMetadata, optional"`

	// Authentication
	Username string `keda:"name=username, order=authParams;triggerMetadata"`
	Password string `keda:"name=password, order=authParams;triggerMetadata"`
}

func (s *solrMetadata) Validate() error {
	if s.Query == "" {
		s.Query = "*:*"
	}
	return nil
}

type solrResponse struct {
	Response struct {
		NumFound int `json:"numFound"`
	} `json:"response"`
}

// NewSolrScaler creates a new solr Scaler
func NewSolrScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseSolrMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Solr metadata: %w", err)
	}
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	logger := InitializeLogger(config, "solr_scaler")

	return &solrScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// parseSolrMetadata parses the metadata and returns a solrMetadata or an error if the ScalerConfig is invalid.
func parseSolrMetadata(config *scalersconfig.ScalerConfig) (*solrMetadata, error) {
	meta := &solrMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing solr metadata: %w", err)
	}

	if !config.AsMetricSource && meta.TargetQueryValue == 0 {
		return nil, fmt.Errorf("no targetQueryValue given")
	}

	return meta, nil
}

func (s *solrScaler) getItemCount(ctx context.Context) (float64, error) {
	var SolrResponse1 *solrResponse
	var itemCount float64

	url := fmt.Sprintf("%s/solr/%s/select?q=%s&wt=json",
		s.metadata.Host, s.metadata.Collection, s.metadata.Query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}
	// Add BasicAuth
	req.SetBasicAuth(s.metadata.Username, s.metadata.Password)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("error sending request to solr, %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	err = json.Unmarshal(body, &SolrResponse1)
	if err != nil {
		return -1, fmt.Errorf("%w, make sure you enter username, password and collection values correctly in the yaml file", err)
	}
	itemCount = float64(SolrResponse1.Response.NumFound)
	return itemCount, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *solrScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("solr")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetQueryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity query from Solr,and return to external metrics and activity
func (s *solrScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	result, err := s.getItemCount(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to inspect solr, because of %w", err)
	}

	metric := GenerateMetricInMili(metricName, result)

	return append([]external_metrics.ExternalMetricValue{}, metric), result > s.metadata.ActivationTargetQueryValue, nil
}

// Close closes the http client connection.
func (s *solrScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

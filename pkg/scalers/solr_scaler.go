package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type solrScaler struct {
	metricType v2.MetricTargetType
	metadata   *solrMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type solrMetadata struct {
	host                       string
	collection                 string
	targetQueryValue           float64
	activationTargetQueryValue float64
	query                      string
	scalerIndex                int

	// Authentication
	username string
	password string
}

type solrResponse struct {
	Response struct {
		NumFound int `json:"numFound"`
	} `json:"response"`
}

// NewSolrScaler creates a new solr Scaler
func NewSolrScaler(config *ScalerConfig) (Scaler, error) {
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
func parseSolrMetadata(config *ScalerConfig) (*solrMetadata, error) {
	meta := solrMetadata{}

	if val, ok := config.TriggerMetadata["host"]; ok {
		meta.host = val
	} else {
		return nil, fmt.Errorf("no host given")
	}

	if val, ok := config.TriggerMetadata["collection"]; ok {
		meta.collection = val
	} else {
		return nil, fmt.Errorf("no collection given")
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		meta.query = "*:*"
	}

	if val, ok := config.TriggerMetadata["targetQueryValue"]; ok {
		targetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetQueryValue parsing error %w", err)
		}
		meta.targetQueryValue = targetQueryValue
	} else {
		if config.AsMetricSource {
			meta.targetQueryValue = 0
		} else {
			return nil, fmt.Errorf("no targetQueryValue given")
		}
	}

	meta.activationTargetQueryValue = 0
	if val, ok := config.TriggerMetadata["activationTargetQueryValue"]; ok {
		activationTargetQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid activationTargetQueryValue - must be an integer")
		}
		meta.activationTargetQueryValue = activationTargetQueryValue
	}
	// Parse Authentication
	if val, ok := config.AuthParams["username"]; ok {
		meta.username = val
	} else {
		return nil, fmt.Errorf("no username given")
	}

	if val, ok := config.AuthParams["password"]; ok {
		meta.password = val
	} else {
		return nil, fmt.Errorf("no password given")
	}

	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *solrScaler) getItemCount(ctx context.Context) (float64, error) {
	var SolrResponse1 *solrResponse
	var itemCount float64

	url := fmt.Sprintf("%s/solr/%s/select?q=%s&wt=json",
		s.metadata.host, s.metadata.collection, s.metadata.query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}
	// Add BasicAuth
	req.SetBasicAuth(s.metadata.username, s.metadata.password)

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
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString("solr")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetQueryValue),
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

	return append([]external_metrics.ExternalMetricValue{}, metric), result > s.metadata.activationTargetQueryValue, nil
}

// Close closes the http client connection.
func (s *solrScaler) Close(context.Context) error {
	return nil
}

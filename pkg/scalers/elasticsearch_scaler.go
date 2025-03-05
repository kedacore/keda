package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

type elasticsearchScaler struct {
	metricType v2.MetricTargetType
	metadata   elasticsearchMetadata
	esClient   *elasticsearch.Client
	logger     logr.Logger
}

type elasticsearchMetadata struct {
	Addresses             []string `keda:"name=addresses,             order=authParams;triggerMetadata, optional"`
	UnsafeSsl             bool     `keda:"name=unsafeSsl,             order=triggerMetadata, default=false"`
	Username              string   `keda:"name=username,              order=authParams;triggerMetadata, optional"`
	Password              string   `keda:"name=password,              order=authParams;resolvedEnv;triggerMetadata, optional"`
	CloudID               string   `keda:"name=cloudID,               order=authParams;triggerMetadata, optional"`
	APIKey                string   `keda:"name=apiKey,                order=authParams;triggerMetadata, optional"`
	Index                 []string `keda:"name=index,                 order=authParams;triggerMetadata, separator=;"`
	SearchTemplateName    string   `keda:"name=searchTemplateName,    order=authParams;triggerMetadata, optional"`
	Query                 string   `keda:"name=query,                 order=authParams;triggerMetadata, optional"`
	Parameters            []string `keda:"name=parameters,            order=triggerMetadata, optional, separator=;"`
	ValueLocation         string   `keda:"name=valueLocation,         order=authParams;triggerMetadata"`
	TargetValue           float64  `keda:"name=targetValue,           order=authParams;triggerMetadata"`
	ActivationTargetValue float64  `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`
	IgnoreNullValues      bool     `keda:"name=ignoreNullValues,      order=triggerMetadata, default=false"`
	MetricName            string   `keda:"name=metricName,            order=triggerMetadata, optional"`

	TriggerIndex int
}

func (m *elasticsearchMetadata) Validate() error {
	if (m.CloudID != "" || m.APIKey != "") && (len(m.Addresses) > 0 || m.Username != "" || m.Password != "") {
		return fmt.Errorf("can't provide both cloud config and endpoint addresses")
	}
	if (m.CloudID == "" && m.APIKey == "") && (len(m.Addresses) == 0 && m.Username == "" && m.Password == "") {
		return fmt.Errorf("must provide either cloud config or endpoint addresses")
	}
	if (m.CloudID != "" && m.APIKey == "") || (m.CloudID == "" && m.APIKey != "") {
		return fmt.Errorf("both cloudID and apiKey must be provided when cloudID or apiKey is used")
	}
	if len(m.Addresses) > 0 && (m.Username == "" || m.Password == "") {
		return fmt.Errorf("both username and password must be provided when addresses is used")
	}
	if m.SearchTemplateName == "" && m.Query == "" {
		return fmt.Errorf("either searchTemplateName or query must be provided")
	}
	if m.SearchTemplateName != "" && m.Query != "" {
		return fmt.Errorf("cannot provide both searchTemplateName and query")
	}

	return nil
}

func NewElasticsearchScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "elasticsearch_scaler")

	meta, err := parseElasticsearchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing elasticsearch metadata: %w", err)
	}

	esClient, err := newElasticsearchClient(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error getting elasticsearch client: %w", err)
	}
	return &elasticsearchScaler{
		metricType: metricType,
		metadata:   meta,
		esClient:   esClient,
		logger:     logger,
	}, nil
}

func parseElasticsearchMetadata(config *scalersconfig.ScalerConfig) (elasticsearchMetadata, error) {
	meta := elasticsearchMetadata{}
	err := config.TypedConfig(&meta)

	if err != nil {
		return meta, err
	}

	if meta.SearchTemplateName != "" {
		meta.MetricName = GenerateMetricNameWithIndex(config.TriggerIndex, util.NormalizeString(fmt.Sprintf("elasticsearch-%s", meta.SearchTemplateName)))
	} else {
		meta.MetricName = GenerateMetricNameWithIndex(config.TriggerIndex, "elasticsearch-query")
	}

	meta.TriggerIndex = config.TriggerIndex

	return meta, nil
}

func newElasticsearchClient(meta elasticsearchMetadata, logger logr.Logger) (*elasticsearch.Client, error) {
	var config elasticsearch.Config

	if meta.CloudID != "" {
		config = elasticsearch.Config{
			CloudID: meta.CloudID,
			APIKey:  meta.APIKey,
		}
	} else {
		config = elasticsearch.Config{
			Addresses: meta.Addresses,
			Username:  meta.Username,
			Password:  meta.Password,
		}
	}

	config.Transport = util.CreateHTTPTransport(meta.UnsafeSsl)
	esClient, err := elasticsearch.NewClient(config)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error when creating client: %s", err))
		return nil, err
	}

	_, err = esClient.Info()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Found error when pinging search engine: %s", err))
		return nil, err
	}
	return esClient, nil
}

func (s *elasticsearchScaler) Close(_ context.Context) error {
	return nil
}

// getQueryResult returns result of the scaler query
func (s *elasticsearchScaler) getQueryResult(ctx context.Context) (float64, error) {
	// Build the request body.
	var res *esapi.Response
	var err error

	if s.metadata.SearchTemplateName != "" {
		// Using SearchTemplateName
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(buildQuery(&s.metadata)); err != nil {
			s.logger.Error(err, "Error encoding query: %s", err)
		}
		res, err = s.esClient.SearchTemplate(
			&body,
			s.esClient.SearchTemplate.WithIndex(s.metadata.Index...),
			s.esClient.SearchTemplate.WithContext(ctx),
		)
	} else {
		// Using Query
		res, err = s.esClient.Search(
			s.esClient.Search.WithIndex(s.metadata.Index...),
			s.esClient.Search.WithBody(strings.NewReader(s.metadata.Query)),
			s.esClient.Search.WithContext(ctx),
		)
	}

	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Could not query elasticsearch: %s", err))
		return 0, err
	}

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	v, err := getValueFromSearch(b, s.metadata.ValueLocation, s.metadata.IgnoreNullValues)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func buildQuery(metadata *elasticsearchMetadata) map[string]interface{} {
	parameters := map[string]interface{}{}
	for _, p := range metadata.Parameters {
		if p != "" {
			kv := strings.Split(p, ":")
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			parameters[key] = value
		}
	}
	query := map[string]interface{}{
		"id": metadata.SearchTemplateName,
	}
	if len(parameters) > 0 {
		query["params"] = parameters
	}
	return query
}

func getValueFromSearch(body []byte, valueLocation string, ignoreNullValues bool) (float64, error) {
	r := gjson.GetBytes(body, valueLocation)
	errorMsg := "valueLocation must point to value of type number but got: '%s'"

	if r.Type == gjson.Null {
		if ignoreNullValues {
			return 0, nil // Return 0 when the value is null and we're ignoring null values
		}
		return 0, fmt.Errorf(errorMsg, "Null")
	}

	if r.Type == gjson.String {
		q, err := strconv.ParseFloat(r.String(), 64)
		if err != nil {
			return 0, fmt.Errorf(errorMsg, r.String())
		}
		return q, nil
	}

	if r.Type != gjson.Number {
		return 0, fmt.Errorf(errorMsg, r.Type.String())
	}
	return r.Num, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *elasticsearchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.MetricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *elasticsearchScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting elasticsearch: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetValue, nil
}

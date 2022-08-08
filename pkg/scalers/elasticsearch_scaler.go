package scalers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type elasticsearchScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *elasticsearchMetadata
	esClient   *elasticsearch.Client
	logger     logr.Logger
}

type elasticsearchMetadata struct {
	addresses             []string
	unsafeSsl             bool
	username              string
	password              string
	indexes               []string
	searchTemplateName    string
	parameters            []string
	valueLocation         string
	targetValue           float64
	activationTargetValue float64
	metricName            string
}

// NewElasticsearchScaler creates a new elasticsearch scaler
func NewElasticsearchScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "elasticsearch_scaler")

	meta, err := parseElasticsearchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing elasticsearch metadata: %s", err)
	}

	esClient, err := newElasticsearchClient(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error getting elasticsearch client: %s", err)
	}
	return &elasticsearchScaler{
		metricType: metricType,
		metadata:   meta,
		esClient:   esClient,
		logger:     logger,
	}, nil
}

const defaultUnsafeSsl = false

func parseElasticsearchMetadata(config *ScalerConfig) (*elasticsearchMetadata, error) {
	meta := elasticsearchMetadata{}

	var err error
	addresses, err := GetFromAuthOrMeta(config, "addresses")
	if err != nil {
		return nil, err
	}
	meta.addresses = splitAndTrimBySep(addresses, ",")

	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		meta.unsafeSsl, err = strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %s", err)
		}
	} else {
		meta.unsafeSsl = defaultUnsafeSsl
	}

	if val, ok := config.AuthParams["username"]; ok {
		meta.username = val
	} else if val, ok := config.TriggerMetadata["username"]; ok {
		meta.username = val
	}

	if config.AuthParams["password"] != "" {
		meta.password = config.AuthParams["password"]
	} else if config.TriggerMetadata["passwordFromEnv"] != "" {
		meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
	}

	index, err := GetFromAuthOrMeta(config, "index")
	if err != nil {
		return nil, err
	}
	meta.indexes = splitAndTrimBySep(index, ";")

	meta.searchTemplateName, err = GetFromAuthOrMeta(config, "searchTemplateName")
	if err != nil {
		return nil, err
	}

	if val, ok := config.TriggerMetadata["parameters"]; ok {
		meta.parameters = splitAndTrimBySep(val, ";")
	}

	meta.valueLocation, err = GetFromAuthOrMeta(config, "valueLocation")
	if err != nil {
		return nil, err
	}

	targetValue, err := GetFromAuthOrMeta(config, "targetValue")
	if err != nil {
		return nil, err
	}
	meta.targetValue, err = strconv.ParseFloat(targetValue, 64)
	if err != nil {
		return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
	}

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		meta.activationTargetValue, err = strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %s", err.Error())
		}
	}

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("elasticsearch-%s", meta.searchTemplateName)))
	return &meta, nil
}

// newElasticsearchClient creates elasticsearch db connection
func newElasticsearchClient(meta *elasticsearchMetadata, logger logr.Logger) (*elasticsearch.Client, error) {
	config := elasticsearch.Config{Addresses: meta.addresses}
	if meta.username != "" {
		config.Username = meta.username
	}
	if meta.password != "" {
		config.Password = meta.password
	}

	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: meta.unsafeSsl}
	config.Transport = transport

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

func (s *elasticsearchScaler) Close(ctx context.Context) error {
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *elasticsearchScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getQueryResult(ctx)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Error inspecting elasticsearch: %s", err))
		return false, err
	}
	return messages > s.metadata.activationTargetValue, nil
}

// getQueryResult returns result of the scaler query
func (s *elasticsearchScaler) getQueryResult(ctx context.Context) (float64, error) {
	// Build the request body.
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(buildQuery(s.metadata)); err != nil {
		s.logger.Error(err, "Error encoding query: %s", err)
	}

	// Run the templated search
	res, err := s.esClient.SearchTemplate(
		&body,
		s.esClient.SearchTemplate.WithIndex(s.metadata.indexes...),
		s.esClient.SearchTemplate.WithContext(ctx),
	)
	if err != nil {
		s.logger.Error(err, fmt.Sprintf("Could not query elasticsearch: %s", err))
		return 0, err
	}

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	v, err := getValueFromSearch(b, s.metadata.valueLocation)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func buildQuery(metadata *elasticsearchMetadata) map[string]interface{} {
	parameters := map[string]interface{}{}
	for _, p := range metadata.parameters {
		if p != "" {
			kv := splitAndTrimBySep(p, ":")
			parameters[kv[0]] = kv[1]
		}
	}
	query := map[string]interface{}{
		"id": metadata.searchTemplateName,
	}
	if len(parameters) > 0 {
		query["params"] = parameters
	}
	return query
}

func getValueFromSearch(body []byte, valueLocation string) (float64, error) {
	r := gjson.GetBytes(body, valueLocation)
	errorMsg := "valueLocation must point to value of type number but got: '%s'"
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
func (s *elasticsearchScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *elasticsearchScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting elasticsearch: %s", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Splits a string separated by a specified separator and trims space from all the elements.
func splitAndTrimBySep(s string, sep string) []string {
	x := strings.Split(s, sep)
	for i := range x {
		x[i] = strings.Trim(x[i], " ")
	}
	return x
}

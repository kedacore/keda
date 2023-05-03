package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/util"
)

type elasticsearchScaler struct {
	metricType v2.MetricTargetType
	metadata   *elasticsearchMetadata
	esClient   *elasticsearch.Client
	logger     logr.Logger
}

type elasticsearchMetadata struct {
	addresses             []string
	unsafeSsl             bool
	username              string
	password              string
	cloudID               string
	apiKey                string
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

const defaultUnsafeSsl = false

func hasCloudConfig(meta *elasticsearchMetadata) bool {
	if meta.cloudID != "" {
		return true
	}
	if meta.apiKey != "" {
		return true
	}
	return false
}

func hasEndpointsConfig(meta *elasticsearchMetadata) bool {
	if len(meta.addresses) > 0 {
		return true
	}
	if meta.username != "" {
		return true
	}
	if meta.password != "" {
		return true
	}
	return false
}

func extractEndpointsConfig(config *ScalerConfig, meta *elasticsearchMetadata) error {
	addresses, err := GetFromAuthOrMeta(config, "addresses")
	if err != nil {
		return err
	}

	meta.addresses = splitAndTrimBySep(addresses, ",")
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

	return nil
}

func extractCloudConfig(config *ScalerConfig, meta *elasticsearchMetadata) error {
	cloudID, err := GetFromAuthOrMeta(config, "cloudID")
	if err != nil {
		return err
	}
	meta.cloudID = cloudID

	apiKey, err := GetFromAuthOrMeta(config, "apiKey")
	if err != nil {
		return err
	}
	meta.apiKey = apiKey
	return nil
}

var (
	// ErrElasticsearchMissingAddressesOrCloudConfig is returned when endpoint addresses or cloud config is missing.
	ErrElasticsearchMissingAddressesOrCloudConfig = errors.New("must provide either endpoint addresses or cloud config")

	// ErrElasticsearchConfigConflict is returned when both endpoint addresses and cloud config are provided.
	ErrElasticsearchConfigConflict = errors.New("can't provide endpoint addresses and cloud config at the same time")
)

func parseElasticsearchMetadata(config *ScalerConfig) (*elasticsearchMetadata, error) {
	meta := elasticsearchMetadata{}

	var err error
	addresses, err := GetFromAuthOrMeta(config, "addresses")
	cloudID, errCloudConfig := GetFromAuthOrMeta(config, "cloudID")
	if err != nil && errCloudConfig != nil {
		return nil, ErrElasticsearchMissingAddressesOrCloudConfig
	}

	if err == nil && addresses != "" {
		err = extractEndpointsConfig(config, &meta)
		if err != nil {
			return nil, err
		}
	}
	if errCloudConfig == nil && cloudID != "" {
		err = extractCloudConfig(config, &meta)
		if err != nil {
			return nil, err
		}
	}

	if hasEndpointsConfig(&meta) && hasCloudConfig(&meta) {
		return nil, ErrElasticsearchConfigConflict
	}

	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		unsafeSsl, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	} else {
		meta.unsafeSsl = defaultUnsafeSsl
	}

	index, err := GetFromAuthOrMeta(config, "index")
	if err != nil {
		return nil, err
	}
	meta.indexes = splitAndTrimBySep(index, ";")

	searchTemplateName, err := GetFromAuthOrMeta(config, "searchTemplateName")
	if err != nil {
		return nil, err
	}
	meta.searchTemplateName = searchTemplateName

	if val, ok := config.TriggerMetadata["parameters"]; ok {
		meta.parameters = splitAndTrimBySep(val, ";")
	}

	valueLocation, err := GetFromAuthOrMeta(config, "valueLocation")
	if err != nil {
		return nil, err
	}
	meta.valueLocation = valueLocation

	targetValueString, err := GetFromAuthOrMeta(config, "targetValue")
	if err != nil {
		return nil, err
	}
	targetValue, err := strconv.ParseFloat(targetValueString, 64)
	if err != nil {
		return nil, fmt.Errorf("targetValue parsing error: %w", err)
	}
	meta.targetValue = targetValue

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error: %w", err)
		}
		meta.activationTargetValue = activationTargetValue
	}

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, util.NormalizeString(fmt.Sprintf("elasticsearch-%s", meta.searchTemplateName)))
	return &meta, nil
}

// newElasticsearchClient creates elasticsearch db connection
func newElasticsearchClient(meta *elasticsearchMetadata, logger logr.Logger) (*elasticsearch.Client, error) {
	var config elasticsearch.Config

	if hasCloudConfig(meta) {
		config = elasticsearch.Config{
			CloudID: meta.cloudID,
			APIKey:  meta.apiKey,
		}
	} else {
		config = elasticsearch.Config{
			Addresses: meta.addresses,
		}
		if meta.username != "" {
			config.Username = meta.username
		}
		if meta.password != "" {
			config.Password = meta.password
		}
	}

	config.Transport = util.CreateHTTPTransport(meta.unsafeSsl)
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
func (s *elasticsearchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
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

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetValue, nil
}

// Splits a string separated by a specified separator and trims space from all the elements.
func splitAndTrimBySep(s string, sep string) []string {
	x := strings.Split(s, sep)
	for i := range x {
		x[i] = strings.Trim(x[i], " ")
	}
	return x
}

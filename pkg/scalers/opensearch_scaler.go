package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

type opensearchScaler struct {
	metricType  v2.MetricTargetType
	metadata    opensearchMetadata
	osAPIClient *opensearchapi.Client
	logger      logr.Logger
}

type opensearchMetadata struct {
	Addresses             []string `keda:"name=addresses,             order=authParams;triggerMetadata"`
	UnsafeSsl             bool     `keda:"name=unsafeSsl,             order=triggerMetadata, default=false"`
	EnableTLS             bool     `keda:"name=enableTLS,             order=triggerMetadata;authParams, default=false"`
	Username              string   `keda:"name=username,              order=authParams;triggerMetadata, optional"`
	Password              string   `keda:"name=password,              order=authParams;resolvedEnv;triggerMetadata, optional"`
	CACert                string   `keda:"name=caCert,                order=authParams;triggerMetadata, optional"`
	ClientCert            string   `keda:"name=clientCert,            order=authParams;triggerMetadata, optional"`
	ClientKey             string   `keda:"name=clientKey,             order=authParams;triggerMetadata, optional"`
	Index                 []string `keda:"name=index,                 order=authParams;triggerMetadata, separator=;"`
	SearchTemplateName    string   `keda:"name=searchTemplateName,    order=authParams;triggerMetadata, optional"`
	Query                 string   `keda:"name=query,                 order=authParams;triggerMetadata, optional"`
	Parameters            []string `keda:"name=parameters,            order=triggerMetadata, optional, separator=;"`
	ValueLocation         string   `keda:"name=valueLocation,         order=authParams;triggerMetadata"`
	TargetValue           float64  `keda:"name=targetValue,           order=authParams;triggerMetadata"`
	ActivationTargetValue float64  `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`
	IgnoreNullValues      bool     `keda:"name=ignoreNullValues,      order=triggerMetadata, default=false"`

	metricName string

	TriggerIndex int
}

func (m *opensearchMetadata) Validate() error {
	if m.EnableTLS && (m.ClientCert == "" || m.ClientKey == "") {
		return fmt.Errorf("both clientCert and clientKey must be provided when enableTLS is true")
	}
	if !m.EnableTLS && (m.Username == "" || m.Password == "") {
		return fmt.Errorf("both username and password must be provided for basic auth")
	}
	if m.SearchTemplateName == "" && m.Query == "" {
		return fmt.Errorf("either searchTemplateName or query must be provided")
	}
	if m.SearchTemplateName != "" && m.Query != "" {
		return fmt.Errorf("cannot provide both searchTemplateName and query")
	}

	return nil
}

func NewOpensearchScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "opensearch_scaler")

	meta, err := parseOpensearchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse opensearch metadata: %w", err)
	}

	opensearchAPIClient, err := newOpensearchAPIClient(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	return &opensearchScaler{
		metricType:  metricType,
		metadata:    meta,
		osAPIClient: opensearchAPIClient,
		logger:      logger,
	}, nil
}

func newOpensearchAPIClient(meta opensearchMetadata, logger logr.Logger) (*opensearchapi.Client, error) {
	if meta.EnableTLS {
		return newOpensearchAPIClientWithTLS(meta, logger)
	}
	return newOpensearchAPIClientWithBasicAuth(meta, logger)
}

func newOpensearchAPIClientWithTLS(meta opensearchMetadata, logger logr.Logger) (*opensearchapi.Client, error) {
	tlsConfig, err := util.NewTLSConfig(meta.ClientCert, meta.ClientKey, meta.CACert, meta.UnsafeSsl)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	return newOpensearchAPIClientFromConfig(opensearch.Config{
		Addresses: meta.Addresses,
		Transport: util.CreateHTTPTransportWithTLSConfig(tlsConfig),
	}, logger)
}

func newOpensearchAPIClientWithBasicAuth(meta opensearchMetadata, logger logr.Logger) (*opensearchapi.Client, error) {
	tlsConfig, err := util.NewTLSConfig("", "", meta.CACert, meta.UnsafeSsl)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	return newOpensearchAPIClientFromConfig(opensearch.Config{
		Addresses: meta.Addresses,
		Username:  meta.Username,
		Password:  meta.Password,
		Transport: util.CreateHTTPTransportWithTLSConfig(tlsConfig),
	}, logger)
}

func newOpensearchAPIClientFromConfig(cfg opensearch.Config, logger logr.Logger) (*opensearchapi.Client, error) {
	client, err := opensearchapi.NewClient(opensearchapi.Config{Client: cfg})
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to create opensearch client: %s", err))
		return nil, err
	}

	_, err = client.Ping(context.Background(), nil)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to ping the opensearch engine: %s", err))
		return nil, err
	}

	return client, nil
}

func parseOpensearchMetadata(config *scalersconfig.ScalerConfig) (opensearchMetadata, error) {
	meta := opensearchMetadata{}
	err := config.TypedConfig(&meta)

	if err != nil {
		return meta, err
	}

	if meta.SearchTemplateName != "" {
		meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, util.NormalizeString(fmt.Sprintf("opensearch-%s", meta.SearchTemplateName)))
	} else {
		meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, "opensearch-query")
	}

	return meta, nil
}

func (s *opensearchScaler) Close(_ context.Context) error {
	return nil
}

func (s *opensearchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *opensearchScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queryResult, err := s.getQueryResult(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting opensearch: %w", err)
	}

	metric := GenerateMetricInMili(metricName, queryResult)

	return []external_metrics.ExternalMetricValue{metric}, queryResult > s.metadata.ActivationTargetValue, nil
}

func (s *opensearchScaler) getQueryResult(ctx context.Context) (float64, error) {
	var responseBody []byte
	var err error

	if strings.TrimSpace(s.metadata.SearchTemplateName) != "" {
		responseBody, err = s.searchTemplate(ctx)
		if err != nil {
			return 0, err
		}
	} else {
		responseBody, err = s.search(ctx)
		if err != nil {
			return 0, err
		}
	}

	value, err := s.getValueFromSearchResultByValueLocation(responseBody)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (s *opensearchScaler) getValueFromSearchResultByValueLocation(responseBody []byte) (float64, error) {
	r := gjson.GetBytes(responseBody, s.metadata.ValueLocation)
	errorMsg := "valueLocation must point to value of type number but got: '%s'"

	if r.Type == gjson.Null {
		if s.metadata.IgnoreNullValues {
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

func (s *opensearchScaler) search(ctx context.Context) ([]byte, error) {
	query := strings.TrimSpace(s.metadata.Query)
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query must be provided when searchTemplateName is empty")
	}
	if !json.Valid([]byte(query)) {
		return nil, status.Error(codes.InvalidArgument, "invalid query JSON")
	}

	searchResponse, err := s.osAPIClient.Search(ctx, &opensearchapi.SearchReq{
		Indices: s.metadata.Index,
		Body:    bytes.NewReader([]byte(query)),
	})
	if err != nil {
		return nil, err
	}

	responseBody, err := s.readResponseBody(searchResponse.Inspect().Response)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}

func (s *opensearchScaler) searchTemplate(ctx context.Context) ([]byte, error) {
	query, err := s.buildQueryFromMetadata()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode search template request: %v", err)
	}

	searchTemplResponse, err := s.osAPIClient.SearchTemplate(ctx, opensearchapi.SearchTemplateReq{
		Indices: s.metadata.Index,
		Body:    bytes.NewReader(body),
	})
	if err != nil {
		return nil, err
	}

	responseBody, err := s.readResponseBody(searchTemplResponse.Inspect().Response)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}

func (s *opensearchScaler) buildQueryFromMetadata() (map[string]interface{}, error) {
	parameters := map[string]interface{}{}
	for _, p := range s.metadata.Parameters {
		if p != "" {
			kv := strings.SplitN(p, ":", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid parameter format %q, expected key:value", p)
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if key == "" || value == "" {
				return nil, fmt.Errorf("invalid parameter format %q, expected key:value", p)
			}
			parameters[key] = value
		}
	}
	query := map[string]interface{}{
		"id": s.metadata.SearchTemplateName,
	}
	if len(parameters) > 0 {
		query["params"] = parameters
	}
	return query, nil
}

func (s *opensearchScaler) readResponseBody(resp *opensearch.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, status.Error(codes.Internal, "empty search response")
	}
	defer resp.Body.Close()

	if err := s.checkHTTPStatus(resp.StatusCode); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read search response: %v", err)
	}

	return body, nil
}

// checkHTTPStatus returns a clear error for authentication and authorization failures
func (s *opensearchScaler) checkHTTPStatus(statusCode int) error {
	if statusCode == 401 {
		return fmt.Errorf("opensearch authentication failed (HTTP 401): check username and password")
	}
	if statusCode == 403 {
		return fmt.Errorf("opensearch authorization failed (HTTP 403): user has insufficient permissions")
	}
	return nil
}

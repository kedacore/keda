package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

// forecastDurationMultiplier controls the forecast window size relative to lookAhead.
const forecastDurationMultiplier = 2

// forecastRenewThreshold is the fraction of the forecast window that must remain before renewal is triggered.
const forecastRenewThreshold = 0.5

type elasticForecastScaler struct {
	metricType v2.MetricTargetType
	metadata   elasticForecastMetadata
	esClient   *elasticsearch.Client
	logger     logr.Logger

	mu                 sync.Mutex
	forecastID         string    // ID of the current active forecast
	forecastExpiry     time.Time // when the current forecast documents expire
	previousForecastID string    // ID of the forecast before the most recent renewal
	renewalInProgress  bool      // guards against concurrent renewals
}

type elasticForecastMetadata struct {
	Addresses []string `keda:"name=addresses, order=authParams;triggerMetadata,             optional"`
	UnsafeSsl bool     `keda:"name=unsafeSsl, order=triggerMetadata,                        default=false"`
	Username  string   `keda:"name=username,  order=authParams;triggerMetadata,             optional"`
	Password  string   `keda:"name=password,  order=authParams;resolvedEnv;triggerMetadata, optional"`
	CloudID   string   `keda:"name=cloudID,   order=authParams;triggerMetadata,             optional"`
	APIKey    string   `keda:"name=apiKey,    order=authParams;triggerMetadata,             optional"`

	JobID       string        `keda:"name=jobID,       order=triggerMetadata"`
	LookAhead   time.Duration `keda:"name=lookAhead,   order=triggerMetadata, default=10m"`
	TargetValue float64       `keda:"name=targetValue, order=triggerMetadata"`

	ActivationTargetValue float64 `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`

	// Index controls which .ml-anomalies- index to query. Default is "*" (.ml-anomalies-*).
	Index string `keda:"name=index, order=triggerMetadata, default=*"`

	// Filters for multi-metric forecasts (jobs with by_field or partition_field).
	PartitionFieldValue string `keda:"name=partitionFieldValue, order=triggerMetadata, optional"`
	ByFieldValue        string `keda:"name=byFieldValue,        order=triggerMetadata, optional"`

	MetricName   string
	TriggerIndex int
}

// forecastDuration is the window passed to the ES forecast API.
func (m *elasticForecastMetadata) forecastDuration() time.Duration {
	return m.LookAhead * forecastDurationMultiplier
}

func (m *elasticForecastMetadata) Validate() error {
	cloudMode := m.CloudID != "" || m.APIKey != ""
	addrMode := len(m.Addresses) > 0 || m.Username != "" || m.Password != ""

	if cloudMode && addrMode {
		return fmt.Errorf("cannot provide both cloud config (cloudID/apiKey) and endpoint addresses")
	}
	if !cloudMode && !addrMode {
		return fmt.Errorf("must provide either cloud config (cloudID + apiKey) or endpoint addresses")
	}
	if (m.CloudID != "" && m.APIKey == "") || (m.CloudID == "" && m.APIKey != "") {
		return fmt.Errorf("cloudID and apiKey must both be provided together")
	}
	if len(m.Addresses) > 0 && (m.Username == "" || m.Password == "") {
		return fmt.Errorf("username and password must both be provided when addresses is used")
	}
	if m.LookAhead <= 0 {
		return fmt.Errorf("lookAhead must be positive")
	}
	if m.TargetValue <= 0 {
		return fmt.Errorf("targetValue must be greater than 0")
	}
	return nil
}

func NewElasticForecastScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "elastic_forecast_scaler")

	meta, err := parseElasticForecastMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing elastic-forecast metadata: %w", err)
	}

	esClient, err := newElasticForecastESClient(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating elasticsearch client: %w", err)
	}

	return &elasticForecastScaler{
		metricType: metricType,
		metadata:   meta,
		esClient:   esClient,
		logger:     logger,
	}, nil
}

func parseElasticForecastMetadata(config *scalersconfig.ScalerConfig) (elasticForecastMetadata, error) {
	meta := elasticForecastMetadata{}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, err
	}
	meta.MetricName = GenerateMetricNameWithIndex(
		config.TriggerIndex,
		util.NormalizeString(fmt.Sprintf("elastic-forecast-%s", meta.JobID)),
	)
	meta.TriggerIndex = config.TriggerIndex
	return meta, nil
}

func newElasticForecastESClient(meta elasticForecastMetadata, logger logr.Logger) (*elasticsearch.Client, error) {
	var cfg elasticsearch.Config
	if meta.CloudID != "" {
		cfg = elasticsearch.Config{CloudID: meta.CloudID, APIKey: meta.APIKey}
	} else {
		cfg = elasticsearch.Config{
			Addresses: meta.Addresses,
			Username:  meta.Username,
			Password:  meta.Password,
		}
	}
	cfg.Transport = util.CreateHTTPTransport(meta.UnsafeSsl)

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logger.Error(err, "error creating elasticsearch client")
		return nil, err
	}
	if _, err = client.Info(); err != nil {
		logger.Error(err, "error pinging elasticsearch")
		return nil, err
	}
	return client, nil
}

type forecastAPIResponse struct {
	Acknowledged bool   `json:"acknowledged"`
	ForecastID   string `json:"forecast_id"`
}

// ensureForecastValid checks whether a new forecast needs to be created.
func (s *elasticForecastScaler) ensureForecastValid(ctx context.Context) error {
	duration := s.metadata.forecastDuration()
	renewThreshold := time.Duration(float64(duration) * forecastRenewThreshold)

	s.mu.Lock()
	needsRenewal := s.forecastID == "" || time.Until(s.forecastExpiry) < renewThreshold
	if !needsRenewal || s.renewalInProgress {
		s.mu.Unlock()
		return nil
	}
	// Mark renewal in progress.
	s.renewalInProgress = true
	currentID := s.forecastID
	s.mu.Unlock()

	if currentID == "" {
		s.logger.Info("creating initial ML forecast", "jobID", s.metadata.JobID)
	} else {
		s.logger.Info("renewing ML forecast (window near expiry)",
			"jobID", s.metadata.JobID,
			"currentForecastID", currentID,
			"remainingWindow", time.Until(s.forecastExpiry).Round(time.Second),
			"renewThreshold", renewThreshold,
		)
	}

	err := s.createForecast(ctx)

	s.mu.Lock()
	s.renewalInProgress = false
	s.mu.Unlock()

	return err
}

// createForecast calls the Elasticsearch ML forecast API.
func (s *elasticForecastScaler) createForecast(ctx context.Context) error {
	duration := s.metadata.forecastDuration()
	esDuration := fmt.Sprintf("%ds", int64(math.Ceil(duration.Seconds())))

	payload, err := json.Marshal(map[string]string{
		"duration":   esDuration,
		"expires_in": esDuration,
	})
	if err != nil {
		return fmt.Errorf("marshal forecast body: %w", err)
	}

	path := fmt.Sprintf("/_ml/anomaly_detectors/%s/_forecast", s.metadata.JobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build forecast request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.esClient.Perform(req)
	if err != nil {
		return fmt.Errorf("forecast API call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read forecast response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("forecast API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var fr forecastAPIResponse
	if err := json.Unmarshal(body, &fr); err != nil {
		return fmt.Errorf("unmarshal forecast response: %w", err)
	}
	if fr.ForecastID == "" {
		return fmt.Errorf("forecast API response contained no forecast_id: %s", string(body))
	}

	expiry := time.Now().Add(duration)

	s.mu.Lock()
	if s.forecastID != "" {
		s.previousForecastID = s.forecastID
	}
	s.forecastID = fr.ForecastID
	s.forecastExpiry = expiry
	s.mu.Unlock()

	s.logger.Info("ML forecast created",
		"jobID", s.metadata.JobID,
		"forecastID", fr.ForecastID,
		"lookAhead", s.metadata.LookAhead,
		"forecastDuration", duration,
		"expiresAt", expiry.Format(time.RFC3339),
	)
	return nil
}

// getForecastedValue queries for the model_forecast bucket covering now + lookAhead.
func (s *elasticForecastScaler) getForecastedValue(ctx context.Context) (float64, error) {
	s.mu.Lock()
	forecastID := s.forecastID
	previousID := s.previousForecastID
	s.mu.Unlock()

	if forecastID == "" {
		return 0, fmt.Errorf("no active forecast ID available")
	}

	val, found, err := s.queryForecastBucket(ctx, forecastID)
	if err != nil {
		return 0, err
	}
	if found {
		return val, nil
	}

	// The new forecast may not yet be indexed, then fall back to the previous forecast
	if previousID != "" {
		s.logger.V(1).Info("new forecast not yet indexed, falling back to previous",
			"jobID", s.metadata.JobID,
			"newForecastID", forecastID,
			"previousForecastID", previousID,
		)
		val, found, err = s.queryForecastBucket(ctx, previousID)
		if err != nil {
			return 0, err
		}
		if found {
			return val, nil
		}
	}

	// No forecast bucket available, return 0 (like ignoreNullValues).
	s.logger.V(1).Info("forecast not yet indexed, returning 0",
		"jobID", s.metadata.JobID,
		"forecastID", forecastID,
		"lookAheadTarget", time.Now().Add(s.metadata.LookAhead).Format(time.RFC3339),
	)
	return 0, nil
}

// queryForecastBucket returns the forecast_prediction
func (s *elasticForecastScaler) queryForecastBucket(ctx context.Context, forecastID string) (float64, bool, error) {
	targetMs := time.Now().Add(s.metadata.LookAhead).UnixMilli()

	filters := []interface{}{
		map[string]interface{}{"term": map[string]interface{}{
			"job_id": s.metadata.JobID,
		}},
		map[string]interface{}{"term": map[string]interface{}{
			"forecast_id": forecastID,
		}},
		map[string]interface{}{"term": map[string]interface{}{
			"result_type": "model_forecast",
		}},
	}

	if s.metadata.PartitionFieldValue != "" {
		filters = append(filters, map[string]interface{}{"term": map[string]interface{}{
			"partition_field_value": s.metadata.PartitionFieldValue,
		}})
	}

	if s.metadata.ByFieldValue != "" {
		filters = append(filters, map[string]interface{}{"term": map[string]interface{}{
			"by_field_value": s.metadata.ByFieldValue,
		}})
	}

	// Select the bucket whose timestamp is closest to (but not after) the target moment.
	filters = append(filters, map[string]interface{}{"range": map[string]interface{}{
		"timestamp": map[string]interface{}{"lte": targetMs},
	}})

	query := map[string]interface{}{
		"size": 1,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"timestamp": map[string]interface{}{"order": "desc"},
			},
		},
		"_source": []string{"forecast_prediction", "timestamp"},
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return 0, false, fmt.Errorf("marshal forecast query: %w", err)
	}

	index := fmt.Sprintf(".ml-anomalies-%s", s.metadata.Index)

	res, err := s.esClient.Search(
		s.esClient.Search.WithIndex(index),
		s.esClient.Search.WithBody(strings.NewReader(string(queryBytes))),
		s.esClient.Search.WithContext(ctx),
	)
	if err != nil {
		return 0, false, fmt.Errorf("querying forecast index: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, false, fmt.Errorf("reading forecast query response: %w", err)
	}
	if res.IsError() {
		return 0, false, fmt.Errorf("forecast query returned error %d: %s", res.StatusCode, string(respBody))
	}

	r := gjson.GetBytes(respBody, "hits.hits.0._source.forecast_prediction")
	if !r.Exists() || r.Type == gjson.Null {
		hitsForThisForecast := gjson.GetBytes(respBody, "hits.total.value").Int()
		s.logger.V(1).Info("no forecast bucket matched query",
			"jobID", s.metadata.JobID,
			"forecastID", forecastID,
			"targetMs", targetMs,
			"targetTime", time.UnixMilli(targetMs).Format(time.RFC3339),
			"hitsForThisForecast", hitsForThisForecast,
			"index", index,
		)
		return 0, false, nil
	}

	switch r.Type {
	case gjson.Number:
		return r.Num, true, nil
	case gjson.String:
		var v float64
		if _, err := fmt.Sscanf(r.String(), "%f", &v); err != nil {
			return 0, false, fmt.Errorf("forecast_prediction %q is not numeric: %w", r.String(), err)
		}
		return v, true, nil
	default:
		return 0, false, fmt.Errorf("unexpected forecast_prediction type %s", r.Type.String())
	}
}

func (s *elasticForecastScaler) Close(_ context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec consumed by the HPA controller.
func (s *elasticForecastScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{Name: s.metadata.MetricName},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	return []v2.MetricSpec{
		{External: externalMetric, Type: externalMetricType},
	}
}

// GetMetricsAndActivity ensures a valid forecast exists, then queries the predicted value for now + lookAhead.
func (s *elasticForecastScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	if err := s.ensureForecastValid(ctx); err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error ensuring ML forecast is valid: %w", err)
	}

	val, err := s.getForecastedValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error reading elastic forecast value: %w", err)
	}

	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationTargetValue, nil
}

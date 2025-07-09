package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	honeycombScalerName = "honeycomb"
	honeycombBaseURL    = "https://api.honeycomb.io/1"
	maxPollAttempts     = 5
	initialPollDelay    = 2 * time.Second
    honeycombQueryResultsLimit = 10000
)

type honeycombScaler struct {
	metricType v2.MetricTargetType
	metadata   honeycombMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type honeycombMetadata struct {
	APIKey              string                 `keda:"name=apiKey, order=authParams;triggerMetadata"`
	Dataset             string                 `keda:"name=dataset, order=triggerMetadata"`
	Query               map[string]interface{} `keda:"name=query, order=triggerMetadata, optional"`
	QueryRaw            string                 `keda:"name=queryRaw, order=triggerMetadata, optional"`
	ResultField         string                 `keda:"name=resultField, order=triggerMetadata, optional"`
	ActivationThreshold float64                `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	Threshold           float64                `keda:"name=threshold, order=triggerMetadata"`
	Breakdowns          []string               `keda:"name=breakdowns, order=triggerMetadata, optional"`
	Calculation         string                 `keda:"name=calculation, order=triggerMetadata, default=COUNT"`
	Limit               int                    `keda:"name=limit, order=triggerMetadata, default=1"`
	TimeRange           int                    `keda:"name=timeRange, order=triggerMetadata, default=60"`
	TriggerIndex        int
}

func NewHoneycombScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, fmt.Sprintf("%s_scaler", honeycombScalerName))

	meta, err := parseHoneycombMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing honeycomb metadata: %w", err)
	}

	logger.Info("Initializing Honeycomb Scaler", "dataset", meta.Dataset)

	return &honeycombScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}, nil
}

func parseHoneycombMetadata(config *scalersconfig.ScalerConfig) (honeycombMetadata, error) {
	meta := honeycombMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing honeycomb metadata: %w", err)
	}
	meta.TriggerIndex = config.TriggerIndex

	// Use queryRaw if provided, else build query from legacy fields
	if raw, ok := config.TriggerMetadata["queryRaw"]; ok && raw != "" {
		var q map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &q); err != nil {
			return meta, fmt.Errorf("error parsing queryRaw: %w", err)
		}
		meta.Query = q
	} else if meta.Query == nil {
		q := make(map[string]interface{})
		if len(meta.Breakdowns) > 0 {
			q["breakdowns"] = meta.Breakdowns
		}
		if meta.Calculation != "" {
			q["calculations"] = []map[string]string{{"op": meta.Calculation}}
		}
		if meta.Limit > 0 {
			q["limit"] = meta.Limit
		}
		if meta.TimeRange > 0 {
			q["time_range"] = meta.TimeRange
		}
		meta.Query = q
	}
	if meta.Query == nil {
		return meta, errors.New("no valid query provided in 'queryRaw', 'query', or legacy fields")
	}
	return meta, nil
}

func (s *honeycombScaler) Close(context.Context) error { return nil }

// ----- Core Query Logic -----

func (s *honeycombScaler) executeHoneycombQuery(ctx context.Context) (float64, error) {
	// 1. Create Query
	createURL := fmt.Sprintf("%s/queries/%s", honeycombBaseURL, s.metadata.Dataset)
	bodyBytes, _ := json.Marshal(s.metadata.Query)
	req, _ := http.NewRequestWithContext(ctx, "POST", createURL, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Honeycomb-Team", s.metadata.APIKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("honeycomb create query error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("honeycomb createQuery status: %s - %s", resp.Status, string(body))
	}
	var createRes struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&createRes); err != nil {
		return 0, fmt.Errorf("decode createQuery: %w", err)
	}
	if createRes.ID == "" {
		return 0, errors.New("createQuery: missing query id")
	}

	// 2. Run Query
	runURL := fmt.Sprintf("%s/query_results/%s", honeycombBaseURL, s.metadata.Dataset)
	runBody, _ := json.Marshal(map[string]interface{}{
		"query_id":                   createRes.ID,
		"disable_series":             false,
		"disable_total_by_aggregate": true,
		"disable_other_by_aggregate": true,
		"limit":                      honeycombQueryResultsLimit,
	})
	runReq, _ := http.NewRequestWithContext(ctx, "POST", runURL, bytes.NewBuffer(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq.Header.Set("X-Honeycomb-Team", s.metadata.APIKey)
	runResp, err := s.httpClient.Do(runReq)
	if err != nil {
		return 0, fmt.Errorf("honeycomb run query error: %w", err)
	}
	defer runResp.Body.Close()
	if runResp.StatusCode == 429 {
		return 0, errors.New("honeycomb: rate limited (429), back off and try again later")
	}
	if runResp.StatusCode != 200 && runResp.StatusCode != 201 {
		body, _ := io.ReadAll(runResp.Body)
		return 0, fmt.Errorf("honeycomb runQuery status: %s - %s", runResp.Status, string(body))
	}
	var runRes struct {
		ID       string `json:"id"`
		Complete bool   `json:"complete"`
		Data     struct {
			Results []map[string]interface{} `json:"results"`
		} `json:"data"`
	}
	if err := json.NewDecoder(runResp.Body).Decode(&runRes); err != nil {
		return 0, fmt.Errorf("decode runQuery: %w", err)
	}
	if runRes.ID == "" {
		return 0, errors.New("runQuery: missing queryResult id")
	}
	if runRes.Complete && len(runRes.Data.Results) > 0 {
		return extractResultField(runRes.Data.Results, s.metadata.ResultField)
	}

	// 3. Poll for completion (exponential backoff)
	pollURL := fmt.Sprintf("%s/query_results/%s/%s", honeycombBaseURL, s.metadata.Dataset, runRes.ID)
	pollDelay := initialPollDelay
	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		time.Sleep(pollDelay)
		pollDelay *= 2
		statusReq, _ := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
		statusReq.Header.Set("X-Honeycomb-Team", s.metadata.APIKey)
		statusResp, err := s.httpClient.Do(statusReq)
		if err != nil {
			return 0, fmt.Errorf("honeycomb poll query error: %w", err)
		}
		defer statusResp.Body.Close()
		if statusResp.StatusCode == 429 {
			return 0, errors.New("honeycomb: rate limited (429) on poll, back off and try again later")
		}
		if statusResp.StatusCode != 200 && statusResp.StatusCode != 201 {
			body, _ := io.ReadAll(statusResp.Body)
			return 0, fmt.Errorf("honeycomb pollQuery status: %s - %s", statusResp.Status, string(body))
		}
		var pollRes struct {
			Complete bool `json:"complete"`
			Data     struct {
				Results []map[string]interface{} `json:"results"`
			} `json:"data"`
		}
		if err := json.NewDecoder(statusResp.Body).Decode(&pollRes); err != nil {
			return 0, fmt.Errorf("pollQuery decode error: %w", err)
		}
		if pollRes.Complete && len(pollRes.Data.Results) > 0 {
			return extractResultField(pollRes.Data.Results, s.metadata.ResultField)
		}
	}
	return 0, errors.New("honeycomb: timed out waiting for query result")
}

func extractResultField(results []map[string]interface{}, field string) (float64, error) {
	if len(results) == 0 {
		return 0, errors.New("no results from Honeycomb")
	}
	dataObj, ok := results[0]["data"].(map[string]interface{})
	if !ok {
		return 0, errors.New("missing 'data' field in Honeycomb result")
	}
	if field == "" {
		for _, v := range dataObj {
			switch val := v.(type) {
			case float64:
				return val, nil
			case int:
				return float64(val), nil
			case int64:
				return float64(val), nil
			}
		}
		return 0, errors.New("no numeric value found in Honeycomb result data")
	}
	v, ok := dataObj[field]
	if !ok {
		return 0, fmt.Errorf("field '%s' not found in Honeycomb result data", field)
	}
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	}
	return 0, fmt.Errorf("no numeric value found for field '%s'", field)
}

// ----- KEDA Scaler interface -----
func (s *honeycombScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeHoneycombQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing Honeycomb query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

func (s *honeycombScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(honeycombScalerName)
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}
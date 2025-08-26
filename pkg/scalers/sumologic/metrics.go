package sumologic

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	metricsQueryPath = "api/v1/metricsQueries"
)

var validRollupTypes = map[string]bool{
	"Avg":   true,
	"Sum":   true,
	"Min":   true,
	"Max":   true,
	"Count": true,
}

var validQueryAggregations = map[string]bool{
	"Latest": true,
	"Avg":    true,
	"Sum":    true,
	"Min":    true,
	"Max":    true,
	"Count":  true,
}

func (c *Client) createMetricsQuery(query string, quantization time.Duration, from, to, rollup string) ([]byte, error) {
	metricsQuery := MetricsQuery{
		RowID:        "A",
		Query:        query,
		Quantization: int64(quantization / time.Millisecond),
		Rollup:       rollup,
	}

	timeRange := TimeRange{
		Type: "BeginBoundedTimeRange",
		From: TimeRangeBoundary{
			Type:        "Iso8601TimeRangeBoundary",
			Iso8601Time: from,
		},
		To: TimeRangeBoundary{
			Type:        "Iso8601TimeRangeBoundary",
			Iso8601Time: to,
		},
	}

	requestPayload := MetricsQueryRequest{
		Queries:   []MetricsQuery{metricsQuery},
		TimeRange: timeRange,
	}

	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics query request: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.Host, metricsQueryPath)
	return c.makeRequestWithRetry("POST", url, payload)
}

func (c *Client) createMultiMetricsQuery(queries map[string]string, quantization time.Duration, from, to, rollup string) ([]byte, error) {
	metricsQueries := make([]MetricsQuery, 0)
	for rowID, query := range queries {
		metricsQuery := MetricsQuery{
			RowID:        rowID,
			Query:        query,
			Quantization: int64(quantization / time.Millisecond),
			Rollup:       rollup,
		}
		metricsQueries = append(metricsQueries, metricsQuery)
	}

	timeRange := TimeRange{
		Type: "BeginBoundedTimeRange",
		From: TimeRangeBoundary{
			Type:        "Iso8601TimeRangeBoundary",
			Iso8601Time: from,
		},
		To: TimeRangeBoundary{
			Type:        "Iso8601TimeRangeBoundary",
			Iso8601Time: to,
		},
	}

	requestPayload := MetricsQueryRequest{
		Queries:   metricsQueries,
		TimeRange: timeRange,
	}

	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics query request: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.Host, metricsQueryPath)
	return c.makeRequestWithRetry("POST", url, payload)
}

func (c *Client) parseMetricsQueryResponse(response []byte) (*MetricsQueryResponse, error) {
	var metricsResponse MetricsQueryResponse
	if err := json.Unmarshal(response, &metricsResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics query response: %w", err)
	}

	if metricsResponse.Errors != nil && len(metricsResponse.Errors.Errors) > 0 {
		return nil, fmt.Errorf("query execution failed with errors %v", metricsResponse.Errors.Errors)
	}

	return &metricsResponse, nil
}

func (c *Client) metricsStats(values []float64, dimension string) (*float64, error) {
	if len(values) == 0 {
		return nil, errors.New("no values provided")
	}

	var result float64
	switch dimension {
	case "Latest":
		result = values[len(values)-1]
	case "Sum":
		for _, v := range values {
			result += v
		}
	case "Count":
		result = float64(len(values))
	case "Avg":
		var sum float64
		for _, v := range values {
			sum += v
		}
		result = sum / float64(len(values))
	case "Min":
		minVal := values[0]
		for _, v := range values[1:] {
			if v < minVal {
				minVal = v
			}
		}
		result = minVal
	case "Max":
		maxVal := values[0]
		for _, v := range values[1:] {
			if v > maxVal {
				maxVal = v
			}
		}
		result = maxVal
	default:
		return nil, fmt.Errorf("invalid aggregation '%s', supported values: Latest, Avg, Sum, Count, Min, Max", dimension)
	}

	return &result, nil
}

func IsValidRollupType(rollup string) error {
	if !validRollupTypes[rollup] {
		return fmt.Errorf("invalid rollup value: %s, must be one of Avg, Sum, Min, Max, Count", rollup)
	}
	return nil
}

func IsValidQueryAggregation(aggregation string) error {
	if !validQueryAggregations[aggregation] {
		return fmt.Errorf("invalid aggregation '%s', supported values: Latest, Avg, Sum, Count, Min, Max", aggregation)
	}
	return nil
}

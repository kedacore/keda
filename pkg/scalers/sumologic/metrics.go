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

func (c *Client) createMetricsQuery(query string, quantization time.Duration, from, to string) ([]byte, error) {
	metricsQuery := MetricsQuery{
		RowID:        "A",
		Query:        query,
		Quantization: int64(quantization / time.Millisecond),
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
		return nil, fmt.Errorf("failed to marshal metrics query request: %v", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.Host, metricsQueryPath)
	return c.makeRequest("POST", url, payload)
}

func (c *Client) parseMetricsQueryResponse(response []byte) (*MetricsQueryResponse, error) {
	var metricsResponse MetricsQueryResponse
	if err := json.Unmarshal(response, &metricsResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics query response: %v", err)
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
	case "latest":
		result = values[len(values)-1]
	case "sum":
		for _, v := range values {
			result += v
		}
	case "count":
		result = float64(len(values))
	case "avg":
		var sum float64
		for _, v := range values {
			sum += v
		}
		result = sum / float64(len(values))
	case "min":
		minVal := values[0]
		for _, v := range values[1:] {
			if v < minVal {
				minVal = v
			}
		}
		result = minVal
	case "max":
		maxVal := values[0]
		for _, v := range values[1:] {
			if v > maxVal {
				maxVal = v
			}
		}
		result = maxVal
	default:
		return nil, fmt.Errorf("invalid dimension '%s', supported values: latest, avg, sum, count, min, max", dimension)
	}

	return &result, nil
}

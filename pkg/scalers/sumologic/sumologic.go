package sumologic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

func NewClient(c *Config, sc *scalersconfig.ScalerConfig) (*Client, error) {
	if c.Host == "" {
		return nil, errors.New("host is required")
	}
	if c.AccessID == "" {
		return nil, errors.New("accessID is required")
	}
	if c.AccessKey == "" {
		return nil, errors.New("accessKey is required")
	}

	httpClient := kedautil.CreateHTTPClient(sc.GlobalHTTPTimeout, c.UnsafeSsl)

	client := &Client{
		c,
		httpClient,
	}

	return client, nil
}

func (c *Client) getTimerange(tz string, timerange time.Duration) (string, string, error) {
	location, err := time.LoadLocation(tz)
	if err != nil {
		return "", "", err
	}

	now := time.Now().In(location)
	from := now.Add(-1 * timerange * time.Minute).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	return from, to, nil
}

func (c *Client) makeRequest(method, url string, payload []byte) ([]byte, error) {
	var reqBody io.Reader
	if payload != nil {
		reqBody = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.config.AccessID, c.config.AccessKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error response from server: %s %s %s %s", method, url, respBody, resp.Status)
	}

	return respBody, nil
}

func (c *Client) GetLogSearchResult(query string, timerange time.Duration, tz string) ([]string, error) {
	from, to, err := c.getTimerange(tz, timerange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %v", err)
	}

	payload := []byte(fmt.Sprintf(`{"from":"%s","to":"%s","query":"%s","timeZone":"%s"}`, from, to, query, tz))

	jobID, err := c.createLogSearchJob(payload)
	if err != nil {
		return nil, err
	}
	fmt.Printf("log search job created, id: %s\n", jobID)

	jobStatus, err := c.waitForLogSearchJobCompletion(jobID)
	if err != nil {
		return nil, err
	}

	if jobStatus.RecordCount == 0 {
		return nil, errors.New("only agg queries are supported, please check your query")
	}

	records, err := c.getLogSearchRecords(jobID, jobStatus.RecordCount)
	if err != nil {
		return nil, err
	}

	err = c.deleteLogSearchJob(jobID)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (c *Client) GetMetricsSearchResult(query string, quantization, timerange time.Duration, dimension, tz string) (*float64, error) {
	from, to, err := c.getTimerange(tz, timerange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %w", err)
	}

	resp, err := c.createMetricsQuery(query, quantization, from, to)
	if err != nil {
		return nil, fmt.Errorf("error executing metrics query: %w", err)
	}

	parsedResp, err := c.parseMetricsQueryResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("error parsing metrics query response: %w", err)
	}

	if parsedResp == nil || len(parsedResp.QueryResult) == 0 {
		return nil, errors.New("metrics query response is empty")
	}

	if len(parsedResp.QueryResult[0].TimeSeriesList.TimeSeries) == 0 {
		return nil, errors.New("no time series data found in metrics query response")
	}

	if len(parsedResp.QueryResult[0].TimeSeriesList.TimeSeries) > 1 {
		return nil, errors.New("multiple time series results found, only single time series queries are supported")
	}

	timeseries := parsedResp.QueryResult[0].TimeSeriesList.TimeSeries[0].Points
	if len(timeseries.Timestamps) == 0 || len(timeseries.Values) == 0 {
		return nil, errors.New("metrics query returned empty timestamps or values")
	}

	result, err := c.metricsStats(timeseries.Values, dimension)
	if err != nil {
		return nil, fmt.Errorf("error computing metric stats: %w", err)
	}

	return result, nil
}

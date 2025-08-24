package sumologic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	DefaultQueryAggregator     = "Avg"
	DefaultRollup              = "Avg"
	DefaultLogsPollingInterval = 1 * time.Second
	DefaultMaxRetries          = 3
)

// query types
const (
	Logs    = "logs"
	Metrics = "metrics"
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

	if c.LogsPollingInterval == 0 {
		c.LogsPollingInterval = DefaultLogsPollingInterval
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultMaxRetries
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(sc.GlobalHTTPTimeout, c.UnsafeSsl)
	httpClient.Jar = jar

	logger, err := getLogger(c.LogLevel)
	if err != nil {
		return nil, err
	}

	client := &Client{
		config: c,
		client: httpClient,
		logger: logger,
	}

	return client, nil
}

func getLogger(logLevel string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	switch logLevel {
	case "DEBUG":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "INFO":
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "WARN":
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "ERROR":
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "DPANIC":
		config.Level = zap.NewAtomicLevelAt(zapcore.DPanicLevel)
	case "PANIC":
		config.Level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	case "FATAL":
		config.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel) // Default to INFO
	}

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	return logger, nil
}

func (c *Client) getTimerange(tz string, timerange time.Duration) (string, string, error) {
	location, err := time.LoadLocation(tz)
	if err != nil {
		return "", "", err
	}

	now := time.Now().In(location)
	from := now.Add(-1 * timerange).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	return from, to, nil
}

func (c *Client) makeRequest(method, url string, payload []byte) ([]byte, *http.Response, error) {
	var reqBody io.Reader
	if payload != nil {
		reqBody = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, err
	}

	req.SetBasicAuth(c.config.AccessID, c.config.AccessKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	return respBody, resp, nil
}

func (c *Client) makeRequestWithRetry(method, url string, payload []byte) ([]byte, error) {
	backoff := time.Second

	var lastResp *http.Response
	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		respBody, resp, err := c.makeRequest(method, url, payload)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("error response from server: %s %s %s", method, url, respBody) // non-retryable error
		}

		if resp.StatusCode >= 400 {
			c.logger.Debug("non-OK response from server, retrying",
				zap.String("method", method),
				zap.String("url", url),
				zap.Int("statusCode", resp.StatusCode),
				zap.Int("attempt", attempt+1),
			)
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
			lastResp = resp
			continue
		}
		return respBody, nil
	}
	return nil, fmt.Errorf("request failed after %d attempts with status %s: %s %s", c.config.MaxRetries, lastResp.Status, method, url) // all attempts failed
}

func (c *Client) GetLogSearchResult(query Query) (*float64, error) {
	from, to, err := c.getTimerange(query.Timezone, query.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %w", err)
	}

	jobID, err := c.createLogSearchJob(query.Query, from, to, query.Timezone)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("log search job created", zap.String("jobID", jobID))

	jobStatus, err := c.waitForLogSearchJobCompletion(jobID)
	if err != nil {
		return nil, err
	}

	if jobStatus.MessageCount > 0 && jobStatus.RecordCount == 0 {
		return nil, errors.New("only agg queries are supported, please check your query")
	}

	if jobStatus.RecordCount == 0 {
		zero := float64(0)
		return &zero, nil
	}

	records, err := c.getLogSearchRecords(jobID, jobStatus.RecordCount, query.ResultField)
	if err != nil {
		return nil, err
	}

	err = c.deleteLogSearchJob(jobID)
	if err != nil {
		return nil, err
	}

	result, err := c.metricsStats(records, query.Aggregator)
	if err != nil {
		return nil, fmt.Errorf("error computing metric stats: %w", err)
	}

	return result, nil
}

func (c *Client) GetMetricsSearchResult(query Query) (*float64, error) {
	from, to, err := c.getTimerange(query.Timezone, query.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %w", err)
	}

	resp, err := c.createMetricsQuery(query.Query, query.Quantization, from, to, query.Rollup)
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

	result, err := c.metricsStats(timeseries.Values, query.Aggregator)
	if err != nil {
		return nil, fmt.Errorf("error computing metric stats: %w", err)
	}

	return result, nil
}

func (c *Client) GetMultiMetricsSearchResult(query Query) (*float64, error) {
	from, to, err := c.getTimerange(query.Timezone, query.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time range: %w", err)
	}

	resp, err := c.createMultiMetricsQuery(query.Queries, query.Quantization, from, to, query.Rollup)
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

	var selectedResultSet QueryResult

	for _, queryResult := range parsedResp.QueryResult {
		if queryResult.RowID == query.ResultQueryRowID {
			selectedResultSet = queryResult
			break
		}
	}

	if selectedResultSet.RowID == "" {
		return nil, fmt.Errorf("no query result with matching resultQueryRowID %s found in metrics query response", query.ResultQueryRowID)
	}

	if len(selectedResultSet.TimeSeriesList.TimeSeries) == 0 {
		return nil, errors.New("no time series data found in metrics query response")
	}

	if len(selectedResultSet.TimeSeriesList.TimeSeries) > 1 {
		return nil, errors.New("multiple time series results found, only single time series queries are supported")
	}

	timeseries := selectedResultSet.TimeSeriesList.TimeSeries[0].Points
	if len(timeseries.Timestamps) == 0 || len(timeseries.Values) == 0 {
		return nil, errors.New("metrics query returned empty timestamps or values")
	}

	result, err := c.metricsStats(timeseries.Values, query.Aggregator)
	if err != nil {
		return nil, fmt.Errorf("error computing metric stats: %w", err)
	}

	return result, nil
}

func (c *Client) Close() error {
	if closer, ok := c.client.Transport.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *Client) GetQueryResult(query Query) (float64, error) {
	var result *float64
	var err error

	switch {
	case query.Type == Logs:
		result, err = c.GetLogSearchResult(query)
	case query.Query != "":
		result, err = c.GetMetricsSearchResult(query)
	default:
		result, err = c.GetMultiMetricsSearchResult(query)
	}

	if err != nil {
		return 0, err
	}

	return *result, nil
}

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

func (c *Client) getTimerange(tz string, timerange time.Duration) (string, string) {
	location, err := time.LoadLocation(tz)
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone: %v", err))
	}

	now := time.Now().In(location)
	from := now.Add(-1 * timerange * time.Minute).Format("2006-01-02T15:04:05")
	to := now.Format("2006-01-02T15:04:05")

	return from, to
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error response from server: %s %s %s", method, url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetLogSearchResult(query string, timerange time.Duration, tz string) ([]string, error) {
	from, to := c.getTimerange(tz, timerange)
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

// func (c *Client) GetMetricsSearchResult(jobID string) ( []string, error) {
// }

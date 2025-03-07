package sumologic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	searchJobPath = "api/v1/search/jobs"
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

func (c *Client) createLogSearchJob(payload []byte) (string, error) {
	url := fmt.Sprintf("https://%s/%s", c.config.Host, searchJobPath)

	resp, err := c.makeRequest("POST", url, payload)
	if err != nil {
		return "", err
	}

	var jobResp SearchJobResponse
	if err := json.Unmarshal(resp, &jobResp); err != nil {
		return "", err
	}
	return jobResp.ID, nil
}

func (c *Client) waitForLogSearchJobCompletion(jobID string) (*SearchJobStatus, error) {
	url := fmt.Sprintf("https://%s/%s/%s", c.config.Host, searchJobPath, jobID)

	for {
		resp, err := c.makeRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var status SearchJobStatus
		if err := json.Unmarshal(resp, &status); err != nil {
			return nil, err
		}

		fmt.Printf("log search job state: %s, record count: %d\n", status.State, status.RecordCount)

		if status.State == "DONE GATHERING RESULTS" {
			return &status, nil
		} else if status.State == "CANCELLED" || status.State == "FORCE PAUSED" {
			return nil, fmt.Errorf("search job failed, state: %s", status.State)
		}

		time.Sleep(2 * time.Second)
	}
}

func (c *Client) getLogSearchRecords(jobID string, totalRecords int) ([]string, error) {
	var allRecords []string
	offset := 0
	limit := 10000

	for offset < totalRecords {
		remaining := totalRecords - offset
		if remaining < limit {
			limit = remaining
		}

		url := fmt.Sprintf("https://%s/%s/%s/records?offset=%d&limit=%d", c.config.Host, searchJobPath, jobID, offset, limit)
		resp, err := c.makeRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var recordsResponse RecordsResponse
		if err := json.Unmarshal(resp, &recordsResponse); err != nil {
			return nil, err
		}

		if len(recordsResponse.Records) == 0 {
			break
		}

		for _, record := range recordsResponse.Records {
			if result, exists := record.Map["result"]; exists {
				allRecords = append(allRecords, result)
			}
		}
		offset += limit
	}

	fmt.Printf("log search total records fetched: %d\n", len(allRecords))

	return allRecords, nil
}

func (c *Client) deleteLogSearchJob(jobID string) error {
	url := fmt.Sprintf("https://%s/%s/%s", c.config.Host, searchJobPath, jobID)

	_, err := c.makeRequest("DELETE", url, nil)
	if err == nil {
		fmt.Printf("log search job deleted, id: %s\n", jobID)
	}

	return err
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

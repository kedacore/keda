package sumologic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const (
	searchJobPath = "api/v1/search/jobs"
)

func (c *Client) createLogSearchJob(query, from, to, tz string) (string, error) {
	requestPayload := LogSearchRequest{
		Query:    query,
		From:     from,
		To:       to,
		TimeZone: tz,
	}

	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log search request: %v", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.Host, searchJobPath)
	resp, err := c.makeRequest("POST", url, payload)
	if err != nil {
		return "", err
	}

	var jobResp LogSearchJobResponse
	if err := json.Unmarshal(resp, &jobResp); err != nil {
		return "", err
	}

	return jobResp.ID, nil
}

func (c *Client) waitForLogSearchJobCompletion(jobID string) (*LogSearchJobStatus, error) {
	url := fmt.Sprintf("%s/%s/%s", c.config.Host, searchJobPath, jobID)

	for {
		resp, err := c.makeRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var status LogSearchJobStatus
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

func (c *Client) getLogSearchRecords(jobID string, totalRecords int) ([]float64, error) {
	var allRecords []float64
	offset := 0
	limit := 10000

	for offset < totalRecords {
		remaining := totalRecords - offset
		if remaining < limit {
			limit = remaining
		}

		url := fmt.Sprintf("%s/%s/%s/records?offset=%d&limit=%d", c.config.Host, searchJobPath, jobID, offset, limit)
		resp, err := c.makeRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var recordsResponse LogSearchRecordsResponse
		if err := json.Unmarshal(resp, &recordsResponse); err != nil {
			return nil, err
		}

		if len(recordsResponse.Records) == 0 {
			break
		}

		for _, record := range recordsResponse.Records {
			if result, exists := record.Map["result"]; exists {
				val, err := strconv.ParseFloat(result, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse result value %w", err)
				}
				allRecords = append(allRecords, val)
			}
		}
		offset += limit
	}

	fmt.Printf("log search total records fetched: %d\n", len(allRecords))

	return allRecords, nil
}

func (c *Client) deleteLogSearchJob(jobID string) error {
	url := fmt.Sprintf("%s/%s/%s", c.config.Host, searchJobPath, jobID)

	_, err := c.makeRequest("DELETE", url, nil)
	if err == nil {
		fmt.Printf("log search job deleted, id: %s\n", jobID)
	}

	return err
}

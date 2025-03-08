package sumologic

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	searchJobPath = "api/v1/search/jobs"
)

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

package sumologic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const (
	searchJobPath  = "api/v1/search/jobs"
	stateDone      = "DONE GATHERING RESULTS"
	stateCancelled = "CANCELLED"
	statePaused    = "FORCE PAUSED"
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
		return "", fmt.Errorf("failed to marshal log search request: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.Host, searchJobPath)
	resp, err := c.makeRequestWithRetry("POST", url, payload)
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
		resp, err := c.makeRequestWithRetry("GET", url, nil)
		if err != nil {
			return nil, err
		}

		var status LogSearchJobStatus
		if err := json.Unmarshal(resp, &status); err != nil {
			return nil, err
		}

		c.logger.Debug("log search job state", zap.String("state", status.State), zap.Int("recordCount", status.RecordCount))

		switch status.State {
		case stateDone:
			return &status, nil
		case stateCancelled, statePaused:
			return nil, fmt.Errorf("search job failed, state: %s", status.State)
		}

		time.Sleep(c.config.LogsPollingInterval)
	}
}

func (c *Client) getLogSearchRecords(jobID string, totalRecords int, resultField string) ([]float64, error) {
	var allRecords []float64
	offset := 0
	limit := 10000

	for offset < totalRecords {
		remaining := totalRecords - offset
		if remaining < limit {
			limit = remaining
		}

		url := fmt.Sprintf("%s/%s/%s/records?offset=%d&limit=%d", c.config.Host, searchJobPath, jobID, offset, limit)
		resp, err := c.makeRequestWithRetry("GET", url, nil)
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
			if result, exists := record.Map[resultField]; exists {
				val, err := strconv.ParseFloat(result, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse resultField: %s value %w", resultField, err)
				}
				allRecords = append(allRecords, val)
			}
		}
		offset += limit
	}

	c.logger.Debug("log search total records fetched", zap.Int("totalRecords", len(allRecords)))

	return allRecords, nil
}

func (c *Client) deleteLogSearchJob(jobID string) error {
	url := fmt.Sprintf("%s/%s/%s", c.config.Host, searchJobPath, jobID)

	_, err := c.makeRequestWithRetry("DELETE", url, nil)
	if err == nil {
		c.logger.Debug("log search job deleted", zap.String("jobID", jobID))
	}

	return err
}

//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package driver

import (
	"context"
	"path"
	"strconv"
)

const asyncJobAPI = "_api/job"

type clientAsyncJob struct {
	conn Connection
}

func (c *client) AsyncJob() AsyncJobService {
	return &clientAsyncJob{
		conn: c.conn,
	}
}

func (c *clientAsyncJob) List(ctx context.Context, jobType AsyncJobStatusType, opts *AsyncJobListOptions) ([]string, error) {
	req, err := c.conn.NewRequest("GET", path.Join(asyncJobAPI, pathEscape(string(jobType))))
	if err != nil {
		return nil, WithStack(err)
	}

	if opts != nil && opts.Count != 0 {
		req.SetQuery("count", strconv.Itoa(opts.Count))
	}

	var rawResponse []byte
	ctx = WithRawResponse(ctx, &rawResponse)

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}

	var result []string
	if err = c.conn.Unmarshal(rawResponse, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *clientAsyncJob) Status(ctx context.Context, jobID string) (AsyncJobStatusType, error) {
	req, err := c.conn.NewRequest("GET", path.Join(asyncJobAPI, pathEscape(jobID)))
	if err != nil {
		return "nil", WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}

	switch resp.StatusCode() {
	case 200:
		return JobDone, nil
	case 204:
		return JobPending, nil
	default:
		return "", WithStack(resp.CheckStatus(200, 204))
	}
}

type cancelResponse struct {
	Result bool `json:"result"`
}

func (c *clientAsyncJob) Cancel(ctx context.Context, jobID string) (bool, error) {
	req, err := c.conn.NewRequest("PUT", path.Join(asyncJobAPI, pathEscape(jobID), "cancel"))
	if err != nil {
		return false, WithStack(err)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return false, WithStack(err)
	}

	var data cancelResponse
	if err := resp.ParseBody("", &data); err != nil {
		return false, WithStack(err)
	}
	return data.Result, nil
}

type deleteResponse struct {
	Result bool `json:"result"`
}

func (c *clientAsyncJob) Delete(ctx context.Context, deleteType AsyncJobDeleteType, opts *AsyncJobDeleteOptions) (bool, error) {
	p := ""
	switch deleteType {
	case DeleteAllJobs:
		p = path.Join(asyncJobAPI, pathEscape(string(deleteType)))
	case DeleteExpiredJobs:
		if opts == nil || opts.Stamp.IsZero() {
			return false, WithStack(InvalidArgumentError{Message: "stamp must be set when deleting expired jobs"})
		}
		p = path.Join(asyncJobAPI, pathEscape(string(deleteType)))
	case DeleteSingleJob:
		if opts == nil || opts.JobID == "" {
			return false, WithStack(InvalidArgumentError{Message: "jobID must be set when deleting a single job"})
		}
		p = path.Join(asyncJobAPI, pathEscape(opts.JobID))
	}

	req, err := c.conn.NewRequest("DELETE", p)
	if err != nil {
		return false, WithStack(err)
	}

	if deleteType == DeleteExpiredJobs {
		req.SetQuery("stamp", strconv.FormatInt(opts.Stamp.Unix(), 10))
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}

	if err := resp.CheckStatus(200); err != nil {
		return false, WithStack(err)
	}

	var data deleteResponse
	if err := resp.ParseBody("", &data); err != nil {
		return false, WithStack(err)
	}
	return data.Result, nil
}

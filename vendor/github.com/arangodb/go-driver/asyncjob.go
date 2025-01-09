//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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
	"time"
)

type ClientAsyncJob interface {
	AsyncJob() AsyncJobService
}

// AsyncJobService https://docs.arangodb.com/stable/develop/http-api/jobs/
type AsyncJobService interface {
	// List Returns the ids of job results with a specific status
	List(ctx context.Context, jobType AsyncJobStatusType, opts *AsyncJobListOptions) ([]string, error)
	// Status Returns the status of a specific job
	Status(ctx context.Context, jobID string) (AsyncJobStatusType, error)
	// Cancel Cancels a specific async job
	Cancel(ctx context.Context, jobID string) (bool, error)
	// Delete Deletes async job result
	Delete(ctx context.Context, deleteType AsyncJobDeleteType, opts *AsyncJobDeleteOptions) (bool, error)
}

type AsyncJobStatusType string

const (
	JobDone    AsyncJobStatusType = "done"
	JobPending AsyncJobStatusType = "pending"
)

type AsyncJobListOptions struct {
	// Count The maximum number of ids to return per call.
	// If not specified, a server-defined maximum value will be used.
	Count int `json:"count,omitempty"`
}

type AsyncJobDeleteType string

const (
	DeleteAllJobs     AsyncJobDeleteType = "all"
	DeleteExpiredJobs AsyncJobDeleteType = "expired"
	DeleteSingleJob   AsyncJobDeleteType = "single"
)

type AsyncJobDeleteOptions struct {
	// JobID The id of the job to delete. Works only if type is set to 'single'.
	JobID string `json:"id,omitempty"`

	// Stamp A Unix timestamp specifying the expiration threshold for when the type is set to 'expired'.
	Stamp time.Time `json:"stamp,omitempty"`
}

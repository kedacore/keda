/*
Copyright 2025 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package forgejo

type CountResponse struct {
	Count int64 `json:"count"`
}

type Job struct {
	ID int64 `json:"id"`
	// the repository id
	RepoID int64 `json:"repo_id"`
	// the owner id
	OwnerID int64 `json:"owner_id"`
	// the action run job name
	Name string `json:"name"`
	// the action run job needed ids
	Needs []string `json:"needs"`
	// the action run job labels to run on
	RunsOn []string `json:"runs_on"`
	// the action run job latest task id
	TaskID int64 `json:"task_id"`
	// the action run job status
	Status string `json:"status"`
}

type JobsListResponse struct {
	Jobs []Job `json:"body"`
}

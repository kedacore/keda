//
// DISCLAIMER
//
// Copyright 2022 ArangoDB GmbH, Cologne, Germany
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
	"strings"
)

func (d *database) StartJob(ctx context.Context, options PregelJobOptions) (string, error) {
	id := ""

	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/control_pregel"))
	if err != nil {
		return id, WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return id, WithStack(err)
	}

	var rawResponse []byte
	ctx = WithRawResponse(ctx, &rawResponse)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return id, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return id, WithStack(err)
	}

	return strings.Trim(string(rawResponse), "\""), nil
}

func (d *database) GetJob(ctx context.Context, id string) (*PregelJob, error) {
	escapedId := pathEscape(id)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/control_pregel", escapedId))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data PregelJob
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	return &data, nil
}

func (d *database) GetJobs(ctx context.Context) ([]*PregelJob, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/control_pregel"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}

	var data []*PregelJob
	responses, err := resp.ParseArrayBody()
	if err != nil {
		return nil, WithStack(err)
	}

	for _, response := range responses {
		var job PregelJob
		if err := response.ParseBody("", &job); err != nil {
			return nil, WithStack(err)
		}
		data = append(data, &job)
	}
	return data, nil
}

func (d *database) CancelJob(ctx context.Context, id string) error {
	escapedId := pathEscape(id)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/control_pregel", escapedId))
	if err != nil {
		return WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

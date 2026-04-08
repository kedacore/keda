// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"io"
)

func (h *httpConnect) asyncInsert(ctx context.Context, query string, wait bool, args ...any) error {

	options := queryOptions(ctx)
	options.settings["async_insert"] = 1
	options.settings["wait_for_async_insert"] = 0
	if wait {
		options.settings["wait_for_async_insert"] = 1
	}
	if len(args) > 0 {
		var err error
		query, err = bindQueryOrAppendParameters(true, &options, query, h.location, args...)
		if err != nil {
			return err
		}
	}

	res, err := h.sendQuery(ctx, query, &options, h.headers)
	if res != nil {
		defer res.Body.Close()
		// we don't care about result, so just discard it to reuse connection
		_, _ = io.Copy(io.Discard, res.Body)
	}

	return err
}

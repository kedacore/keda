//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"net/http"
	"path"
)

// newDatabase creates a new Database implementation.
func newDatabase(name string, conn Connection) (Database, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if conn == nil {
		return nil, WithStack(InvalidArgumentError{Message: "conn is nil"})
	}
	return &database{
		name: name,
		conn: conn,
	}, nil
}

// database implements the Database interface.
type database struct {
	name string
	conn Connection
}

// relPath creates the relative path to this database (`_db/<name>`)
func (d *database) relPath() string {
	escapedName := pathEscape(d.name)
	return path.Join("_db", escapedName)
}

// Name returns the name of the database.
func (d *database) Name() string {
	return d.name
}

// Info fetches information about the database.
func (d *database) Info(ctx context.Context) (DatabaseInfo, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/database/current"))
	if err != nil {
		return DatabaseInfo{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return DatabaseInfo{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DatabaseInfo{}, WithStack(err)
	}
	var data DatabaseInfo
	if err := resp.ParseBody("result", &data); err != nil {
		return DatabaseInfo{}, WithStack(err)
	}
	return data, nil
}

// EngineInfo returns information about the database engine being used.
// Note: When your cluster has multiple endpoints (cluster), you will get information
// from the server that is currently being used.
// If you want to know exactly which server the information is from, use a client
// with only a single endpoint and avoid automatic synchronization of endpoints.
func (d *database) EngineInfo(ctx context.Context) (EngineInfo, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/engine"))
	if err != nil {
		return EngineInfo{}, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return EngineInfo{}, WithStack(err)
	}
	if err := resp.CheckStatus(200, 404); err != nil {
		return EngineInfo{}, WithStack(err)
	}
	if resp.StatusCode() == 404 {
		// On version 3.1, this endpoint is not yet supported
		return EngineInfo{Type: EngineTypeMMFiles}, nil
	}
	var data EngineInfo
	if err := resp.ParseBody("", &data); err != nil {
		return EngineInfo{}, WithStack(err)
	}
	return data, nil
}

// Remove removes the entire database.
// If the database does not exist, a NotFoundError is returned.
func (d *database) Remove(ctx context.Context) error {
	req, err := d.conn.NewRequest("DELETE", path.Join("_db/_system/_api/database", pathEscape(d.name)))
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

// Query performs an AQL query, returning a cursor used to iterate over the returned documents.
func (d *database) Query(ctx context.Context, query string, bindVars map[string]interface{}) (Cursor, error) {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/cursor"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := queryRequest{
		Query:    query,
		BindVars: bindVars,
	}
	input.applyContextSettings(ctx)
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return nil, WithStack(err)
	}
	var data cursorData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	col, err := newCursor(data, resp.Endpoint(), d, cs.AllowDirtyReads)
	if err != nil {
		return nil, WithStack(err)
	}
	return col, nil
}

// ValidateQuery validates an AQL query.
// When the query is valid, nil returned, otherwise an error is returned.
// The query is not executed.
func (d *database) ValidateQuery(ctx context.Context, query string) error {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/query"))
	if err != nil {
		return WithStack(err)
	}
	input := parseQueryRequest{
		Query: query,
	}
	if _, err := req.SetBody(input); err != nil {
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

// OptimizerRulesForQueries returns the available optimizer rules for AQL query
// returns an array of objects that contain the name of each available rule and its respective flags.
func (d *database) OptimizerRulesForQueries(ctx context.Context) ([]QueryRule, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/query/rules"))
	if err != nil {
		return []QueryRule{}, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return []QueryRule{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return []QueryRule{}, WithStack(err)
	}

	var data []QueryRule
	responses, err := resp.ParseArrayBody()
	if err != nil {
		return []QueryRule{}, WithStack(err)
	}

	for _, response := range responses {
		var rule QueryRule
		if err := response.ParseBody("", &rule); err != nil {
			return []QueryRule{}, WithStack(err)
		}
		data = append(data, rule)
	}
	return data, nil
}

func (d *database) Transaction(ctx context.Context, action string, options *TransactionOptions) (interface{}, error) {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/transaction"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := transactionRequest{Action: action}
	if options != nil {
		input.MaxTransactionSize = options.MaxTransactionSize
		input.LockTimeout = options.LockTimeout
		input.WaitForSync = options.WaitForSync
		input.IntermediateCommitCount = options.IntermediateCommitCount
		input.Params = options.Params
		input.IntermediateCommitSize = options.IntermediateCommitSize
		input.Collections.Read = options.ReadCollections
		input.Collections.Write = options.WriteCollections
		input.Collections.Exclusive = options.ExclusiveCollections
	}
	if _, err = req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err = resp.CheckStatus(http.StatusOK); err != nil {
		return nil, WithStack(err)
	}

	output := &transactionResponse{}
	if err = resp.ParseBody("", output); err != nil {
		return nil, WithStack(err)
	}

	return output.Result, nil
}

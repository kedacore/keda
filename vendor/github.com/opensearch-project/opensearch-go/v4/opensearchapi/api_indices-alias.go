// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

type aliasClient struct {
	apiClient *Client
}

// Delete executes a delete alias request with the required AliasDeleteReq
func (c aliasClient) Delete(ctx context.Context, req AliasDeleteReq) (*AliasDeleteResp, error) {
	var (
		data AliasDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get alias request with the required AliasGetReq
func (c aliasClient) Get(ctx context.Context, req AliasGetReq) (*AliasGetResp, error) {
	var (
		data AliasGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Indices); err != nil {
		return &data, err
	}

	return &data, nil
}

// Put executes a put alias request with the required AliasPutReq
func (c aliasClient) Put(ctx context.Context, req AliasPutReq) (*AliasPutResp, error) {
	var (
		data AliasPutResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Exists executes an exists alias request with the required AliasExistsReq
func (c aliasClient) Exists(ctx context.Context, req AliasExistsReq) (*opensearch.Response, error) {
	return c.apiClient.do(ctx, req, nil)
}

// AliasDeleteReq represents possible options for the alias delete request
type AliasDeleteReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasDeleteReq) GetRequest() (*http.Request, error) {
	aliases := strings.Join(r.Alias, ",")
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(9 + len(indices) + len(aliases))
	path.WriteString("/")
	path.WriteString(indices)
	path.WriteString("/_alias/")
	path.WriteString(aliases)
	return opensearch.BuildRequest(
		"DELETE",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// AliasDeleteResp represents the returned struct of the alias delete response
type AliasDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasGetReq represents possible options for the alias get request
type AliasGetReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasGetReq) GetRequest() (*http.Request, error) {
	aliases := strings.Join(r.Alias, ",")
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(9 + len(indices) + len(aliases))
	path.WriteString("/")
	path.WriteString(indices)
	path.WriteString("/_alias/")
	path.WriteString(aliases)
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// AliasGetResp represents the returned struct of the alias get response
type AliasGetResp struct {
	Indices map[string]struct {
		Aliases map[string]json.RawMessage `json:"aliases"`
	}
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasPutReq represents possible options for the alias put request
type AliasPutReq struct {
	Indices []string
	Alias   string

	Header http.Header
	Params AliasPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasPutReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(9 + len(indices) + len(r.Alias))
	path.WriteString("/")
	path.WriteString(indices)
	path.WriteString("/_alias/")
	path.WriteString(r.Alias)
	return opensearch.BuildRequest(
		"PUT",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// AliasPutResp represents the returned struct of the alias put response
type AliasPutResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r AliasPutResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// AliasExistsReq represents possible options for the alias exists request
type AliasExistsReq struct {
	Indices []string
	Alias   []string

	Header http.Header
	Params AliasExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r AliasExistsReq) GetRequest() (*http.Request, error) {
	aliases := strings.Join(r.Alias, ",")
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(9 + len(indices) + len(r.Alias))
	path.WriteString("/")
	path.WriteString(indices)
	path.WriteString("/_alias/")
	path.WriteString(aliases)
	return opensearch.BuildRequest(
		"HEAD",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

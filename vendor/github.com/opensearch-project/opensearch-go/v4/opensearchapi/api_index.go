// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// Index executes a /_doc request with the given IndexReq
func (c Client) Index(ctx context.Context, req IndexReq) (*IndexResp, error) {
	var (
		data IndexResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// IndexReq represents possible options for the /_doc request
type IndexReq struct {
	Index      string
	DocumentID string
	Body       io.Reader
	Header     http.Header
	Params     IndexParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexReq) GetRequest() (*http.Request, error) {
	var method, path string

	if r.DocumentID != "" {
		method = "PUT"
		path = fmt.Sprintf("/%s/_doc/%s", r.Index, r.DocumentID)
	} else {
		method = "POST"
		path = fmt.Sprintf("/%s/_doc", r.Index)
	}

	return opensearch.BuildRequest(
		method,
		path,
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndexResp represents the returned struct of the /_doc response
type IndexResp struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string `json:"result"`
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	SeqNo       int    `json:"_seq_no"`
	PrimaryTerm int    `json:"_primary_term"`
	Type        string `json:"_type"` // Deprecated field
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

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

// Update executes a /_update request with the optional UpdateReq
func (c Client) Update(ctx context.Context, req UpdateReq) (*UpdateResp, error) {
	var (
		data UpdateResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// UpdateReq represents possible options for the /_update request
type UpdateReq struct {
	Index      string
	DocumentID string

	Body io.Reader

	Header http.Header
	Params UpdateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r UpdateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_update/%s", r.Index, r.DocumentID),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// UpdateResp represents the returned struct of the /_update response
type UpdateResp struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string `json:"result"`
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	SeqNo       int              `json:"_seq_no"`
	PrimaryTerm int              `json:"_primary_term"`
	Type        string           `json:"_type"` // Deprecated field
	Get         *DocumentGetResp `json:"get,omitempty"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r UpdateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

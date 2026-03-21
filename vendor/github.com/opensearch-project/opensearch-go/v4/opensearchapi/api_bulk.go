// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// Bulk executes a /_bulk request with the needed BulkReq
func (c Client) Bulk(ctx context.Context, req BulkReq) (*BulkResp, error) {
	var (
		data BulkResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// BulkReq represents possible options for the /_bulk request
type BulkReq struct {
	Index  string
	Body   io.Reader
	Header http.Header
	Params BulkParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r BulkReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	//nolint:gomnd // 7 is the max number of static chars
	path.Grow(7 + len(r.Index))

	if len(r.Index) > 0 {
		path.WriteString("/")
		path.WriteString(r.Index)
	}

	path.WriteString("/_bulk")

	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// BulkResp represents the returned struct of the /_bulk response
type BulkResp struct {
	Took     int                       `json:"took"`
	Errors   bool                      `json:"errors"`
	Items    []map[string]BulkRespItem `json:"items"`
	response *opensearch.Response
}

// BulkRespItem represents an item of the BulkResp
type BulkRespItem struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Type    string `json:"_type"` // Deprecated field
	Result  string `json:"result"`
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	SeqNo       int `json:"_seq_no"`
	PrimaryTerm int `json:"_primary_term"`
	Status      int `json:"status"`
	Error       *struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
		Cause  struct {
			Type        string    `json:"type"`
			Reason      string    `json:"reason"`
			ScriptStack *[]string `json:"script_stack,omitempty"`
			Script      *string   `json:"script,omitempty"`
			Lang        *string   `json:"lang,omitempty"`
			Position    *struct {
				Offset int `json:"offset"`
				Start  int `json:"start"`
				End    int `json:"end"`
			} `json:"position,omitempty"`
			Cause *struct {
				Type   string  `json:"type"`
				Reason *string `json:"reason"`
			} `json:"caused_by"`
		} `json:"caused_by,omitempty"`
	} `json:"error,omitempty"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r BulkResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

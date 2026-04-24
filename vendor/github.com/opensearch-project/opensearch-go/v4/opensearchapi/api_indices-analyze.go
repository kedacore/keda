// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesAnalyzeReq represents possible options for the <indices>/_analyze request
type IndicesAnalyzeReq struct {
	Index string
	Body  IndicesAnalyzeBody

	Header http.Header
	Params IndicesAnalyzeParams
}

// IndicesAnalyzeBody represents the request body for the indices analyze request
type IndicesAnalyzeBody struct {
	Analyzer   string   `json:"analyzer,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
	CharFilter []string `json:"char_filter,omitempty"`
	Explain    bool     `json:"explain,omitempty"`
	Field      string   `json:"field,omitempty"`
	Filter     []string `json:"filter,omitempty"`
	Normalizer string   `json:"normalizer,omitempty"`
	Text       []string `json:"text"`
	Tokenizer  string   `json:"tokenizer,omitempty"`
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesAnalyzeReq) GetRequest() (*http.Request, error) {
	body, err := json.Marshal(r.Body)
	if err != nil {
		return nil, err
	}

	var path strings.Builder
	path.Grow(10 + len(r.Index))
	if len(r.Index) != 0 {
		path.WriteString("/")
		path.WriteString(r.Index)
	}
	path.WriteString("/_analyze")
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		bytes.NewReader(body),
		r.Params.get(),
		r.Header,
	)
}

// IndicesAnalyzeResp represents the returned struct of the index create response
type IndicesAnalyzeResp struct {
	Tokens []IndicesAnalyzeToken `json:"tokens"`
	Detail struct {
		CustomAnalyzer bool                         `json:"custom_analyzer"`
		Charfilters    []IndicesAnalyzeCharfilter   `json:"charfilters"`
		Tokenizer      IndicesAnalyzeTokenizer      `json:"tokenizer"`
		Tokenfilters   []IndicesAnalyzeTokenfilters `json:"tokenfilters"`
		Analyzer       IndicesAnalyzeInfo           `json:"analyzer"`
	} `json:"detail"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesAnalyzeResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// IndicesAnalyzeToken is a sut type of IndicesAnalyzeResp containing information about analyzer token
type IndicesAnalyzeToken struct {
	Token       string `json:"token"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Type        string `json:"type"`
	Position    int    `json:"position"`
}

// IndicesAnalyzeTokenizer is a sub type of IndicesAnalyzerResp containing information about the tokenizer name and tokens
type IndicesAnalyzeTokenizer struct {
	Name   string                `json:"name"`
	Tokens []IndicesAnalyzeToken `json:"tokens"`
}

// IndicesAnalyzeTokenfilters is a sub type of IndicesAnalyzerResp containing information about the token filers name and tokens
type IndicesAnalyzeTokenfilters struct {
	Name   string `json:"name"`
	Tokens []struct {
		Token       string `json:"token"`
		StartOffset int    `json:"start_offset"`
		EndOffset   int    `json:"end_offset"`
		Type        string `json:"type"`
		Position    int    `json:"position"`
		Keyword     bool   `json:"keyword"`
	} `json:"tokens"`
}

// IndicesAnalyzeCharfilter is a sub type of IndicesAnalyzerResp containing information about the char filter name and filtered text
type IndicesAnalyzeCharfilter struct {
	Name         string   `json:"name"`
	FilteredText []string `json:"filtered_text"`
}

// IndicesAnalyzeInfo is a sub type of IndicesAnalyzerResp containing information about the analyzer name and tokens
type IndicesAnalyzeInfo struct {
	Name   string `json:"name"`
	Tokens []struct {
		Token          string `json:"token"`
		StartOffset    int    `json:"start_offset"`
		EndOffset      int    `json:"end_offset"`
		Type           string `json:"type"`
		Position       int    `json:"position"`
		Bytes          string `json:"bytes"`
		PositionLength int    `json:"positionLength"`
		TermFrequency  int    `json:"termFrequency"`
	} `json:"tokens"`
}

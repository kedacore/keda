// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type scriptClient struct {
	apiClient *Client
}

// Delete executes a delete script request with the required ScriptDeleteReq
func (c scriptClient) Delete(ctx context.Context, req ScriptDeleteReq) (*ScriptDeleteResp, error) {
	var (
		data ScriptDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Put executes an put script request with the required ScriptPutReq
func (c scriptClient) Put(ctx context.Context, req ScriptPutReq) (*ScriptPutResp, error) {
	var (
		data ScriptPutResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a /_script request with the required ScriptGetReq
func (c scriptClient) Get(ctx context.Context, req ScriptGetReq) (*ScriptGetResp, error) {
	var (
		data ScriptGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Context executes a /_script_context request with the optional ScriptContextReq
func (c scriptClient) Context(ctx context.Context, req *ScriptContextReq) (*ScriptContextResp, error) {
	if req == nil {
		req = &ScriptContextReq{}
	}

	var (
		data ScriptContextResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Language executes a /_script_context request with the optional ScriptLanguageReq
func (c scriptClient) Language(ctx context.Context, req *ScriptLanguageReq) (*ScriptLanguageResp, error) {
	if req == nil {
		req = &ScriptLanguageReq{}
	}

	var (
		data ScriptLanguageResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// PainlessExecute executes a /_script request with the required ScriptPainlessExecuteReq
func (c scriptClient) PainlessExecute(ctx context.Context, req ScriptPainlessExecuteReq) (*ScriptPainlessExecuteResp, error) {
	var (
		data ScriptPainlessExecuteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

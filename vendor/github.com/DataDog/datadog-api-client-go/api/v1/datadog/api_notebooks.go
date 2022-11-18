// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"bytes"
	_context "context"
	_ioutil "io/ioutil"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"
)

// NotebooksApiService NotebooksApi service.
type NotebooksApiService service

type apiCreateNotebookRequest struct {
	ctx        _context.Context
	ApiService *NotebooksApiService
	body       *NotebookCreateRequest
}

func (a *NotebooksApiService) buildCreateNotebookRequest(ctx _context.Context, body NotebookCreateRequest) (apiCreateNotebookRequest, error) {
	req := apiCreateNotebookRequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}
	return req, nil
}

// CreateNotebook Create a notebook.
// Create a notebook using the specified options.
func (a *NotebooksApiService) CreateNotebook(ctx _context.Context, body NotebookCreateRequest) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateNotebookRequest(ctx, body)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.createNotebookExecute(req)
}

// createNotebookExecute executes the request.
func (a *NotebooksApiService) createNotebookExecute(r apiCreateNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "NotebooksApiService.CreateNotebook")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, reportError("body is required and must be specified")
	}

	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{"application/json"}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}

	// body params
	localVarPostBody = r.body
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["apiKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-API-KEY"] = key
			}
		}
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["appKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-APPLICATION-KEY"] = key
			}
		}
	}
	req, err := a.client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := _ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = _ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 403 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type apiDeleteNotebookRequest struct {
	ctx        _context.Context
	ApiService *NotebooksApiService
	notebookId int64
}

func (a *NotebooksApiService) buildDeleteNotebookRequest(ctx _context.Context, notebookId int64) (apiDeleteNotebookRequest, error) {
	req := apiDeleteNotebookRequest{
		ApiService: a,
		ctx:        ctx,
		notebookId: notebookId,
	}
	return req, nil
}

// DeleteNotebook Delete a notebook.
// Delete a notebook using the specified ID.
func (a *NotebooksApiService) DeleteNotebook(ctx _context.Context, notebookId int64) (*_nethttp.Response, error) {
	req, err := a.buildDeleteNotebookRequest(ctx, notebookId)
	if err != nil {
		return nil, err
	}

	return req.ApiService.deleteNotebookExecute(req)
}

// deleteNotebookExecute executes the request.
func (a *NotebooksApiService) deleteNotebookExecute(r apiDeleteNotebookRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "NotebooksApiService.DeleteNotebook")
	if err != nil {
		return nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(parameterToString(r.notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"*/*"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["apiKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-API-KEY"] = key
			}
		}
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["appKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-APPLICATION-KEY"] = key
			}
		}
	}
	req, err := a.client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return nil, err
	}

	localVarHTTPResponse, err := a.client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarHTTPResponse, err
	}

	localVarBody, err := _ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = _ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 403 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 404 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarHTTPResponse, newErr
	}

	return localVarHTTPResponse, nil
}

type apiGetNotebookRequest struct {
	ctx        _context.Context
	ApiService *NotebooksApiService
	notebookId int64
}

func (a *NotebooksApiService) buildGetNotebookRequest(ctx _context.Context, notebookId int64) (apiGetNotebookRequest, error) {
	req := apiGetNotebookRequest{
		ApiService: a,
		ctx:        ctx,
		notebookId: notebookId,
	}
	return req, nil
}

// GetNotebook Get a notebook.
// Get a notebook using the specified notebook ID.
func (a *NotebooksApiService) GetNotebook(ctx _context.Context, notebookId int64) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildGetNotebookRequest(ctx, notebookId)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getNotebookExecute(req)
}

// getNotebookExecute executes the request.
func (a *NotebooksApiService) getNotebookExecute(r apiGetNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "NotebooksApiService.GetNotebook")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(parameterToString(r.notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["apiKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-API-KEY"] = key
			}
		}
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["appKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-APPLICATION-KEY"] = key
			}
		}
	}
	req, err := a.client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := _ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = _ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 403 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 404 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type apiListNotebooksRequest struct {
	ctx                 _context.Context
	ApiService          *NotebooksApiService
	authorHandle        *string
	excludeAuthorHandle *string
	start               *int64
	count               *int64
	sortField           *string
	sortDir             *string
	query               *string
	includeCells        *bool
	isTemplate          *bool
	typeVar             *string
}

// ListNotebooksOptionalParameters holds optional parameters for ListNotebooks.
type ListNotebooksOptionalParameters struct {
	AuthorHandle        *string
	ExcludeAuthorHandle *string
	Start               *int64
	Count               *int64
	SortField           *string
	SortDir             *string
	Query               *string
	IncludeCells        *bool
	IsTemplate          *bool
	Type                *string
}

// NewListNotebooksOptionalParameters creates an empty struct for parameters.
func NewListNotebooksOptionalParameters() *ListNotebooksOptionalParameters {
	this := ListNotebooksOptionalParameters{}
	return &this
}

// WithAuthorHandle sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithAuthorHandle(authorHandle string) *ListNotebooksOptionalParameters {
	r.AuthorHandle = &authorHandle
	return r
}

// WithExcludeAuthorHandle sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithExcludeAuthorHandle(excludeAuthorHandle string) *ListNotebooksOptionalParameters {
	r.ExcludeAuthorHandle = &excludeAuthorHandle
	return r
}

// WithStart sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithStart(start int64) *ListNotebooksOptionalParameters {
	r.Start = &start
	return r
}

// WithCount sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithCount(count int64) *ListNotebooksOptionalParameters {
	r.Count = &count
	return r
}

// WithSortField sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithSortField(sortField string) *ListNotebooksOptionalParameters {
	r.SortField = &sortField
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithSortDir(sortDir string) *ListNotebooksOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithQuery(query string) *ListNotebooksOptionalParameters {
	r.Query = &query
	return r
}

// WithIncludeCells sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithIncludeCells(includeCells bool) *ListNotebooksOptionalParameters {
	r.IncludeCells = &includeCells
	return r
}

// WithIsTemplate sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithIsTemplate(isTemplate bool) *ListNotebooksOptionalParameters {
	r.IsTemplate = &isTemplate
	return r
}

// WithType sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithType(typeVar string) *ListNotebooksOptionalParameters {
	r.Type = &typeVar
	return r
}

func (a *NotebooksApiService) buildListNotebooksRequest(ctx _context.Context, o ...ListNotebooksOptionalParameters) (apiListNotebooksRequest, error) {
	req := apiListNotebooksRequest{
		ApiService: a,
		ctx:        ctx,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type ListNotebooksOptionalParameters is allowed")
	}

	if o != nil {
		req.authorHandle = o[0].AuthorHandle
		req.excludeAuthorHandle = o[0].ExcludeAuthorHandle
		req.start = o[0].Start
		req.count = o[0].Count
		req.sortField = o[0].SortField
		req.sortDir = o[0].SortDir
		req.query = o[0].Query
		req.includeCells = o[0].IncludeCells
		req.isTemplate = o[0].IsTemplate
		req.typeVar = o[0].Type
	}
	return req, nil
}

// ListNotebooks Get all notebooks.
// Get all notebooks. This can also be used to search for notebooks with a particular `query` in the notebook
// `name` or author `handle`.
func (a *NotebooksApiService) ListNotebooks(ctx _context.Context, o ...ListNotebooksOptionalParameters) (NotebooksResponse, *_nethttp.Response, error) {
	req, err := a.buildListNotebooksRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue NotebooksResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.listNotebooksExecute(req)
}

// listNotebooksExecute executes the request.
func (a *NotebooksApiService) listNotebooksExecute(r apiListNotebooksRequest) (NotebooksResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebooksResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "NotebooksApiService.ListNotebooks")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.authorHandle != nil {
		localVarQueryParams.Add("author_handle", parameterToString(*r.authorHandle, ""))
	}
	if r.excludeAuthorHandle != nil {
		localVarQueryParams.Add("exclude_author_handle", parameterToString(*r.excludeAuthorHandle, ""))
	}
	if r.start != nil {
		localVarQueryParams.Add("start", parameterToString(*r.start, ""))
	}
	if r.count != nil {
		localVarQueryParams.Add("count", parameterToString(*r.count, ""))
	}
	if r.sortField != nil {
		localVarQueryParams.Add("sort_field", parameterToString(*r.sortField, ""))
	}
	if r.sortDir != nil {
		localVarQueryParams.Add("sort_dir", parameterToString(*r.sortDir, ""))
	}
	if r.query != nil {
		localVarQueryParams.Add("query", parameterToString(*r.query, ""))
	}
	if r.includeCells != nil {
		localVarQueryParams.Add("include_cells", parameterToString(*r.includeCells, ""))
	}
	if r.isTemplate != nil {
		localVarQueryParams.Add("is_template", parameterToString(*r.isTemplate, ""))
	}
	if r.typeVar != nil {
		localVarQueryParams.Add("type", parameterToString(*r.typeVar, ""))
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["apiKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-API-KEY"] = key
			}
		}
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["appKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-APPLICATION-KEY"] = key
			}
		}
	}
	req, err := a.client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := _ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = _ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 403 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type apiUpdateNotebookRequest struct {
	ctx        _context.Context
	ApiService *NotebooksApiService
	notebookId int64
	body       *NotebookUpdateRequest
}

func (a *NotebooksApiService) buildUpdateNotebookRequest(ctx _context.Context, notebookId int64, body NotebookUpdateRequest) (apiUpdateNotebookRequest, error) {
	req := apiUpdateNotebookRequest{
		ApiService: a,
		ctx:        ctx,
		notebookId: notebookId,
		body:       &body,
	}
	return req, nil
}

// UpdateNotebook Update a notebook.
// Update a notebook using the specified ID.
func (a *NotebooksApiService) UpdateNotebook(ctx _context.Context, notebookId int64, body NotebookUpdateRequest) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateNotebookRequest(ctx, notebookId, body)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.updateNotebookExecute(req)
}

// updateNotebookExecute executes the request.
func (a *NotebooksApiService) updateNotebookExecute(r apiUpdateNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "NotebooksApiService.UpdateNotebook")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(parameterToString(r.notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, reportError("body is required and must be specified")
	}

	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{"application/json"}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}

	// body params
	localVarPostBody = r.body
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["apiKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-API-KEY"] = key
			}
		}
	}
	if r.ctx != nil {
		// API Key Authentication
		if auth, ok := r.ctx.Value(ContextAPIKeys).(map[string]APIKey); ok {
			if apiKey, ok := auth["appKeyAuth"]; ok {
				var key string
				if apiKey.Prefix != "" {
					key = apiKey.Prefix + " " + apiKey.Key
				} else {
					key = apiKey.Key
				}
				localVarHeaderParams["DD-APPLICATION-KEY"] = key
			}
		}
	}
	req, err := a.client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := _ioutil.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = _ioutil.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 403 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 404 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 409 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.client.decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

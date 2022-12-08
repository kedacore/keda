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
)

// LogsApiService LogsApi service.
type LogsApiService service

type apiListLogsRequest struct {
	ctx        _context.Context
	ApiService *LogsApiService
	body       *LogsListRequest
}

func (a *LogsApiService) buildListLogsRequest(ctx _context.Context, body LogsListRequest) (apiListLogsRequest, error) {
	req := apiListLogsRequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}
	return req, nil
}

// ListLogs Search logs.
// List endpoint returns logs that match a log search query.
// [Results are paginated][1].
//
// **If you are considering archiving logs for your organization,
// consider use of the Datadog archive capabilities instead of the log list API.
// See [Datadog Logs Archive documentation][2].**
//
// [1]: /logs/guide/collect-multiple-logs-with-pagination
// [2]: https://docs.datadoghq.com/logs/archives
func (a *LogsApiService) ListLogs(ctx _context.Context, body LogsListRequest) (LogsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListLogsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue LogsListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.listLogsExecute(req)
}

// listLogsExecute executes the request.
func (a *LogsApiService) listLogsExecute(r apiListLogsRequest) (LogsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "LogsApiService.ListLogs")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/logs-queries/list"

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
			var v LogsAPIErrorResponse
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

type apiSubmitLogRequest struct {
	ctx             _context.Context
	ApiService      *LogsApiService
	body            *[]HTTPLogItem
	contentEncoding *ContentEncoding
	ddtags          *string
}

// SubmitLogOptionalParameters holds optional parameters for SubmitLog.
type SubmitLogOptionalParameters struct {
	ContentEncoding *ContentEncoding
	Ddtags          *string
}

// NewSubmitLogOptionalParameters creates an empty struct for parameters.
func NewSubmitLogOptionalParameters() *SubmitLogOptionalParameters {
	this := SubmitLogOptionalParameters{}
	return &this
}

// WithContentEncoding sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithContentEncoding(contentEncoding ContentEncoding) *SubmitLogOptionalParameters {
	r.ContentEncoding = &contentEncoding
	return r
}

// WithDdtags sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithDdtags(ddtags string) *SubmitLogOptionalParameters {
	r.Ddtags = &ddtags
	return r
}

func (a *LogsApiService) buildSubmitLogRequest(ctx _context.Context, body []HTTPLogItem, o ...SubmitLogOptionalParameters) (apiSubmitLogRequest, error) {
	req := apiSubmitLogRequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type SubmitLogOptionalParameters is allowed")
	}

	if o != nil {
		req.contentEncoding = o[0].ContentEncoding
		req.ddtags = o[0].Ddtags
	}
	return req, nil
}

// SubmitLog Send logs.
// Send your logs to your Datadog platform over HTTP. Limits per HTTP request are:
//
// - Maximum content size per payload (uncompressed): 5MB
// - Maximum size for a single log: 1MB
// - Maximum array size if sending multiple logs in an array: 1000 entries
//
// Any log exceeding 1MB is accepted and truncated by Datadog:
// - For a single log request, the API truncates the log at 1MB and returns a 2xx.
// - For a multi-logs request, the API processes all logs, truncates only logs larger than 1MB, and returns a 2xx.
//
// Datadog recommends sending your logs compressed.
// Add the `Content-Encoding: gzip` header to the request when sending compressed logs.
//
// The status codes answered by the HTTP API are:
// - 200: OK
// - 400: Bad request (likely an issue in the payload formatting)
// - 403: Permission issue (likely using an invalid API Key)
// - 413: Payload too large (batch is above 5MB uncompressed)
// - 5xx: Internal error, request should be retried after some time
func (a *LogsApiService) SubmitLog(ctx _context.Context, body []HTTPLogItem, o ...SubmitLogOptionalParameters) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildSubmitLogRequest(ctx, body, o...)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return req.ApiService.submitLogExecute(req)
}

// submitLogExecute executes the request.
func (a *LogsApiService) submitLogExecute(r apiSubmitLogRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "LogsApiService.SubmitLog")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/v1/input"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, reportError("body is required and must be specified")
	}
	if r.ddtags != nil {
		localVarQueryParams.Add("ddtags", parameterToString(*r.ddtags, ""))
	}

	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{"application/json", "application/json;simple", "application/logplex-1", "text/plain"}

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

	if r.contentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = parameterToString(*r.contentEncoding, "")
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
			var v HTTPLogError
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

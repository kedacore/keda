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

// HostsApiService HostsApi service.
type HostsApiService service

type apiGetHostTotalsRequest struct {
	ctx        _context.Context
	ApiService *HostsApiService
	from       *int64
}

// GetHostTotalsOptionalParameters holds optional parameters for GetHostTotals.
type GetHostTotalsOptionalParameters struct {
	From *int64
}

// NewGetHostTotalsOptionalParameters creates an empty struct for parameters.
func NewGetHostTotalsOptionalParameters() *GetHostTotalsOptionalParameters {
	this := GetHostTotalsOptionalParameters{}
	return &this
}

// WithFrom sets the corresponding parameter name and returns the struct.
func (r *GetHostTotalsOptionalParameters) WithFrom(from int64) *GetHostTotalsOptionalParameters {
	r.From = &from
	return r
}

func (a *HostsApiService) buildGetHostTotalsRequest(ctx _context.Context, o ...GetHostTotalsOptionalParameters) (apiGetHostTotalsRequest, error) {
	req := apiGetHostTotalsRequest{
		ApiService: a,
		ctx:        ctx,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type GetHostTotalsOptionalParameters is allowed")
	}

	if o != nil {
		req.from = o[0].From
	}
	return req, nil
}

// GetHostTotals Get the total number of active hosts.
// This endpoint returns the total number of active and up hosts in your Datadog account.
// Active means the host has reported in the past hour, and up means it has reported in the past two hours.
func (a *HostsApiService) GetHostTotals(ctx _context.Context, o ...GetHostTotalsOptionalParameters) (HostTotals, *_nethttp.Response, error) {
	req, err := a.buildGetHostTotalsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue HostTotals
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getHostTotalsExecute(req)
}

// getHostTotalsExecute executes the request.
func (a *HostsApiService) getHostTotalsExecute(r apiGetHostTotalsRequest) (HostTotals, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HostTotals
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "HostsApiService.GetHostTotals")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/hosts/totals"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.from != nil {
		localVarQueryParams.Add("from", parameterToString(*r.from, ""))
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

type apiListHostsRequest struct {
	ctx                   _context.Context
	ApiService            *HostsApiService
	filter                *string
	sortField             *string
	sortDir               *string
	start                 *int64
	count                 *int64
	from                  *int64
	includeMutedHostsData *bool
	includeHostsMetadata  *bool
}

// ListHostsOptionalParameters holds optional parameters for ListHosts.
type ListHostsOptionalParameters struct {
	Filter                *string
	SortField             *string
	SortDir               *string
	Start                 *int64
	Count                 *int64
	From                  *int64
	IncludeMutedHostsData *bool
	IncludeHostsMetadata  *bool
}

// NewListHostsOptionalParameters creates an empty struct for parameters.
func NewListHostsOptionalParameters() *ListHostsOptionalParameters {
	this := ListHostsOptionalParameters{}
	return &this
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithFilter(filter string) *ListHostsOptionalParameters {
	r.Filter = &filter
	return r
}

// WithSortField sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithSortField(sortField string) *ListHostsOptionalParameters {
	r.SortField = &sortField
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithSortDir(sortDir string) *ListHostsOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithStart sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithStart(start int64) *ListHostsOptionalParameters {
	r.Start = &start
	return r
}

// WithCount sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithCount(count int64) *ListHostsOptionalParameters {
	r.Count = &count
	return r
}

// WithFrom sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithFrom(from int64) *ListHostsOptionalParameters {
	r.From = &from
	return r
}

// WithIncludeMutedHostsData sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithIncludeMutedHostsData(includeMutedHostsData bool) *ListHostsOptionalParameters {
	r.IncludeMutedHostsData = &includeMutedHostsData
	return r
}

// WithIncludeHostsMetadata sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithIncludeHostsMetadata(includeHostsMetadata bool) *ListHostsOptionalParameters {
	r.IncludeHostsMetadata = &includeHostsMetadata
	return r
}

func (a *HostsApiService) buildListHostsRequest(ctx _context.Context, o ...ListHostsOptionalParameters) (apiListHostsRequest, error) {
	req := apiListHostsRequest{
		ApiService: a,
		ctx:        ctx,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type ListHostsOptionalParameters is allowed")
	}

	if o != nil {
		req.filter = o[0].Filter
		req.sortField = o[0].SortField
		req.sortDir = o[0].SortDir
		req.start = o[0].Start
		req.count = o[0].Count
		req.from = o[0].From
		req.includeMutedHostsData = o[0].IncludeMutedHostsData
		req.includeHostsMetadata = o[0].IncludeHostsMetadata
	}
	return req, nil
}

// ListHosts Get all hosts for your organization.
// This endpoint allows searching for hosts by name, alias, or tag.
// Hosts live within the past 3 hours are included by default.
// Retention is 7 days.
// Results are paginated with a max of 1000 results at a time.
func (a *HostsApiService) ListHosts(ctx _context.Context, o ...ListHostsOptionalParameters) (HostListResponse, *_nethttp.Response, error) {
	req, err := a.buildListHostsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue HostListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.listHostsExecute(req)
}

// listHostsExecute executes the request.
func (a *HostsApiService) listHostsExecute(r apiListHostsRequest) (HostListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HostListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "HostsApiService.ListHosts")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/hosts"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filter != nil {
		localVarQueryParams.Add("filter", parameterToString(*r.filter, ""))
	}
	if r.sortField != nil {
		localVarQueryParams.Add("sort_field", parameterToString(*r.sortField, ""))
	}
	if r.sortDir != nil {
		localVarQueryParams.Add("sort_dir", parameterToString(*r.sortDir, ""))
	}
	if r.start != nil {
		localVarQueryParams.Add("start", parameterToString(*r.start, ""))
	}
	if r.count != nil {
		localVarQueryParams.Add("count", parameterToString(*r.count, ""))
	}
	if r.from != nil {
		localVarQueryParams.Add("from", parameterToString(*r.from, ""))
	}
	if r.includeMutedHostsData != nil {
		localVarQueryParams.Add("include_muted_hosts_data", parameterToString(*r.includeMutedHostsData, ""))
	}
	if r.includeHostsMetadata != nil {
		localVarQueryParams.Add("include_hosts_metadata", parameterToString(*r.includeHostsMetadata, ""))
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

type apiMuteHostRequest struct {
	ctx        _context.Context
	ApiService *HostsApiService
	hostName   string
	body       *HostMuteSettings
}

func (a *HostsApiService) buildMuteHostRequest(ctx _context.Context, hostName string, body HostMuteSettings) (apiMuteHostRequest, error) {
	req := apiMuteHostRequest{
		ApiService: a,
		ctx:        ctx,
		hostName:   hostName,
		body:       &body,
	}
	return req, nil
}

// MuteHost Mute a host.
// Mute a host.
func (a *HostsApiService) MuteHost(ctx _context.Context, hostName string, body HostMuteSettings) (HostMuteResponse, *_nethttp.Response, error) {
	req, err := a.buildMuteHostRequest(ctx, hostName, body)
	if err != nil {
		var localVarReturnValue HostMuteResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.muteHostExecute(req)
}

// muteHostExecute executes the request.
func (a *HostsApiService) muteHostExecute(r apiMuteHostRequest) (HostMuteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue HostMuteResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "HostsApiService.MuteHost")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/host/{host_name}/mute"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(parameterToString(r.hostName, "")), -1)

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

type apiUnmuteHostRequest struct {
	ctx        _context.Context
	ApiService *HostsApiService
	hostName   string
}

func (a *HostsApiService) buildUnmuteHostRequest(ctx _context.Context, hostName string) (apiUnmuteHostRequest, error) {
	req := apiUnmuteHostRequest{
		ApiService: a,
		ctx:        ctx,
		hostName:   hostName,
	}
	return req, nil
}

// UnmuteHost Unmute a host.
// Unmutes a host. This endpoint takes no JSON arguments.
func (a *HostsApiService) UnmuteHost(ctx _context.Context, hostName string) (HostMuteResponse, *_nethttp.Response, error) {
	req, err := a.buildUnmuteHostRequest(ctx, hostName)
	if err != nil {
		var localVarReturnValue HostMuteResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.unmuteHostExecute(req)
}

// unmuteHostExecute executes the request.
func (a *HostsApiService) unmuteHostExecute(r apiUnmuteHostRequest) (HostMuteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue HostMuteResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "HostsApiService.UnmuteHost")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/host/{host_name}/unmute"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(parameterToString(r.hostName, "")), -1)

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

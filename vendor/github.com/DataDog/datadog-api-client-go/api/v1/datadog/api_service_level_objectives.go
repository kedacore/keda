// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"bytes"
	_context "context"
	_fmt "fmt"
	_ioutil "io/ioutil"
	_log "log"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"
)

// ServiceLevelObjectivesApiService ServiceLevelObjectivesApi service.
type ServiceLevelObjectivesApiService service

type apiCheckCanDeleteSLORequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	ids        *string
}

func (a *ServiceLevelObjectivesApiService) buildCheckCanDeleteSLORequest(ctx _context.Context, ids string) (apiCheckCanDeleteSLORequest, error) {
	req := apiCheckCanDeleteSLORequest{
		ApiService: a,
		ctx:        ctx,
		ids:        &ids,
	}
	return req, nil
}

// CheckCanDeleteSLO Check if SLOs can be safely deleted.
// Check if an SLO can be safely deleted. For example,
// assure an SLO can be deleted without disrupting a dashboard.
func (a *ServiceLevelObjectivesApiService) CheckCanDeleteSLO(ctx _context.Context, ids string) (CheckCanDeleteSLOResponse, *_nethttp.Response, error) {
	req, err := a.buildCheckCanDeleteSLORequest(ctx, ids)
	if err != nil {
		var localVarReturnValue CheckCanDeleteSLOResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.checkCanDeleteSLOExecute(req)
}

// checkCanDeleteSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) checkCanDeleteSLOExecute(r apiCheckCanDeleteSLORequest) (CheckCanDeleteSLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CheckCanDeleteSLOResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.CheckCanDeleteSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/can_delete"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.ids == nil {
		return localVarReturnValue, nil, reportError("ids is required and must be specified")
	}
	localVarQueryParams.Add("ids", parameterToString(*r.ids, ""))

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
		if localVarHTTPResponse.StatusCode == 409 {
			var v CheckCanDeleteSLOResponse
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

type apiCreateSLORequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	body       *ServiceLevelObjectiveRequest
}

func (a *ServiceLevelObjectivesApiService) buildCreateSLORequest(ctx _context.Context, body ServiceLevelObjectiveRequest) (apiCreateSLORequest, error) {
	req := apiCreateSLORequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}
	return req, nil
}

// CreateSLO Create an SLO object.
// Create a service level objective object.
func (a *ServiceLevelObjectivesApiService) CreateSLO(ctx _context.Context, body ServiceLevelObjectiveRequest) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateSLORequest(ctx, body)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.createSLOExecute(req)
}

// createSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) createSLOExecute(r apiCreateSLORequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.CreateSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo"

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

type apiDeleteSLORequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	sloId      string
	force      *string
}

// DeleteSLOOptionalParameters holds optional parameters for DeleteSLO.
type DeleteSLOOptionalParameters struct {
	Force *string
}

// NewDeleteSLOOptionalParameters creates an empty struct for parameters.
func NewDeleteSLOOptionalParameters() *DeleteSLOOptionalParameters {
	this := DeleteSLOOptionalParameters{}
	return &this
}

// WithForce sets the corresponding parameter name and returns the struct.
func (r *DeleteSLOOptionalParameters) WithForce(force string) *DeleteSLOOptionalParameters {
	r.Force = &force
	return r
}

func (a *ServiceLevelObjectivesApiService) buildDeleteSLORequest(ctx _context.Context, sloId string, o ...DeleteSLOOptionalParameters) (apiDeleteSLORequest, error) {
	req := apiDeleteSLORequest{
		ApiService: a,
		ctx:        ctx,
		sloId:      sloId,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type DeleteSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.force = o[0].Force
	}
	return req, nil
}

// DeleteSLO Delete an SLO.
// Permanently delete the specified service level objective object.
//
// If an SLO is used in a dashboard, the `DELETE /v1/slo/` endpoint returns
// a 409 conflict error because the SLO is referenced in a dashboard.
func (a *ServiceLevelObjectivesApiService) DeleteSLO(ctx _context.Context, sloId string, o ...DeleteSLOOptionalParameters) (SLODeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteSLORequest(ctx, sloId, o...)
	if err != nil {
		var localVarReturnValue SLODeleteResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.deleteSLOExecute(req)
}

// deleteSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) deleteSLOExecute(r apiDeleteSLORequest) (SLODeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue SLODeleteResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.DeleteSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(parameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.force != nil {
		localVarQueryParams.Add("force", parameterToString(*r.force, ""))
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
			var v SLODeleteResponse
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

type apiDeleteSLOTimeframeInBulkRequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	body       *map[string][]SLOTimeframe
}

func (a *ServiceLevelObjectivesApiService) buildDeleteSLOTimeframeInBulkRequest(ctx _context.Context, body map[string][]SLOTimeframe) (apiDeleteSLOTimeframeInBulkRequest, error) {
	req := apiDeleteSLOTimeframeInBulkRequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}
	return req, nil
}

// DeleteSLOTimeframeInBulk Bulk Delete SLO Timeframes.
// Delete (or partially delete) multiple service level objective objects.
//
// This endpoint facilitates deletion of one or more thresholds for one or more
// service level objective objects. If all thresholds are deleted, the service level
// objective object is deleted as well.
func (a *ServiceLevelObjectivesApiService) DeleteSLOTimeframeInBulk(ctx _context.Context, body map[string][]SLOTimeframe) (SLOBulkDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteSLOTimeframeInBulkRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SLOBulkDeleteResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.deleteSLOTimeframeInBulkExecute(req)
}

// deleteSLOTimeframeInBulkExecute executes the request.
func (a *ServiceLevelObjectivesApiService) deleteSLOTimeframeInBulkExecute(r apiDeleteSLOTimeframeInBulkRequest) (SLOBulkDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SLOBulkDeleteResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.DeleteSLOTimeframeInBulk")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/bulk_delete"

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

type apiGetSLORequest struct {
	ctx                    _context.Context
	ApiService             *ServiceLevelObjectivesApiService
	sloId                  string
	withConfiguredAlertIds *bool
}

// GetSLOOptionalParameters holds optional parameters for GetSLO.
type GetSLOOptionalParameters struct {
	WithConfiguredAlertIds *bool
}

// NewGetSLOOptionalParameters creates an empty struct for parameters.
func NewGetSLOOptionalParameters() *GetSLOOptionalParameters {
	this := GetSLOOptionalParameters{}
	return &this
}

// WithWithConfiguredAlertIds sets the corresponding parameter name and returns the struct.
func (r *GetSLOOptionalParameters) WithWithConfiguredAlertIds(withConfiguredAlertIds bool) *GetSLOOptionalParameters {
	r.WithConfiguredAlertIds = &withConfiguredAlertIds
	return r
}

func (a *ServiceLevelObjectivesApiService) buildGetSLORequest(ctx _context.Context, sloId string, o ...GetSLOOptionalParameters) (apiGetSLORequest, error) {
	req := apiGetSLORequest{
		ApiService: a,
		ctx:        ctx,
		sloId:      sloId,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type GetSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.withConfiguredAlertIds = o[0].WithConfiguredAlertIds
	}
	return req, nil
}

// GetSLO Get an SLO's details.
// Get a service level objective object.
func (a *ServiceLevelObjectivesApiService) GetSLO(ctx _context.Context, sloId string, o ...GetSLOOptionalParameters) (SLOResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLORequest(ctx, sloId, o...)
	if err != nil {
		var localVarReturnValue SLOResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getSLOExecute(req)
}

// getSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) getSLOExecute(r apiGetSLORequest) (SLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.GetSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(parameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.withConfiguredAlertIds != nil {
		localVarQueryParams.Add("with_configured_alert_ids", parameterToString(*r.withConfiguredAlertIds, ""))
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

type apiGetSLOCorrectionsRequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	sloId      string
}

func (a *ServiceLevelObjectivesApiService) buildGetSLOCorrectionsRequest(ctx _context.Context, sloId string) (apiGetSLOCorrectionsRequest, error) {
	req := apiGetSLOCorrectionsRequest{
		ApiService: a,
		ctx:        ctx,
		sloId:      sloId,
	}
	return req, nil
}

// GetSLOCorrections Get Corrections For an SLO.
// Get corrections applied to an SLO
func (a *ServiceLevelObjectivesApiService) GetSLOCorrections(ctx _context.Context, sloId string) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLOCorrectionsRequest(ctx, sloId)
	if err != nil {
		var localVarReturnValue SLOCorrectionListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getSLOCorrectionsExecute(req)
}

// getSLOCorrectionsExecute executes the request.
func (a *ServiceLevelObjectivesApiService) getSLOCorrectionsExecute(r apiGetSLOCorrectionsRequest) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.GetSLOCorrections")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}/corrections"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(parameterToString(r.sloId, "")), -1)

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

type apiGetSLOHistoryRequest struct {
	ctx             _context.Context
	ApiService      *ServiceLevelObjectivesApiService
	sloId           string
	fromTs          *int64
	toTs            *int64
	target          *float64
	applyCorrection *bool
}

// GetSLOHistoryOptionalParameters holds optional parameters for GetSLOHistory.
type GetSLOHistoryOptionalParameters struct {
	Target          *float64
	ApplyCorrection *bool
}

// NewGetSLOHistoryOptionalParameters creates an empty struct for parameters.
func NewGetSLOHistoryOptionalParameters() *GetSLOHistoryOptionalParameters {
	this := GetSLOHistoryOptionalParameters{}
	return &this
}

// WithTarget sets the corresponding parameter name and returns the struct.
func (r *GetSLOHistoryOptionalParameters) WithTarget(target float64) *GetSLOHistoryOptionalParameters {
	r.Target = &target
	return r
}

// WithApplyCorrection sets the corresponding parameter name and returns the struct.
func (r *GetSLOHistoryOptionalParameters) WithApplyCorrection(applyCorrection bool) *GetSLOHistoryOptionalParameters {
	r.ApplyCorrection = &applyCorrection
	return r
}

func (a *ServiceLevelObjectivesApiService) buildGetSLOHistoryRequest(ctx _context.Context, sloId string, fromTs int64, toTs int64, o ...GetSLOHistoryOptionalParameters) (apiGetSLOHistoryRequest, error) {
	req := apiGetSLOHistoryRequest{
		ApiService: a,
		ctx:        ctx,
		sloId:      sloId,
		fromTs:     &fromTs,
		toTs:       &toTs,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type GetSLOHistoryOptionalParameters is allowed")
	}

	if o != nil {
		req.target = o[0].Target
		req.applyCorrection = o[0].ApplyCorrection
	}
	return req, nil
}

// GetSLOHistory Get an SLO's history.
// Get a specific SLO’s history, regardless of its SLO type.
//
// The detailed history data is structured according to the source data type.
// For example, metric data is included for event SLOs that use
// the metric source, and monitor SLO types include the monitor transition history.
//
// **Note:** There are different response formats for event based and time based SLOs.
// Examples of both are shown.
func (a *ServiceLevelObjectivesApiService) GetSLOHistory(ctx _context.Context, sloId string, fromTs int64, toTs int64, o ...GetSLOHistoryOptionalParameters) (SLOHistoryResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLOHistoryRequest(ctx, sloId, fromTs, toTs, o...)
	if err != nil {
		var localVarReturnValue SLOHistoryResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getSLOHistoryExecute(req)
}

// getSLOHistoryExecute executes the request.
func (a *ServiceLevelObjectivesApiService) getSLOHistoryExecute(r apiGetSLOHistoryRequest) (SLOHistoryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOHistoryResponse
	)

	operationId := "GetSLOHistory"
	if r.ApiService.client.cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, GenericOpenAPIError{error: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.GetSLOHistory")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}/history"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(parameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.fromTs == nil {
		return localVarReturnValue, nil, reportError("fromTs is required and must be specified")
	}
	if r.toTs == nil {
		return localVarReturnValue, nil, reportError("toTs is required and must be specified")
	}
	localVarQueryParams.Add("from_ts", parameterToString(*r.fromTs, ""))
	localVarQueryParams.Add("to_ts", parameterToString(*r.toTs, ""))
	if r.target != nil {
		localVarQueryParams.Add("target", parameterToString(*r.target, ""))
	}
	if r.applyCorrection != nil {
		localVarQueryParams.Add("apply_correction", parameterToString(*r.applyCorrection, ""))
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

type apiListSLOsRequest struct {
	ctx          _context.Context
	ApiService   *ServiceLevelObjectivesApiService
	ids          *string
	query        *string
	tagsQuery    *string
	metricsQuery *string
	limit        *int64
	offset       *int64
}

// ListSLOsOptionalParameters holds optional parameters for ListSLOs.
type ListSLOsOptionalParameters struct {
	Ids          *string
	Query        *string
	TagsQuery    *string
	MetricsQuery *string
	Limit        *int64
	Offset       *int64
}

// NewListSLOsOptionalParameters creates an empty struct for parameters.
func NewListSLOsOptionalParameters() *ListSLOsOptionalParameters {
	this := ListSLOsOptionalParameters{}
	return &this
}

// WithIds sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithIds(ids string) *ListSLOsOptionalParameters {
	r.Ids = &ids
	return r
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithQuery(query string) *ListSLOsOptionalParameters {
	r.Query = &query
	return r
}

// WithTagsQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithTagsQuery(tagsQuery string) *ListSLOsOptionalParameters {
	r.TagsQuery = &tagsQuery
	return r
}

// WithMetricsQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithMetricsQuery(metricsQuery string) *ListSLOsOptionalParameters {
	r.MetricsQuery = &metricsQuery
	return r
}

// WithLimit sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithLimit(limit int64) *ListSLOsOptionalParameters {
	r.Limit = &limit
	return r
}

// WithOffset sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithOffset(offset int64) *ListSLOsOptionalParameters {
	r.Offset = &offset
	return r
}

func (a *ServiceLevelObjectivesApiService) buildListSLOsRequest(ctx _context.Context, o ...ListSLOsOptionalParameters) (apiListSLOsRequest, error) {
	req := apiListSLOsRequest{
		ApiService: a,
		ctx:        ctx,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type ListSLOsOptionalParameters is allowed")
	}

	if o != nil {
		req.ids = o[0].Ids
		req.query = o[0].Query
		req.tagsQuery = o[0].TagsQuery
		req.metricsQuery = o[0].MetricsQuery
		req.limit = o[0].Limit
		req.offset = o[0].Offset
	}
	return req, nil
}

// ListSLOs Get all SLOs.
// Get a list of service level objective objects for your organization.
func (a *ServiceLevelObjectivesApiService) ListSLOs(ctx _context.Context, o ...ListSLOsOptionalParameters) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildListSLOsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.listSLOsExecute(req)
}

// listSLOsExecute executes the request.
func (a *ServiceLevelObjectivesApiService) listSLOsExecute(r apiListSLOsRequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.ListSLOs")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.ids != nil {
		localVarQueryParams.Add("ids", parameterToString(*r.ids, ""))
	}
	if r.query != nil {
		localVarQueryParams.Add("query", parameterToString(*r.query, ""))
	}
	if r.tagsQuery != nil {
		localVarQueryParams.Add("tags_query", parameterToString(*r.tagsQuery, ""))
	}
	if r.metricsQuery != nil {
		localVarQueryParams.Add("metrics_query", parameterToString(*r.metricsQuery, ""))
	}
	if r.limit != nil {
		localVarQueryParams.Add("limit", parameterToString(*r.limit, ""))
	}
	if r.offset != nil {
		localVarQueryParams.Add("offset", parameterToString(*r.offset, ""))
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

type apiSearchSLORequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	query      *string
	pageSize   *int64
	pageNumber *int64
}

// SearchSLOOptionalParameters holds optional parameters for SearchSLO.
type SearchSLOOptionalParameters struct {
	Query      *string
	PageSize   *int64
	PageNumber *int64
}

// NewSearchSLOOptionalParameters creates an empty struct for parameters.
func NewSearchSLOOptionalParameters() *SearchSLOOptionalParameters {
	this := SearchSLOOptionalParameters{}
	return &this
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithQuery(query string) *SearchSLOOptionalParameters {
	r.Query = &query
	return r
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithPageSize(pageSize int64) *SearchSLOOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithPageNumber(pageNumber int64) *SearchSLOOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

func (a *ServiceLevelObjectivesApiService) buildSearchSLORequest(ctx _context.Context, o ...SearchSLOOptionalParameters) (apiSearchSLORequest, error) {
	req := apiSearchSLORequest{
		ApiService: a,
		ctx:        ctx,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type SearchSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.query = o[0].Query
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
	}
	return req, nil
}

// SearchSLO Search for SLOs.
// Get a list of service level objective objects for your organization.
func (a *ServiceLevelObjectivesApiService) SearchSLO(ctx _context.Context, o ...SearchSLOOptionalParameters) (SearchSLOResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchSLORequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SearchSLOResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.searchSLOExecute(req)
}

// searchSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) searchSLOExecute(r apiSearchSLORequest) (SearchSLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SearchSLOResponse
	)

	operationId := "SearchSLO"
	if r.ApiService.client.cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, GenericOpenAPIError{error: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.SearchSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/search"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.query != nil {
		localVarQueryParams.Add("query", parameterToString(*r.query, ""))
	}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", parameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page[number]", parameterToString(*r.pageNumber, ""))
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

type apiUpdateSLORequest struct {
	ctx        _context.Context
	ApiService *ServiceLevelObjectivesApiService
	sloId      string
	body       *ServiceLevelObjective
}

func (a *ServiceLevelObjectivesApiService) buildUpdateSLORequest(ctx _context.Context, sloId string, body ServiceLevelObjective) (apiUpdateSLORequest, error) {
	req := apiUpdateSLORequest{
		ApiService: a,
		ctx:        ctx,
		sloId:      sloId,
		body:       &body,
	}
	return req, nil
}

// UpdateSLO Update an SLO.
// Update the specified service level objective object.
func (a *ServiceLevelObjectivesApiService) UpdateSLO(ctx _context.Context, sloId string, body ServiceLevelObjective) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateSLORequest(ctx, sloId, body)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.updateSLOExecute(req)
}

// updateSLOExecute executes the request.
func (a *ServiceLevelObjectivesApiService) updateSLOExecute(r apiUpdateSLORequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "ServiceLevelObjectivesApiService.UpdateSLO")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(parameterToString(r.sloId, "")), -1)

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

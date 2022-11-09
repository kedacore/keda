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

// SnapshotsApiService SnapshotsApi service.
type SnapshotsApiService service

type apiGetGraphSnapshotRequest struct {
	ctx         _context.Context
	ApiService  *SnapshotsApiService
	start       *int64
	end         *int64
	metricQuery *string
	eventQuery  *string
	graphDef    *string
	title       *string
	height      *int64
	width       *int64
}

// GetGraphSnapshotOptionalParameters holds optional parameters for GetGraphSnapshot.
type GetGraphSnapshotOptionalParameters struct {
	MetricQuery *string
	EventQuery  *string
	GraphDef    *string
	Title       *string
	Height      *int64
	Width       *int64
}

// NewGetGraphSnapshotOptionalParameters creates an empty struct for parameters.
func NewGetGraphSnapshotOptionalParameters() *GetGraphSnapshotOptionalParameters {
	this := GetGraphSnapshotOptionalParameters{}
	return &this
}

// WithMetricQuery sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithMetricQuery(metricQuery string) *GetGraphSnapshotOptionalParameters {
	r.MetricQuery = &metricQuery
	return r
}

// WithEventQuery sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithEventQuery(eventQuery string) *GetGraphSnapshotOptionalParameters {
	r.EventQuery = &eventQuery
	return r
}

// WithGraphDef sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithGraphDef(graphDef string) *GetGraphSnapshotOptionalParameters {
	r.GraphDef = &graphDef
	return r
}

// WithTitle sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithTitle(title string) *GetGraphSnapshotOptionalParameters {
	r.Title = &title
	return r
}

// WithHeight sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithHeight(height int64) *GetGraphSnapshotOptionalParameters {
	r.Height = &height
	return r
}

// WithWidth sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithWidth(width int64) *GetGraphSnapshotOptionalParameters {
	r.Width = &width
	return r
}

func (a *SnapshotsApiService) buildGetGraphSnapshotRequest(ctx _context.Context, start int64, end int64, o ...GetGraphSnapshotOptionalParameters) (apiGetGraphSnapshotRequest, error) {
	req := apiGetGraphSnapshotRequest{
		ApiService: a,
		ctx:        ctx,
		start:      &start,
		end:        &end,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type GetGraphSnapshotOptionalParameters is allowed")
	}

	if o != nil {
		req.metricQuery = o[0].MetricQuery
		req.eventQuery = o[0].EventQuery
		req.graphDef = o[0].GraphDef
		req.title = o[0].Title
		req.height = o[0].Height
		req.width = o[0].Width
	}
	return req, nil
}

// GetGraphSnapshot Take graph snapshots.
// Take graph snapshots.
// **Note**: When a snapshot is created, there is some delay before it is available.
func (a *SnapshotsApiService) GetGraphSnapshot(ctx _context.Context, start int64, end int64, o ...GetGraphSnapshotOptionalParameters) (GraphSnapshot, *_nethttp.Response, error) {
	req, err := a.buildGetGraphSnapshotRequest(ctx, start, end, o...)
	if err != nil {
		var localVarReturnValue GraphSnapshot
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getGraphSnapshotExecute(req)
}

// getGraphSnapshotExecute executes the request.
func (a *SnapshotsApiService) getGraphSnapshotExecute(r apiGetGraphSnapshotRequest) (GraphSnapshot, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue GraphSnapshot
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "SnapshotsApiService.GetGraphSnapshot")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/graph/snapshot"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.start == nil {
		return localVarReturnValue, nil, reportError("start is required and must be specified")
	}
	if r.end == nil {
		return localVarReturnValue, nil, reportError("end is required and must be specified")
	}
	localVarQueryParams.Add("start", parameterToString(*r.start, ""))
	localVarQueryParams.Add("end", parameterToString(*r.end, ""))
	if r.metricQuery != nil {
		localVarQueryParams.Add("metric_query", parameterToString(*r.metricQuery, ""))
	}
	if r.eventQuery != nil {
		localVarQueryParams.Add("event_query", parameterToString(*r.eventQuery, ""))
	}
	if r.graphDef != nil {
		localVarQueryParams.Add("graph_def", parameterToString(*r.graphDef, ""))
	}
	if r.title != nil {
		localVarQueryParams.Add("title", parameterToString(*r.title, ""))
	}
	if r.height != nil {
		localVarQueryParams.Add("height", parameterToString(*r.height, ""))
	}
	if r.width != nil {
		localVarQueryParams.Add("width", parameterToString(*r.width, ""))
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

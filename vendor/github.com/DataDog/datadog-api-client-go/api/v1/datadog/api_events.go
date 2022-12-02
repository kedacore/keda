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

// EventsApiService EventsApi service.
type EventsApiService service

type apiCreateEventRequest struct {
	ctx        _context.Context
	ApiService *EventsApiService
	body       *EventCreateRequest
}

func (a *EventsApiService) buildCreateEventRequest(ctx _context.Context, body EventCreateRequest) (apiCreateEventRequest, error) {
	req := apiCreateEventRequest{
		ApiService: a,
		ctx:        ctx,
		body:       &body,
	}
	return req, nil
}

// CreateEvent Post an event.
// This endpoint allows you to post events to the stream.
// Tag them, set priority and event aggregate them with other events.
func (a *EventsApiService) CreateEvent(ctx _context.Context, body EventCreateRequest) (EventCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateEventRequest(ctx, body)
	if err != nil {
		var localVarReturnValue EventCreateResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.createEventExecute(req)
}

// createEventExecute executes the request.
func (a *EventsApiService) createEventExecute(r apiCreateEventRequest) (EventCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue EventCreateResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "EventsApiService.CreateEvent")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

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

type apiGetEventRequest struct {
	ctx        _context.Context
	ApiService *EventsApiService
	eventId    int64
}

func (a *EventsApiService) buildGetEventRequest(ctx _context.Context, eventId int64) (apiGetEventRequest, error) {
	req := apiGetEventRequest{
		ApiService: a,
		ctx:        ctx,
		eventId:    eventId,
	}
	return req, nil
}

// GetEvent Get an event.
// This endpoint allows you to query for event details.
//
// **Note**: If the event you’re querying contains markdown formatting of any kind,
// you may see characters such as `%`,`\`,`n` in your output.
func (a *EventsApiService) GetEvent(ctx _context.Context, eventId int64) (EventResponse, *_nethttp.Response, error) {
	req, err := a.buildGetEventRequest(ctx, eventId)
	if err != nil {
		var localVarReturnValue EventResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.getEventExecute(req)
}

// getEventExecute executes the request.
func (a *EventsApiService) getEventExecute(r apiGetEventRequest) (EventResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "EventsApiService.GetEvent")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events/{event_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"event_id"+"}", _neturl.PathEscape(parameterToString(r.eventId, "")), -1)

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

type apiListEventsRequest struct {
	ctx              _context.Context
	ApiService       *EventsApiService
	start            *int64
	end              *int64
	priority         *EventPriority
	sources          *string
	tags             *string
	unaggregated     *bool
	excludeAggregate *bool
	page             *int32
}

// ListEventsOptionalParameters holds optional parameters for ListEvents.
type ListEventsOptionalParameters struct {
	Priority         *EventPriority
	Sources          *string
	Tags             *string
	Unaggregated     *bool
	ExcludeAggregate *bool
	Page             *int32
}

// NewListEventsOptionalParameters creates an empty struct for parameters.
func NewListEventsOptionalParameters() *ListEventsOptionalParameters {
	this := ListEventsOptionalParameters{}
	return &this
}

// WithPriority sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPriority(priority EventPriority) *ListEventsOptionalParameters {
	r.Priority = &priority
	return r
}

// WithSources sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithSources(sources string) *ListEventsOptionalParameters {
	r.Sources = &sources
	return r
}

// WithTags sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithTags(tags string) *ListEventsOptionalParameters {
	r.Tags = &tags
	return r
}

// WithUnaggregated sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithUnaggregated(unaggregated bool) *ListEventsOptionalParameters {
	r.Unaggregated = &unaggregated
	return r
}

// WithExcludeAggregate sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithExcludeAggregate(excludeAggregate bool) *ListEventsOptionalParameters {
	r.ExcludeAggregate = &excludeAggregate
	return r
}

// WithPage sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPage(page int32) *ListEventsOptionalParameters {
	r.Page = &page
	return r
}

func (a *EventsApiService) buildListEventsRequest(ctx _context.Context, start int64, end int64, o ...ListEventsOptionalParameters) (apiListEventsRequest, error) {
	req := apiListEventsRequest{
		ApiService: a,
		ctx:        ctx,
		start:      &start,
		end:        &end,
	}

	if len(o) > 1 {
		return req, reportError("only one argument of type ListEventsOptionalParameters is allowed")
	}

	if o != nil {
		req.priority = o[0].Priority
		req.sources = o[0].Sources
		req.tags = o[0].Tags
		req.unaggregated = o[0].Unaggregated
		req.excludeAggregate = o[0].ExcludeAggregate
		req.page = o[0].Page
	}
	return req, nil
}

// ListEvents Query the event stream.
// The event stream can be queried and filtered by time, priority, sources and tags.
//
// **Notes**:
// - If the event you’re querying contains markdown formatting of any kind,
// you may see characters such as `%`,`\`,`n` in your output.
//
// - This endpoint returns a maximum of `1000` most recent results. To return additional results,
// identify the last timestamp of the last result and set that as the `end` query time to
// paginate the results. You can also use the page parameter to specify which set of `1000` results to return.
func (a *EventsApiService) ListEvents(ctx _context.Context, start int64, end int64, o ...ListEventsOptionalParameters) (EventListResponse, *_nethttp.Response, error) {
	req, err := a.buildListEventsRequest(ctx, start, end, o...)
	if err != nil {
		var localVarReturnValue EventListResponse
		return localVarReturnValue, nil, err
	}

	return req.ApiService.listEventsExecute(req)
}

// listEventsExecute executes the request.
func (a *EventsApiService) listEventsExecute(r apiListEventsRequest) (EventListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventListResponse
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "EventsApiService.ListEvents")
	if err != nil {
		return localVarReturnValue, nil, GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

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
	if r.priority != nil {
		localVarQueryParams.Add("priority", parameterToString(*r.priority, ""))
	}
	if r.sources != nil {
		localVarQueryParams.Add("sources", parameterToString(*r.sources, ""))
	}
	if r.tags != nil {
		localVarQueryParams.Add("tags", parameterToString(*r.tags, ""))
	}
	if r.unaggregated != nil {
		localVarQueryParams.Add("unaggregated", parameterToString(*r.unaggregated, ""))
	}
	if r.excludeAggregate != nil {
		localVarQueryParams.Add("exclude_aggregate", parameterToString(*r.excludeAggregate, ""))
	}
	if r.page != nil {
		localVarQueryParams.Add("page", parameterToString(*r.page, ""))
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

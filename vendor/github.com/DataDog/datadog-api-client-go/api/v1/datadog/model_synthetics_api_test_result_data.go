// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsAPITestResultData Object containing results for your Synthetic API test.
type SyntheticsAPITestResultData struct {
	// Object describing the SSL certificate used for a Synthetic test.
	Cert *SyntheticsSSLCertificate `json:"cert,omitempty"`
	// Status of a Synthetic test.
	EventType *SyntheticsTestProcessStatus `json:"eventType,omitempty"`
	// The API test failure details.
	Failure *SyntheticsApiTestResultFailure `json:"failure,omitempty"`
	// The API test HTTP status code.
	HttpStatusCode *int64 `json:"httpStatusCode,omitempty"`
	// Request header object used for the API test.
	RequestHeaders map[string]interface{} `json:"requestHeaders,omitempty"`
	// Response body returned for the API test.
	ResponseBody *string `json:"responseBody,omitempty"`
	// Response headers returned for the API test.
	ResponseHeaders map[string]interface{} `json:"responseHeaders,omitempty"`
	// Global size in byte of the API test response.
	ResponseSize *int64 `json:"responseSize,omitempty"`
	// Object containing all metrics and their values collected for a Synthetic API test.
	// Learn more about those metrics in [Synthetics documentation](https://docs.datadoghq.com/synthetics/#metrics).
	Timings *SyntheticsTiming `json:"timings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAPITestResultData instantiates a new SyntheticsAPITestResultData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAPITestResultData() *SyntheticsAPITestResultData {
	this := SyntheticsAPITestResultData{}
	return &this
}

// NewSyntheticsAPITestResultDataWithDefaults instantiates a new SyntheticsAPITestResultData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAPITestResultDataWithDefaults() *SyntheticsAPITestResultData {
	this := SyntheticsAPITestResultData{}
	return &this
}

// GetCert returns the Cert field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetCert() SyntheticsSSLCertificate {
	if o == nil || o.Cert == nil {
		var ret SyntheticsSSLCertificate
		return ret
	}
	return *o.Cert
}

// GetCertOk returns a tuple with the Cert field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetCertOk() (*SyntheticsSSLCertificate, bool) {
	if o == nil || o.Cert == nil {
		return nil, false
	}
	return o.Cert, true
}

// HasCert returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasCert() bool {
	if o != nil && o.Cert != nil {
		return true
	}

	return false
}

// SetCert gets a reference to the given SyntheticsSSLCertificate and assigns it to the Cert field.
func (o *SyntheticsAPITestResultData) SetCert(v SyntheticsSSLCertificate) {
	o.Cert = &v
}

// GetEventType returns the EventType field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetEventType() SyntheticsTestProcessStatus {
	if o == nil || o.EventType == nil {
		var ret SyntheticsTestProcessStatus
		return ret
	}
	return *o.EventType
}

// GetEventTypeOk returns a tuple with the EventType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetEventTypeOk() (*SyntheticsTestProcessStatus, bool) {
	if o == nil || o.EventType == nil {
		return nil, false
	}
	return o.EventType, true
}

// HasEventType returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasEventType() bool {
	if o != nil && o.EventType != nil {
		return true
	}

	return false
}

// SetEventType gets a reference to the given SyntheticsTestProcessStatus and assigns it to the EventType field.
func (o *SyntheticsAPITestResultData) SetEventType(v SyntheticsTestProcessStatus) {
	o.EventType = &v
}

// GetFailure returns the Failure field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetFailure() SyntheticsApiTestResultFailure {
	if o == nil || o.Failure == nil {
		var ret SyntheticsApiTestResultFailure
		return ret
	}
	return *o.Failure
}

// GetFailureOk returns a tuple with the Failure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetFailureOk() (*SyntheticsApiTestResultFailure, bool) {
	if o == nil || o.Failure == nil {
		return nil, false
	}
	return o.Failure, true
}

// HasFailure returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasFailure() bool {
	if o != nil && o.Failure != nil {
		return true
	}

	return false
}

// SetFailure gets a reference to the given SyntheticsApiTestResultFailure and assigns it to the Failure field.
func (o *SyntheticsAPITestResultData) SetFailure(v SyntheticsApiTestResultFailure) {
	o.Failure = &v
}

// GetHttpStatusCode returns the HttpStatusCode field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetHttpStatusCode() int64 {
	if o == nil || o.HttpStatusCode == nil {
		var ret int64
		return ret
	}
	return *o.HttpStatusCode
}

// GetHttpStatusCodeOk returns a tuple with the HttpStatusCode field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetHttpStatusCodeOk() (*int64, bool) {
	if o == nil || o.HttpStatusCode == nil {
		return nil, false
	}
	return o.HttpStatusCode, true
}

// HasHttpStatusCode returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasHttpStatusCode() bool {
	if o != nil && o.HttpStatusCode != nil {
		return true
	}

	return false
}

// SetHttpStatusCode gets a reference to the given int64 and assigns it to the HttpStatusCode field.
func (o *SyntheticsAPITestResultData) SetHttpStatusCode(v int64) {
	o.HttpStatusCode = &v
}

// GetRequestHeaders returns the RequestHeaders field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetRequestHeaders() map[string]interface{} {
	if o == nil || o.RequestHeaders == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.RequestHeaders
}

// GetRequestHeadersOk returns a tuple with the RequestHeaders field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetRequestHeadersOk() (*map[string]interface{}, bool) {
	if o == nil || o.RequestHeaders == nil {
		return nil, false
	}
	return &o.RequestHeaders, true
}

// HasRequestHeaders returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasRequestHeaders() bool {
	if o != nil && o.RequestHeaders != nil {
		return true
	}

	return false
}

// SetRequestHeaders gets a reference to the given map[string]interface{} and assigns it to the RequestHeaders field.
func (o *SyntheticsAPITestResultData) SetRequestHeaders(v map[string]interface{}) {
	o.RequestHeaders = v
}

// GetResponseBody returns the ResponseBody field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetResponseBody() string {
	if o == nil || o.ResponseBody == nil {
		var ret string
		return ret
	}
	return *o.ResponseBody
}

// GetResponseBodyOk returns a tuple with the ResponseBody field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetResponseBodyOk() (*string, bool) {
	if o == nil || o.ResponseBody == nil {
		return nil, false
	}
	return o.ResponseBody, true
}

// HasResponseBody returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasResponseBody() bool {
	if o != nil && o.ResponseBody != nil {
		return true
	}

	return false
}

// SetResponseBody gets a reference to the given string and assigns it to the ResponseBody field.
func (o *SyntheticsAPITestResultData) SetResponseBody(v string) {
	o.ResponseBody = &v
}

// GetResponseHeaders returns the ResponseHeaders field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetResponseHeaders() map[string]interface{} {
	if o == nil || o.ResponseHeaders == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.ResponseHeaders
}

// GetResponseHeadersOk returns a tuple with the ResponseHeaders field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetResponseHeadersOk() (*map[string]interface{}, bool) {
	if o == nil || o.ResponseHeaders == nil {
		return nil, false
	}
	return &o.ResponseHeaders, true
}

// HasResponseHeaders returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasResponseHeaders() bool {
	if o != nil && o.ResponseHeaders != nil {
		return true
	}

	return false
}

// SetResponseHeaders gets a reference to the given map[string]interface{} and assigns it to the ResponseHeaders field.
func (o *SyntheticsAPITestResultData) SetResponseHeaders(v map[string]interface{}) {
	o.ResponseHeaders = v
}

// GetResponseSize returns the ResponseSize field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetResponseSize() int64 {
	if o == nil || o.ResponseSize == nil {
		var ret int64
		return ret
	}
	return *o.ResponseSize
}

// GetResponseSizeOk returns a tuple with the ResponseSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetResponseSizeOk() (*int64, bool) {
	if o == nil || o.ResponseSize == nil {
		return nil, false
	}
	return o.ResponseSize, true
}

// HasResponseSize returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasResponseSize() bool {
	if o != nil && o.ResponseSize != nil {
		return true
	}

	return false
}

// SetResponseSize gets a reference to the given int64 and assigns it to the ResponseSize field.
func (o *SyntheticsAPITestResultData) SetResponseSize(v int64) {
	o.ResponseSize = &v
}

// GetTimings returns the Timings field value if set, zero value otherwise.
func (o *SyntheticsAPITestResultData) GetTimings() SyntheticsTiming {
	if o == nil || o.Timings == nil {
		var ret SyntheticsTiming
		return ret
	}
	return *o.Timings
}

// GetTimingsOk returns a tuple with the Timings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITestResultData) GetTimingsOk() (*SyntheticsTiming, bool) {
	if o == nil || o.Timings == nil {
		return nil, false
	}
	return o.Timings, true
}

// HasTimings returns a boolean if a field has been set.
func (o *SyntheticsAPITestResultData) HasTimings() bool {
	if o != nil && o.Timings != nil {
		return true
	}

	return false
}

// SetTimings gets a reference to the given SyntheticsTiming and assigns it to the Timings field.
func (o *SyntheticsAPITestResultData) SetTimings(v SyntheticsTiming) {
	o.Timings = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAPITestResultData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cert != nil {
		toSerialize["cert"] = o.Cert
	}
	if o.EventType != nil {
		toSerialize["eventType"] = o.EventType
	}
	if o.Failure != nil {
		toSerialize["failure"] = o.Failure
	}
	if o.HttpStatusCode != nil {
		toSerialize["httpStatusCode"] = o.HttpStatusCode
	}
	if o.RequestHeaders != nil {
		toSerialize["requestHeaders"] = o.RequestHeaders
	}
	if o.ResponseBody != nil {
		toSerialize["responseBody"] = o.ResponseBody
	}
	if o.ResponseHeaders != nil {
		toSerialize["responseHeaders"] = o.ResponseHeaders
	}
	if o.ResponseSize != nil {
		toSerialize["responseSize"] = o.ResponseSize
	}
	if o.Timings != nil {
		toSerialize["timings"] = o.Timings
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAPITestResultData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cert            *SyntheticsSSLCertificate       `json:"cert,omitempty"`
		EventType       *SyntheticsTestProcessStatus    `json:"eventType,omitempty"`
		Failure         *SyntheticsApiTestResultFailure `json:"failure,omitempty"`
		HttpStatusCode  *int64                          `json:"httpStatusCode,omitempty"`
		RequestHeaders  map[string]interface{}          `json:"requestHeaders,omitempty"`
		ResponseBody    *string                         `json:"responseBody,omitempty"`
		ResponseHeaders map[string]interface{}          `json:"responseHeaders,omitempty"`
		ResponseSize    *int64                          `json:"responseSize,omitempty"`
		Timings         *SyntheticsTiming               `json:"timings,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.EventType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Cert != nil && all.Cert.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Cert = all.Cert
	o.EventType = all.EventType
	if all.Failure != nil && all.Failure.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Failure = all.Failure
	o.HttpStatusCode = all.HttpStatusCode
	o.RequestHeaders = all.RequestHeaders
	o.ResponseBody = all.ResponseBody
	o.ResponseHeaders = all.ResponseHeaders
	o.ResponseSize = all.ResponseSize
	if all.Timings != nil && all.Timings.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Timings = all.Timings
	return nil
}

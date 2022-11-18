// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// HTTPLogItem Logs that are sent over HTTP.
type HTTPLogItem struct {
	// The integration name associated with your log: the technology from which the log originated.
	// When it matches an integration name, Datadog automatically installs the corresponding parsers and facets.
	// See [reserved attributes](https://docs.datadoghq.com/logs/log_collection/#reserved-attributes).
	Ddsource *string `json:"ddsource,omitempty"`
	// Tags associated with your logs.
	Ddtags *string `json:"ddtags,omitempty"`
	// The name of the originating host of the log.
	Hostname *string `json:"hostname,omitempty"`
	// The message [reserved attribute](https://docs.datadoghq.com/logs/log_collection/#reserved-attributes)
	// of your log. By default, Datadog ingests the value of the message attribute as the body of the log entry.
	// That value is then highlighted and displayed in the Logstream, where it is indexed for full text search.
	Message string `json:"message"`
	// The name of the application or service generating the log events.
	// It is used to switch from Logs to APM, so make sure you define the same value when you use both products.
	// See [reserved attributes](https://docs.datadoghq.com/logs/log_collection/#reserved-attributes).
	Service *string `json:"service,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]string
}

// NewHTTPLogItem instantiates a new HTTPLogItem object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHTTPLogItem(message string) *HTTPLogItem {
	this := HTTPLogItem{}
	this.Message = message
	return &this
}

// NewHTTPLogItemWithDefaults instantiates a new HTTPLogItem object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHTTPLogItemWithDefaults() *HTTPLogItem {
	this := HTTPLogItem{}
	return &this
}

// GetDdsource returns the Ddsource field value if set, zero value otherwise.
func (o *HTTPLogItem) GetDdsource() string {
	if o == nil || o.Ddsource == nil {
		var ret string
		return ret
	}
	return *o.Ddsource
}

// GetDdsourceOk returns a tuple with the Ddsource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HTTPLogItem) GetDdsourceOk() (*string, bool) {
	if o == nil || o.Ddsource == nil {
		return nil, false
	}
	return o.Ddsource, true
}

// HasDdsource returns a boolean if a field has been set.
func (o *HTTPLogItem) HasDdsource() bool {
	if o != nil && o.Ddsource != nil {
		return true
	}

	return false
}

// SetDdsource gets a reference to the given string and assigns it to the Ddsource field.
func (o *HTTPLogItem) SetDdsource(v string) {
	o.Ddsource = &v
}

// GetDdtags returns the Ddtags field value if set, zero value otherwise.
func (o *HTTPLogItem) GetDdtags() string {
	if o == nil || o.Ddtags == nil {
		var ret string
		return ret
	}
	return *o.Ddtags
}

// GetDdtagsOk returns a tuple with the Ddtags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HTTPLogItem) GetDdtagsOk() (*string, bool) {
	if o == nil || o.Ddtags == nil {
		return nil, false
	}
	return o.Ddtags, true
}

// HasDdtags returns a boolean if a field has been set.
func (o *HTTPLogItem) HasDdtags() bool {
	if o != nil && o.Ddtags != nil {
		return true
	}

	return false
}

// SetDdtags gets a reference to the given string and assigns it to the Ddtags field.
func (o *HTTPLogItem) SetDdtags(v string) {
	o.Ddtags = &v
}

// GetHostname returns the Hostname field value if set, zero value otherwise.
func (o *HTTPLogItem) GetHostname() string {
	if o == nil || o.Hostname == nil {
		var ret string
		return ret
	}
	return *o.Hostname
}

// GetHostnameOk returns a tuple with the Hostname field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HTTPLogItem) GetHostnameOk() (*string, bool) {
	if o == nil || o.Hostname == nil {
		return nil, false
	}
	return o.Hostname, true
}

// HasHostname returns a boolean if a field has been set.
func (o *HTTPLogItem) HasHostname() bool {
	if o != nil && o.Hostname != nil {
		return true
	}

	return false
}

// SetHostname gets a reference to the given string and assigns it to the Hostname field.
func (o *HTTPLogItem) SetHostname(v string) {
	o.Hostname = &v
}

// GetMessage returns the Message field value.
func (o *HTTPLogItem) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *HTTPLogItem) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *HTTPLogItem) SetMessage(v string) {
	o.Message = v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *HTTPLogItem) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HTTPLogItem) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *HTTPLogItem) HasService() bool {
	if o != nil && o.Service != nil {
		return true
	}

	return false
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *HTTPLogItem) SetService(v string) {
	o.Service = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HTTPLogItem) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Ddsource != nil {
		toSerialize["ddsource"] = o.Ddsource
	}
	if o.Ddtags != nil {
		toSerialize["ddtags"] = o.Ddtags
	}
	if o.Hostname != nil {
		toSerialize["hostname"] = o.Hostname
	}
	toSerialize["message"] = o.Message
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HTTPLogItem) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Message *string `json:"message"`
	}{}
	all := struct {
		Ddsource *string `json:"ddsource,omitempty"`
		Ddtags   *string `json:"ddtags,omitempty"`
		Hostname *string `json:"hostname,omitempty"`
		Message  string  `json:"message"`
		Service  *string `json:"service,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Message == nil {
		return fmt.Errorf("Required field message missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Ddsource = all.Ddsource
	o.Ddtags = all.Ddtags
	o.Hostname = all.Hostname
	o.Message = all.Message
	o.Service = all.Service
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestRequestProxy The proxy to perform the test.
type SyntheticsTestRequestProxy struct {
	// Headers to include when performing the test.
	Headers map[string]string `json:"headers,omitempty"`
	// URL of the proxy to perform the test.
	Url string `json:"url"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestRequestProxy instantiates a new SyntheticsTestRequestProxy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestRequestProxy(url string) *SyntheticsTestRequestProxy {
	this := SyntheticsTestRequestProxy{}
	this.Url = url
	return &this
}

// NewSyntheticsTestRequestProxyWithDefaults instantiates a new SyntheticsTestRequestProxy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestRequestProxyWithDefaults() *SyntheticsTestRequestProxy {
	this := SyntheticsTestRequestProxy{}
	return &this
}

// GetHeaders returns the Headers field value if set, zero value otherwise.
func (o *SyntheticsTestRequestProxy) GetHeaders() map[string]string {
	if o == nil || o.Headers == nil {
		var ret map[string]string
		return ret
	}
	return o.Headers
}

// GetHeadersOk returns a tuple with the Headers field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequestProxy) GetHeadersOk() (*map[string]string, bool) {
	if o == nil || o.Headers == nil {
		return nil, false
	}
	return &o.Headers, true
}

// HasHeaders returns a boolean if a field has been set.
func (o *SyntheticsTestRequestProxy) HasHeaders() bool {
	if o != nil && o.Headers != nil {
		return true
	}

	return false
}

// SetHeaders gets a reference to the given map[string]string and assigns it to the Headers field.
func (o *SyntheticsTestRequestProxy) SetHeaders(v map[string]string) {
	o.Headers = v
}

// GetUrl returns the Url field value.
func (o *SyntheticsTestRequestProxy) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequestProxy) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *SyntheticsTestRequestProxy) SetUrl(v string) {
	o.Url = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestRequestProxy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Headers != nil {
		toSerialize["headers"] = o.Headers
	}
	toSerialize["url"] = o.Url

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestRequestProxy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Url *string `json:"url"`
	}{}
	all := struct {
		Headers map[string]string `json:"headers,omitempty"`
		Url     string            `json:"url"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Url == nil {
		return fmt.Errorf("Required field url missing")
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
	o.Headers = all.Headers
	o.Url = all.Url
	return nil
}

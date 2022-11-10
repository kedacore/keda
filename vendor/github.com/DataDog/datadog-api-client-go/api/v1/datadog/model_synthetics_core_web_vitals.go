// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsCoreWebVitals Core Web Vitals attached to a browser test step.
type SyntheticsCoreWebVitals struct {
	// Cumulative Layout Shift.
	Cls *float64 `json:"cls,omitempty"`
	// Largest Contentful Paint in milliseconds.
	Lcp *float64 `json:"lcp,omitempty"`
	// URL attached to the metrics.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsCoreWebVitals instantiates a new SyntheticsCoreWebVitals object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsCoreWebVitals() *SyntheticsCoreWebVitals {
	this := SyntheticsCoreWebVitals{}
	return &this
}

// NewSyntheticsCoreWebVitalsWithDefaults instantiates a new SyntheticsCoreWebVitals object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsCoreWebVitalsWithDefaults() *SyntheticsCoreWebVitals {
	this := SyntheticsCoreWebVitals{}
	return &this
}

// GetCls returns the Cls field value if set, zero value otherwise.
func (o *SyntheticsCoreWebVitals) GetCls() float64 {
	if o == nil || o.Cls == nil {
		var ret float64
		return ret
	}
	return *o.Cls
}

// GetClsOk returns a tuple with the Cls field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCoreWebVitals) GetClsOk() (*float64, bool) {
	if o == nil || o.Cls == nil {
		return nil, false
	}
	return o.Cls, true
}

// HasCls returns a boolean if a field has been set.
func (o *SyntheticsCoreWebVitals) HasCls() bool {
	if o != nil && o.Cls != nil {
		return true
	}

	return false
}

// SetCls gets a reference to the given float64 and assigns it to the Cls field.
func (o *SyntheticsCoreWebVitals) SetCls(v float64) {
	o.Cls = &v
}

// GetLcp returns the Lcp field value if set, zero value otherwise.
func (o *SyntheticsCoreWebVitals) GetLcp() float64 {
	if o == nil || o.Lcp == nil {
		var ret float64
		return ret
	}
	return *o.Lcp
}

// GetLcpOk returns a tuple with the Lcp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCoreWebVitals) GetLcpOk() (*float64, bool) {
	if o == nil || o.Lcp == nil {
		return nil, false
	}
	return o.Lcp, true
}

// HasLcp returns a boolean if a field has been set.
func (o *SyntheticsCoreWebVitals) HasLcp() bool {
	if o != nil && o.Lcp != nil {
		return true
	}

	return false
}

// SetLcp gets a reference to the given float64 and assigns it to the Lcp field.
func (o *SyntheticsCoreWebVitals) SetLcp(v float64) {
	o.Lcp = &v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *SyntheticsCoreWebVitals) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsCoreWebVitals) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *SyntheticsCoreWebVitals) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *SyntheticsCoreWebVitals) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsCoreWebVitals) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cls != nil {
		toSerialize["cls"] = o.Cls
	}
	if o.Lcp != nil {
		toSerialize["lcp"] = o.Lcp
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsCoreWebVitals) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cls *float64 `json:"cls,omitempty"`
		Lcp *float64 `json:"lcp,omitempty"`
		Url *string  `json:"url,omitempty"`
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
	o.Cls = all.Cls
	o.Lcp = all.Lcp
	o.Url = all.Url
	return nil
}

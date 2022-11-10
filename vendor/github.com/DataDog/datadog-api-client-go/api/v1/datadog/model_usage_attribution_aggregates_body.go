// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageAttributionAggregatesBody The object containing the aggregates.
type UsageAttributionAggregatesBody struct {
	// The aggregate type.
	AggType *string `json:"agg_type,omitempty"`
	// The field.
	Field *string `json:"field,omitempty"`
	// The value for a given field.
	Value *float64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageAttributionAggregatesBody instantiates a new UsageAttributionAggregatesBody object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageAttributionAggregatesBody() *UsageAttributionAggregatesBody {
	this := UsageAttributionAggregatesBody{}
	return &this
}

// NewUsageAttributionAggregatesBodyWithDefaults instantiates a new UsageAttributionAggregatesBody object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageAttributionAggregatesBodyWithDefaults() *UsageAttributionAggregatesBody {
	this := UsageAttributionAggregatesBody{}
	return &this
}

// GetAggType returns the AggType field value if set, zero value otherwise.
func (o *UsageAttributionAggregatesBody) GetAggType() string {
	if o == nil || o.AggType == nil {
		var ret string
		return ret
	}
	return *o.AggType
}

// GetAggTypeOk returns a tuple with the AggType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionAggregatesBody) GetAggTypeOk() (*string, bool) {
	if o == nil || o.AggType == nil {
		return nil, false
	}
	return o.AggType, true
}

// HasAggType returns a boolean if a field has been set.
func (o *UsageAttributionAggregatesBody) HasAggType() bool {
	if o != nil && o.AggType != nil {
		return true
	}

	return false
}

// SetAggType gets a reference to the given string and assigns it to the AggType field.
func (o *UsageAttributionAggregatesBody) SetAggType(v string) {
	o.AggType = &v
}

// GetField returns the Field field value if set, zero value otherwise.
func (o *UsageAttributionAggregatesBody) GetField() string {
	if o == nil || o.Field == nil {
		var ret string
		return ret
	}
	return *o.Field
}

// GetFieldOk returns a tuple with the Field field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionAggregatesBody) GetFieldOk() (*string, bool) {
	if o == nil || o.Field == nil {
		return nil, false
	}
	return o.Field, true
}

// HasField returns a boolean if a field has been set.
func (o *UsageAttributionAggregatesBody) HasField() bool {
	if o != nil && o.Field != nil {
		return true
	}

	return false
}

// SetField gets a reference to the given string and assigns it to the Field field.
func (o *UsageAttributionAggregatesBody) SetField(v string) {
	o.Field = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *UsageAttributionAggregatesBody) GetValue() float64 {
	if o == nil || o.Value == nil {
		var ret float64
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionAggregatesBody) GetValueOk() (*float64, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *UsageAttributionAggregatesBody) HasValue() bool {
	if o != nil && o.Value != nil {
		return true
	}

	return false
}

// SetValue gets a reference to the given float64 and assigns it to the Value field.
func (o *UsageAttributionAggregatesBody) SetValue(v float64) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageAttributionAggregatesBody) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AggType != nil {
		toSerialize["agg_type"] = o.AggType
	}
	if o.Field != nil {
		toSerialize["field"] = o.Field
	}
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageAttributionAggregatesBody) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AggType *string  `json:"agg_type,omitempty"`
		Field   *string  `json:"field,omitempty"`
		Value   *float64 `json:"value,omitempty"`
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
	o.AggType = all.AggType
	o.Field = all.Field
	o.Value = all.Value
	return nil
}

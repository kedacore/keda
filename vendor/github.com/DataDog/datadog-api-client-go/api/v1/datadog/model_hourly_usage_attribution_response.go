// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HourlyUsageAttributionResponse Response containing the hourly usage attribution by tag(s).
type HourlyUsageAttributionResponse struct {
	// The object containing document metadata.
	Metadata *HourlyUsageAttributionMetadata `json:"metadata,omitempty"`
	// Get the hourly usage attribution by tag(s).
	Usage []HourlyUsageAttributionBody `json:"usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageAttributionResponse instantiates a new HourlyUsageAttributionResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageAttributionResponse() *HourlyUsageAttributionResponse {
	this := HourlyUsageAttributionResponse{}
	return &this
}

// NewHourlyUsageAttributionResponseWithDefaults instantiates a new HourlyUsageAttributionResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageAttributionResponseWithDefaults() *HourlyUsageAttributionResponse {
	this := HourlyUsageAttributionResponse{}
	return &this
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *HourlyUsageAttributionResponse) GetMetadata() HourlyUsageAttributionMetadata {
	if o == nil || o.Metadata == nil {
		var ret HourlyUsageAttributionMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionResponse) GetMetadataOk() (*HourlyUsageAttributionMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *HourlyUsageAttributionResponse) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given HourlyUsageAttributionMetadata and assigns it to the Metadata field.
func (o *HourlyUsageAttributionResponse) SetMetadata(v HourlyUsageAttributionMetadata) {
	o.Metadata = &v
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *HourlyUsageAttributionResponse) GetUsage() []HourlyUsageAttributionBody {
	if o == nil || o.Usage == nil {
		var ret []HourlyUsageAttributionBody
		return ret
	}
	return o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionResponse) GetUsageOk() (*[]HourlyUsageAttributionBody, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return &o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *HourlyUsageAttributionResponse) HasUsage() bool {
	if o != nil && o.Usage != nil {
		return true
	}

	return false
}

// SetUsage gets a reference to the given []HourlyUsageAttributionBody and assigns it to the Usage field.
func (o *HourlyUsageAttributionResponse) SetUsage(v []HourlyUsageAttributionBody) {
	o.Usage = v
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageAttributionResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.Usage != nil {
		toSerialize["usage"] = o.Usage
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageAttributionResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Metadata *HourlyUsageAttributionMetadata `json:"metadata,omitempty"`
		Usage    []HourlyUsageAttributionBody    `json:"usage,omitempty"`
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
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Usage = all.Usage
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorSearchResponseCounts The counts of monitors per different criteria.
type MonitorSearchResponseCounts struct {
	// Search facets.
	Muted []MonitorSearchCountItem `json:"muted,omitempty"`
	// Search facets.
	Status []MonitorSearchCountItem `json:"status,omitempty"`
	// Search facets.
	Tag []MonitorSearchCountItem `json:"tag,omitempty"`
	// Search facets.
	Type []MonitorSearchCountItem `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorSearchResponseCounts instantiates a new MonitorSearchResponseCounts object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorSearchResponseCounts() *MonitorSearchResponseCounts {
	this := MonitorSearchResponseCounts{}
	return &this
}

// NewMonitorSearchResponseCountsWithDefaults instantiates a new MonitorSearchResponseCounts object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorSearchResponseCountsWithDefaults() *MonitorSearchResponseCounts {
	this := MonitorSearchResponseCounts{}
	return &this
}

// GetMuted returns the Muted field value if set, zero value otherwise.
func (o *MonitorSearchResponseCounts) GetMuted() []MonitorSearchCountItem {
	if o == nil || o.Muted == nil {
		var ret []MonitorSearchCountItem
		return ret
	}
	return o.Muted
}

// GetMutedOk returns a tuple with the Muted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseCounts) GetMutedOk() (*[]MonitorSearchCountItem, bool) {
	if o == nil || o.Muted == nil {
		return nil, false
	}
	return &o.Muted, true
}

// HasMuted returns a boolean if a field has been set.
func (o *MonitorSearchResponseCounts) HasMuted() bool {
	if o != nil && o.Muted != nil {
		return true
	}

	return false
}

// SetMuted gets a reference to the given []MonitorSearchCountItem and assigns it to the Muted field.
func (o *MonitorSearchResponseCounts) SetMuted(v []MonitorSearchCountItem) {
	o.Muted = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *MonitorSearchResponseCounts) GetStatus() []MonitorSearchCountItem {
	if o == nil || o.Status == nil {
		var ret []MonitorSearchCountItem
		return ret
	}
	return o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseCounts) GetStatusOk() (*[]MonitorSearchCountItem, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return &o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *MonitorSearchResponseCounts) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given []MonitorSearchCountItem and assigns it to the Status field.
func (o *MonitorSearchResponseCounts) SetStatus(v []MonitorSearchCountItem) {
	o.Status = v
}

// GetTag returns the Tag field value if set, zero value otherwise.
func (o *MonitorSearchResponseCounts) GetTag() []MonitorSearchCountItem {
	if o == nil || o.Tag == nil {
		var ret []MonitorSearchCountItem
		return ret
	}
	return o.Tag
}

// GetTagOk returns a tuple with the Tag field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseCounts) GetTagOk() (*[]MonitorSearchCountItem, bool) {
	if o == nil || o.Tag == nil {
		return nil, false
	}
	return &o.Tag, true
}

// HasTag returns a boolean if a field has been set.
func (o *MonitorSearchResponseCounts) HasTag() bool {
	if o != nil && o.Tag != nil {
		return true
	}

	return false
}

// SetTag gets a reference to the given []MonitorSearchCountItem and assigns it to the Tag field.
func (o *MonitorSearchResponseCounts) SetTag(v []MonitorSearchCountItem) {
	o.Tag = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MonitorSearchResponseCounts) GetType() []MonitorSearchCountItem {
	if o == nil || o.Type == nil {
		var ret []MonitorSearchCountItem
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponseCounts) GetTypeOk() (*[]MonitorSearchCountItem, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return &o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MonitorSearchResponseCounts) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given []MonitorSearchCountItem and assigns it to the Type field.
func (o *MonitorSearchResponseCounts) SetType(v []MonitorSearchCountItem) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorSearchResponseCounts) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Muted != nil {
		toSerialize["muted"] = o.Muted
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Tag != nil {
		toSerialize["tag"] = o.Tag
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorSearchResponseCounts) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Muted  []MonitorSearchCountItem `json:"muted,omitempty"`
		Status []MonitorSearchCountItem `json:"status,omitempty"`
		Tag    []MonitorSearchCountItem `json:"tag,omitempty"`
		Type   []MonitorSearchCountItem `json:"type,omitempty"`
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
	o.Muted = all.Muted
	o.Status = all.Status
	o.Tag = all.Tag
	o.Type = all.Type
	return nil
}

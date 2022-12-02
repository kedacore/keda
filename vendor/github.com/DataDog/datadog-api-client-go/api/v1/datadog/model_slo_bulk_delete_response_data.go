// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOBulkDeleteResponseData An array of service level objective objects.
type SLOBulkDeleteResponseData struct {
	// An array of service level objective object IDs that indicates
	// which objects that were completely deleted.
	Deleted []string `json:"deleted,omitempty"`
	// An array of service level objective object IDs that indicates
	// which objects that were modified (objects for which at least one
	// threshold was deleted, but that were not completely deleted).
	Updated []string `json:"updated,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOBulkDeleteResponseData instantiates a new SLOBulkDeleteResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOBulkDeleteResponseData() *SLOBulkDeleteResponseData {
	this := SLOBulkDeleteResponseData{}
	return &this
}

// NewSLOBulkDeleteResponseDataWithDefaults instantiates a new SLOBulkDeleteResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOBulkDeleteResponseDataWithDefaults() *SLOBulkDeleteResponseData {
	this := SLOBulkDeleteResponseData{}
	return &this
}

// GetDeleted returns the Deleted field value if set, zero value otherwise.
func (o *SLOBulkDeleteResponseData) GetDeleted() []string {
	if o == nil || o.Deleted == nil {
		var ret []string
		return ret
	}
	return o.Deleted
}

// GetDeletedOk returns a tuple with the Deleted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOBulkDeleteResponseData) GetDeletedOk() (*[]string, bool) {
	if o == nil || o.Deleted == nil {
		return nil, false
	}
	return &o.Deleted, true
}

// HasDeleted returns a boolean if a field has been set.
func (o *SLOBulkDeleteResponseData) HasDeleted() bool {
	if o != nil && o.Deleted != nil {
		return true
	}

	return false
}

// SetDeleted gets a reference to the given []string and assigns it to the Deleted field.
func (o *SLOBulkDeleteResponseData) SetDeleted(v []string) {
	o.Deleted = v
}

// GetUpdated returns the Updated field value if set, zero value otherwise.
func (o *SLOBulkDeleteResponseData) GetUpdated() []string {
	if o == nil || o.Updated == nil {
		var ret []string
		return ret
	}
	return o.Updated
}

// GetUpdatedOk returns a tuple with the Updated field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOBulkDeleteResponseData) GetUpdatedOk() (*[]string, bool) {
	if o == nil || o.Updated == nil {
		return nil, false
	}
	return &o.Updated, true
}

// HasUpdated returns a boolean if a field has been set.
func (o *SLOBulkDeleteResponseData) HasUpdated() bool {
	if o != nil && o.Updated != nil {
		return true
	}

	return false
}

// SetUpdated gets a reference to the given []string and assigns it to the Updated field.
func (o *SLOBulkDeleteResponseData) SetUpdated(v []string) {
	o.Updated = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOBulkDeleteResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Deleted != nil {
		toSerialize["deleted"] = o.Deleted
	}
	if o.Updated != nil {
		toSerialize["updated"] = o.Updated
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOBulkDeleteResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Deleted []string `json:"deleted,omitempty"`
		Updated []string `json:"updated,omitempty"`
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
	o.Deleted = all.Deleted
	o.Updated = all.Updated
	return nil
}

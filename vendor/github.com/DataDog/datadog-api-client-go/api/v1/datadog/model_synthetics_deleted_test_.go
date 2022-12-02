// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// SyntheticsDeletedTest Object containing a deleted Synthetic test ID with the associated
// deletion timestamp.
type SyntheticsDeletedTest struct {
	// Deletion timestamp of the Synthetic test ID.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// The Synthetic test ID deleted.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsDeletedTest instantiates a new SyntheticsDeletedTest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsDeletedTest() *SyntheticsDeletedTest {
	this := SyntheticsDeletedTest{}
	return &this
}

// NewSyntheticsDeletedTestWithDefaults instantiates a new SyntheticsDeletedTest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsDeletedTestWithDefaults() *SyntheticsDeletedTest {
	this := SyntheticsDeletedTest{}
	return &this
}

// GetDeletedAt returns the DeletedAt field value if set, zero value otherwise.
func (o *SyntheticsDeletedTest) GetDeletedAt() time.Time {
	if o == nil || o.DeletedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.DeletedAt
}

// GetDeletedAtOk returns a tuple with the DeletedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsDeletedTest) GetDeletedAtOk() (*time.Time, bool) {
	if o == nil || o.DeletedAt == nil {
		return nil, false
	}
	return o.DeletedAt, true
}

// HasDeletedAt returns a boolean if a field has been set.
func (o *SyntheticsDeletedTest) HasDeletedAt() bool {
	if o != nil && o.DeletedAt != nil {
		return true
	}

	return false
}

// SetDeletedAt gets a reference to the given time.Time and assigns it to the DeletedAt field.
func (o *SyntheticsDeletedTest) SetDeletedAt(v time.Time) {
	o.DeletedAt = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *SyntheticsDeletedTest) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsDeletedTest) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *SyntheticsDeletedTest) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *SyntheticsDeletedTest) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsDeletedTest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DeletedAt != nil {
		if o.DeletedAt.Nanosecond() == 0 {
			toSerialize["deleted_at"] = o.DeletedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["deleted_at"] = o.DeletedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsDeletedTest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DeletedAt *time.Time `json:"deleted_at,omitempty"`
		PublicId  *string    `json:"public_id,omitempty"`
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
	o.DeletedAt = all.DeletedAt
	o.PublicId = all.PublicId
	return nil
}

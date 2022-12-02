// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamColumn Widget column.
type ListStreamColumn struct {
	// Widget column field.
	Field string `json:"field"`
	// Widget column width.
	Width ListStreamColumnWidth `json:"width"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewListStreamColumn instantiates a new ListStreamColumn object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamColumn(field string, width ListStreamColumnWidth) *ListStreamColumn {
	this := ListStreamColumn{}
	this.Field = field
	this.Width = width
	return &this
}

// NewListStreamColumnWithDefaults instantiates a new ListStreamColumn object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamColumnWithDefaults() *ListStreamColumn {
	this := ListStreamColumn{}
	return &this
}

// GetField returns the Field field value.
func (o *ListStreamColumn) GetField() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Field
}

// GetFieldOk returns a tuple with the Field field value
// and a boolean to check if the value has been set.
func (o *ListStreamColumn) GetFieldOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Field, true
}

// SetField sets field value.
func (o *ListStreamColumn) SetField(v string) {
	o.Field = v
}

// GetWidth returns the Width field value.
func (o *ListStreamColumn) GetWidth() ListStreamColumnWidth {
	if o == nil {
		var ret ListStreamColumnWidth
		return ret
	}
	return o.Width
}

// GetWidthOk returns a tuple with the Width field value
// and a boolean to check if the value has been set.
func (o *ListStreamColumn) GetWidthOk() (*ListStreamColumnWidth, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Width, true
}

// SetWidth sets field value.
func (o *ListStreamColumn) SetWidth(v ListStreamColumnWidth) {
	o.Width = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamColumn) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["field"] = o.Field
	toSerialize["width"] = o.Width

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ListStreamColumn) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Field *string                `json:"field"`
		Width *ListStreamColumnWidth `json:"width"`
	}{}
	all := struct {
		Field string                `json:"field"`
		Width ListStreamColumnWidth `json:"width"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Field == nil {
		return fmt.Errorf("Required field field missing")
	}
	if required.Width == nil {
		return fmt.Errorf("Required field width missing")
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
	if v := all.Width; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Field = all.Field
	o.Width = all.Width
	return nil
}

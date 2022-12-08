// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetFieldSort Which column and order to sort by
type WidgetFieldSort struct {
	// Facet path for the column
	Column string `json:"column"`
	// Widget sorting methods.
	Order WidgetSort `json:"order"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetFieldSort instantiates a new WidgetFieldSort object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetFieldSort(column string, order WidgetSort) *WidgetFieldSort {
	this := WidgetFieldSort{}
	this.Column = column
	this.Order = order
	return &this
}

// NewWidgetFieldSortWithDefaults instantiates a new WidgetFieldSort object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetFieldSortWithDefaults() *WidgetFieldSort {
	this := WidgetFieldSort{}
	return &this
}

// GetColumn returns the Column field value.
func (o *WidgetFieldSort) GetColumn() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Column
}

// GetColumnOk returns a tuple with the Column field value
// and a boolean to check if the value has been set.
func (o *WidgetFieldSort) GetColumnOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Column, true
}

// SetColumn sets field value.
func (o *WidgetFieldSort) SetColumn(v string) {
	o.Column = v
}

// GetOrder returns the Order field value.
func (o *WidgetFieldSort) GetOrder() WidgetSort {
	if o == nil {
		var ret WidgetSort
		return ret
	}
	return o.Order
}

// GetOrderOk returns a tuple with the Order field value
// and a boolean to check if the value has been set.
func (o *WidgetFieldSort) GetOrderOk() (*WidgetSort, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Order, true
}

// SetOrder sets field value.
func (o *WidgetFieldSort) SetOrder(v WidgetSort) {
	o.Order = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetFieldSort) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["column"] = o.Column
	toSerialize["order"] = o.Order

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetFieldSort) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Column *string     `json:"column"`
		Order  *WidgetSort `json:"order"`
	}{}
	all := struct {
		Column string     `json:"column"`
		Order  WidgetSort `json:"order"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Column == nil {
		return fmt.Errorf("Required field column missing")
	}
	if required.Order == nil {
		return fmt.Errorf("Required field order missing")
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
	if v := all.Order; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Column = all.Column
	o.Order = all.Order
	return nil
}

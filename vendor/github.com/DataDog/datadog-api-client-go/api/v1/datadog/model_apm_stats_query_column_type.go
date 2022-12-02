// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ApmStatsQueryColumnType Column properties.
type ApmStatsQueryColumnType struct {
	// A user-assigned alias for the column.
	Alias *string `json:"alias,omitempty"`
	// Define a display mode for the table cell.
	CellDisplayMode *TableWidgetCellDisplayMode `json:"cell_display_mode,omitempty"`
	// Column name.
	Name string `json:"name"`
	// Widget sorting methods.
	Order *WidgetSort `json:"order,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewApmStatsQueryColumnType instantiates a new ApmStatsQueryColumnType object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewApmStatsQueryColumnType(name string) *ApmStatsQueryColumnType {
	this := ApmStatsQueryColumnType{}
	this.Name = name
	return &this
}

// NewApmStatsQueryColumnTypeWithDefaults instantiates a new ApmStatsQueryColumnType object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewApmStatsQueryColumnTypeWithDefaults() *ApmStatsQueryColumnType {
	this := ApmStatsQueryColumnType{}
	return &this
}

// GetAlias returns the Alias field value if set, zero value otherwise.
func (o *ApmStatsQueryColumnType) GetAlias() string {
	if o == nil || o.Alias == nil {
		var ret string
		return ret
	}
	return *o.Alias
}

// GetAliasOk returns a tuple with the Alias field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryColumnType) GetAliasOk() (*string, bool) {
	if o == nil || o.Alias == nil {
		return nil, false
	}
	return o.Alias, true
}

// HasAlias returns a boolean if a field has been set.
func (o *ApmStatsQueryColumnType) HasAlias() bool {
	if o != nil && o.Alias != nil {
		return true
	}

	return false
}

// SetAlias gets a reference to the given string and assigns it to the Alias field.
func (o *ApmStatsQueryColumnType) SetAlias(v string) {
	o.Alias = &v
}

// GetCellDisplayMode returns the CellDisplayMode field value if set, zero value otherwise.
func (o *ApmStatsQueryColumnType) GetCellDisplayMode() TableWidgetCellDisplayMode {
	if o == nil || o.CellDisplayMode == nil {
		var ret TableWidgetCellDisplayMode
		return ret
	}
	return *o.CellDisplayMode
}

// GetCellDisplayModeOk returns a tuple with the CellDisplayMode field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryColumnType) GetCellDisplayModeOk() (*TableWidgetCellDisplayMode, bool) {
	if o == nil || o.CellDisplayMode == nil {
		return nil, false
	}
	return o.CellDisplayMode, true
}

// HasCellDisplayMode returns a boolean if a field has been set.
func (o *ApmStatsQueryColumnType) HasCellDisplayMode() bool {
	if o != nil && o.CellDisplayMode != nil {
		return true
	}

	return false
}

// SetCellDisplayMode gets a reference to the given TableWidgetCellDisplayMode and assigns it to the CellDisplayMode field.
func (o *ApmStatsQueryColumnType) SetCellDisplayMode(v TableWidgetCellDisplayMode) {
	o.CellDisplayMode = &v
}

// GetName returns the Name field value.
func (o *ApmStatsQueryColumnType) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryColumnType) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *ApmStatsQueryColumnType) SetName(v string) {
	o.Name = v
}

// GetOrder returns the Order field value if set, zero value otherwise.
func (o *ApmStatsQueryColumnType) GetOrder() WidgetSort {
	if o == nil || o.Order == nil {
		var ret WidgetSort
		return ret
	}
	return *o.Order
}

// GetOrderOk returns a tuple with the Order field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryColumnType) GetOrderOk() (*WidgetSort, bool) {
	if o == nil || o.Order == nil {
		return nil, false
	}
	return o.Order, true
}

// HasOrder returns a boolean if a field has been set.
func (o *ApmStatsQueryColumnType) HasOrder() bool {
	if o != nil && o.Order != nil {
		return true
	}

	return false
}

// SetOrder gets a reference to the given WidgetSort and assigns it to the Order field.
func (o *ApmStatsQueryColumnType) SetOrder(v WidgetSort) {
	o.Order = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ApmStatsQueryColumnType) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Alias != nil {
		toSerialize["alias"] = o.Alias
	}
	if o.CellDisplayMode != nil {
		toSerialize["cell_display_mode"] = o.CellDisplayMode
	}
	toSerialize["name"] = o.Name
	if o.Order != nil {
		toSerialize["order"] = o.Order
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ApmStatsQueryColumnType) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Alias           *string                     `json:"alias,omitempty"`
		CellDisplayMode *TableWidgetCellDisplayMode `json:"cell_display_mode,omitempty"`
		Name            string                      `json:"name"`
		Order           *WidgetSort                 `json:"order,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if v := all.CellDisplayMode; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Order; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Alias = all.Alias
	o.CellDisplayMode = all.CellDisplayMode
	o.Name = all.Name
	o.Order = all.Order
	return nil
}

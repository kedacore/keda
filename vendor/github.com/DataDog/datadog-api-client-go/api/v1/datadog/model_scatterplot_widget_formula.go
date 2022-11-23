// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ScatterplotWidgetFormula Formula to be used in a Scatterplot widget query.
type ScatterplotWidgetFormula struct {
	// Expression alias.
	Alias *string `json:"alias,omitempty"`
	// Dimension of the Scatterplot.
	Dimension ScatterplotDimension `json:"dimension"`
	// String expression built from queries, formulas, and functions.
	Formula string `json:"formula"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewScatterplotWidgetFormula instantiates a new ScatterplotWidgetFormula object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScatterplotWidgetFormula(dimension ScatterplotDimension, formula string) *ScatterplotWidgetFormula {
	this := ScatterplotWidgetFormula{}
	this.Dimension = dimension
	this.Formula = formula
	return &this
}

// NewScatterplotWidgetFormulaWithDefaults instantiates a new ScatterplotWidgetFormula object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScatterplotWidgetFormulaWithDefaults() *ScatterplotWidgetFormula {
	this := ScatterplotWidgetFormula{}
	return &this
}

// GetAlias returns the Alias field value if set, zero value otherwise.
func (o *ScatterplotWidgetFormula) GetAlias() string {
	if o == nil || o.Alias == nil {
		var ret string
		return ret
	}
	return *o.Alias
}

// GetAliasOk returns a tuple with the Alias field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterplotWidgetFormula) GetAliasOk() (*string, bool) {
	if o == nil || o.Alias == nil {
		return nil, false
	}
	return o.Alias, true
}

// HasAlias returns a boolean if a field has been set.
func (o *ScatterplotWidgetFormula) HasAlias() bool {
	if o != nil && o.Alias != nil {
		return true
	}

	return false
}

// SetAlias gets a reference to the given string and assigns it to the Alias field.
func (o *ScatterplotWidgetFormula) SetAlias(v string) {
	o.Alias = &v
}

// GetDimension returns the Dimension field value.
func (o *ScatterplotWidgetFormula) GetDimension() ScatterplotDimension {
	if o == nil {
		var ret ScatterplotDimension
		return ret
	}
	return o.Dimension
}

// GetDimensionOk returns a tuple with the Dimension field value
// and a boolean to check if the value has been set.
func (o *ScatterplotWidgetFormula) GetDimensionOk() (*ScatterplotDimension, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Dimension, true
}

// SetDimension sets field value.
func (o *ScatterplotWidgetFormula) SetDimension(v ScatterplotDimension) {
	o.Dimension = v
}

// GetFormula returns the Formula field value.
func (o *ScatterplotWidgetFormula) GetFormula() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Formula
}

// GetFormulaOk returns a tuple with the Formula field value
// and a boolean to check if the value has been set.
func (o *ScatterplotWidgetFormula) GetFormulaOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Formula, true
}

// SetFormula sets field value.
func (o *ScatterplotWidgetFormula) SetFormula(v string) {
	o.Formula = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScatterplotWidgetFormula) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Alias != nil {
		toSerialize["alias"] = o.Alias
	}
	toSerialize["dimension"] = o.Dimension
	toSerialize["formula"] = o.Formula

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScatterplotWidgetFormula) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Dimension *ScatterplotDimension `json:"dimension"`
		Formula   *string               `json:"formula"`
	}{}
	all := struct {
		Alias     *string              `json:"alias,omitempty"`
		Dimension ScatterplotDimension `json:"dimension"`
		Formula   string               `json:"formula"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Dimension == nil {
		return fmt.Errorf("Required field dimension missing")
	}
	if required.Formula == nil {
		return fmt.Errorf("Required field formula missing")
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
	if v := all.Dimension; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Alias = all.Alias
	o.Dimension = all.Dimension
	o.Formula = all.Formula
	return nil
}

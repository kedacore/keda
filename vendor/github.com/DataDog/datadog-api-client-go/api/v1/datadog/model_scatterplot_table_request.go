// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// ScatterplotTableRequest Scatterplot request containing formulas and functions.
type ScatterplotTableRequest struct {
	// List of Scatterplot formulas that operate on queries.
	Formulas []ScatterplotWidgetFormula `json:"formulas,omitempty"`
	// List of queries that can be returned directly or used in formulas.
	Queries []FormulaAndFunctionQueryDefinition `json:"queries,omitempty"`
	// Timeseries or Scalar response.
	ResponseFormat *FormulaAndFunctionResponseFormat `json:"response_format,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewScatterplotTableRequest instantiates a new ScatterplotTableRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScatterplotTableRequest() *ScatterplotTableRequest {
	this := ScatterplotTableRequest{}
	return &this
}

// NewScatterplotTableRequestWithDefaults instantiates a new ScatterplotTableRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScatterplotTableRequestWithDefaults() *ScatterplotTableRequest {
	this := ScatterplotTableRequest{}
	return &this
}

// GetFormulas returns the Formulas field value if set, zero value otherwise.
func (o *ScatterplotTableRequest) GetFormulas() []ScatterplotWidgetFormula {
	if o == nil || o.Formulas == nil {
		var ret []ScatterplotWidgetFormula
		return ret
	}
	return o.Formulas
}

// GetFormulasOk returns a tuple with the Formulas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterplotTableRequest) GetFormulasOk() (*[]ScatterplotWidgetFormula, bool) {
	if o == nil || o.Formulas == nil {
		return nil, false
	}
	return &o.Formulas, true
}

// HasFormulas returns a boolean if a field has been set.
func (o *ScatterplotTableRequest) HasFormulas() bool {
	if o != nil && o.Formulas != nil {
		return true
	}

	return false
}

// SetFormulas gets a reference to the given []ScatterplotWidgetFormula and assigns it to the Formulas field.
func (o *ScatterplotTableRequest) SetFormulas(v []ScatterplotWidgetFormula) {
	o.Formulas = v
}

// GetQueries returns the Queries field value if set, zero value otherwise.
func (o *ScatterplotTableRequest) GetQueries() []FormulaAndFunctionQueryDefinition {
	if o == nil || o.Queries == nil {
		var ret []FormulaAndFunctionQueryDefinition
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterplotTableRequest) GetQueriesOk() (*[]FormulaAndFunctionQueryDefinition, bool) {
	if o == nil || o.Queries == nil {
		return nil, false
	}
	return &o.Queries, true
}

// HasQueries returns a boolean if a field has been set.
func (o *ScatterplotTableRequest) HasQueries() bool {
	if o != nil && o.Queries != nil {
		return true
	}

	return false
}

// SetQueries gets a reference to the given []FormulaAndFunctionQueryDefinition and assigns it to the Queries field.
func (o *ScatterplotTableRequest) SetQueries(v []FormulaAndFunctionQueryDefinition) {
	o.Queries = v
}

// GetResponseFormat returns the ResponseFormat field value if set, zero value otherwise.
func (o *ScatterplotTableRequest) GetResponseFormat() FormulaAndFunctionResponseFormat {
	if o == nil || o.ResponseFormat == nil {
		var ret FormulaAndFunctionResponseFormat
		return ret
	}
	return *o.ResponseFormat
}

// GetResponseFormatOk returns a tuple with the ResponseFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScatterplotTableRequest) GetResponseFormatOk() (*FormulaAndFunctionResponseFormat, bool) {
	if o == nil || o.ResponseFormat == nil {
		return nil, false
	}
	return o.ResponseFormat, true
}

// HasResponseFormat returns a boolean if a field has been set.
func (o *ScatterplotTableRequest) HasResponseFormat() bool {
	if o != nil && o.ResponseFormat != nil {
		return true
	}

	return false
}

// SetResponseFormat gets a reference to the given FormulaAndFunctionResponseFormat and assigns it to the ResponseFormat field.
func (o *ScatterplotTableRequest) SetResponseFormat(v FormulaAndFunctionResponseFormat) {
	o.ResponseFormat = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScatterplotTableRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Formulas != nil {
		toSerialize["formulas"] = o.Formulas
	}
	if o.Queries != nil {
		toSerialize["queries"] = o.Queries
	}
	if o.ResponseFormat != nil {
		toSerialize["response_format"] = o.ResponseFormat
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScatterplotTableRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Formulas       []ScatterplotWidgetFormula          `json:"formulas,omitempty"`
		Queries        []FormulaAndFunctionQueryDefinition `json:"queries,omitempty"`
		ResponseFormat *FormulaAndFunctionResponseFormat   `json:"response_format,omitempty"`
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
	if v := all.ResponseFormat; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Formulas = all.Formulas
	o.Queries = all.Queries
	o.ResponseFormat = all.ResponseFormat
	return nil
}

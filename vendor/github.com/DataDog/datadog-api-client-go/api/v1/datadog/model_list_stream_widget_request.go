// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamWidgetRequest Updated list stream widget.
type ListStreamWidgetRequest struct {
	// Widget columns.
	Columns []ListStreamColumn `json:"columns"`
	// Updated list stream widget.
	Query ListStreamQuery `json:"query"`
	// Widget response format.
	ResponseFormat ListStreamResponseFormat `json:"response_format"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewListStreamWidgetRequest instantiates a new ListStreamWidgetRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamWidgetRequest(columns []ListStreamColumn, query ListStreamQuery, responseFormat ListStreamResponseFormat) *ListStreamWidgetRequest {
	this := ListStreamWidgetRequest{}
	this.Columns = columns
	this.Query = query
	this.ResponseFormat = responseFormat
	return &this
}

// NewListStreamWidgetRequestWithDefaults instantiates a new ListStreamWidgetRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamWidgetRequestWithDefaults() *ListStreamWidgetRequest {
	this := ListStreamWidgetRequest{}
	return &this
}

// GetColumns returns the Columns field value.
func (o *ListStreamWidgetRequest) GetColumns() []ListStreamColumn {
	if o == nil {
		var ret []ListStreamColumn
		return ret
	}
	return o.Columns
}

// GetColumnsOk returns a tuple with the Columns field value
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetRequest) GetColumnsOk() (*[]ListStreamColumn, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Columns, true
}

// SetColumns sets field value.
func (o *ListStreamWidgetRequest) SetColumns(v []ListStreamColumn) {
	o.Columns = v
}

// GetQuery returns the Query field value.
func (o *ListStreamWidgetRequest) GetQuery() ListStreamQuery {
	if o == nil {
		var ret ListStreamQuery
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetRequest) GetQueryOk() (*ListStreamQuery, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *ListStreamWidgetRequest) SetQuery(v ListStreamQuery) {
	o.Query = v
}

// GetResponseFormat returns the ResponseFormat field value.
func (o *ListStreamWidgetRequest) GetResponseFormat() ListStreamResponseFormat {
	if o == nil {
		var ret ListStreamResponseFormat
		return ret
	}
	return o.ResponseFormat
}

// GetResponseFormatOk returns a tuple with the ResponseFormat field value
// and a boolean to check if the value has been set.
func (o *ListStreamWidgetRequest) GetResponseFormatOk() (*ListStreamResponseFormat, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ResponseFormat, true
}

// SetResponseFormat sets field value.
func (o *ListStreamWidgetRequest) SetResponseFormat(v ListStreamResponseFormat) {
	o.ResponseFormat = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamWidgetRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["columns"] = o.Columns
	toSerialize["query"] = o.Query
	toSerialize["response_format"] = o.ResponseFormat

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ListStreamWidgetRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Columns        *[]ListStreamColumn       `json:"columns"`
		Query          *ListStreamQuery          `json:"query"`
		ResponseFormat *ListStreamResponseFormat `json:"response_format"`
	}{}
	all := struct {
		Columns        []ListStreamColumn       `json:"columns"`
		Query          ListStreamQuery          `json:"query"`
		ResponseFormat ListStreamResponseFormat `json:"response_format"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Columns == nil {
		return fmt.Errorf("Required field columns missing")
	}
	if required.Query == nil {
		return fmt.Errorf("Required field query missing")
	}
	if required.ResponseFormat == nil {
		return fmt.Errorf("Required field response_format missing")
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
	if v := all.ResponseFormat; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Columns = all.Columns
	if all.Query.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Query = all.Query
	o.ResponseFormat = all.ResponseFormat
	return nil
}

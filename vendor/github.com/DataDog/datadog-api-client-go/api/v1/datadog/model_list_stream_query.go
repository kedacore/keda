// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ListStreamQuery Updated list stream widget.
type ListStreamQuery struct {
	// Source from which to query items to display in the stream.
	DataSource ListStreamSource `json:"data_source"`
	// List of indexes.
	Indexes []string `json:"indexes,omitempty"`
	// Widget query.
	QueryString string `json:"query_string"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewListStreamQuery instantiates a new ListStreamQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamQuery(dataSource ListStreamSource, queryString string) *ListStreamQuery {
	this := ListStreamQuery{}
	this.DataSource = dataSource
	this.QueryString = queryString
	return &this
}

// NewListStreamQueryWithDefaults instantiates a new ListStreamQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamQueryWithDefaults() *ListStreamQuery {
	this := ListStreamQuery{}
	var dataSource ListStreamSource = LISTSTREAMSOURCE_APM_ISSUE_STREAM
	this.DataSource = dataSource
	return &this
}

// GetDataSource returns the DataSource field value.
func (o *ListStreamQuery) GetDataSource() ListStreamSource {
	if o == nil {
		var ret ListStreamSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetDataSourceOk() (*ListStreamSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *ListStreamQuery) SetDataSource(v ListStreamSource) {
	o.DataSource = v
}

// GetIndexes returns the Indexes field value if set, zero value otherwise.
func (o *ListStreamQuery) GetIndexes() []string {
	if o == nil || o.Indexes == nil {
		var ret []string
		return ret
	}
	return o.Indexes
}

// GetIndexesOk returns a tuple with the Indexes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetIndexesOk() (*[]string, bool) {
	if o == nil || o.Indexes == nil {
		return nil, false
	}
	return &o.Indexes, true
}

// HasIndexes returns a boolean if a field has been set.
func (o *ListStreamQuery) HasIndexes() bool {
	if o != nil && o.Indexes != nil {
		return true
	}

	return false
}

// SetIndexes gets a reference to the given []string and assigns it to the Indexes field.
func (o *ListStreamQuery) SetIndexes(v []string) {
	o.Indexes = v
}

// GetQueryString returns the QueryString field value.
func (o *ListStreamQuery) GetQueryString() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.QueryString
}

// GetQueryStringOk returns a tuple with the QueryString field value
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetQueryStringOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.QueryString, true
}

// SetQueryString sets field value.
func (o *ListStreamQuery) SetQueryString(v string) {
	o.QueryString = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data_source"] = o.DataSource
	if o.Indexes != nil {
		toSerialize["indexes"] = o.Indexes
	}
	toSerialize["query_string"] = o.QueryString

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ListStreamQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource  *ListStreamSource `json:"data_source"`
		QueryString *string           `json:"query_string"`
	}{}
	all := struct {
		DataSource  ListStreamSource `json:"data_source"`
		Indexes     []string         `json:"indexes,omitempty"`
		QueryString string           `json:"query_string"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("Required field data_source missing")
	}
	if required.QueryString == nil {
		return fmt.Errorf("Required field query_string missing")
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
	if v := all.DataSource; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.DataSource = all.DataSource
	o.Indexes = all.Indexes
	o.QueryString = all.QueryString
	return nil
}

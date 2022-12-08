// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookTimeseriesCellAttributes The attributes of a notebook `timeseries` cell.
type NotebookTimeseriesCellAttributes struct {
	// The timeseries visualization allows you to display the evolution of one or more metrics, log events, or Indexed Spans over time.
	Definition TimeseriesWidgetDefinition `json:"definition"`
	// The size of the graph.
	GraphSize *NotebookGraphSize `json:"graph_size,omitempty"`
	// Object describing how to split the graph to display multiple visualizations per request.
	SplitBy *NotebookSplitBy `json:"split_by,omitempty"`
	// Timeframe for the notebook cell. When 'null', the notebook global time is used.
	Time NullableNotebookCellTime `json:"time,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookTimeseriesCellAttributes instantiates a new NotebookTimeseriesCellAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookTimeseriesCellAttributes(definition TimeseriesWidgetDefinition) *NotebookTimeseriesCellAttributes {
	this := NotebookTimeseriesCellAttributes{}
	this.Definition = definition
	return &this
}

// NewNotebookTimeseriesCellAttributesWithDefaults instantiates a new NotebookTimeseriesCellAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookTimeseriesCellAttributesWithDefaults() *NotebookTimeseriesCellAttributes {
	this := NotebookTimeseriesCellAttributes{}
	return &this
}

// GetDefinition returns the Definition field value.
func (o *NotebookTimeseriesCellAttributes) GetDefinition() TimeseriesWidgetDefinition {
	if o == nil {
		var ret TimeseriesWidgetDefinition
		return ret
	}
	return o.Definition
}

// GetDefinitionOk returns a tuple with the Definition field value
// and a boolean to check if the value has been set.
func (o *NotebookTimeseriesCellAttributes) GetDefinitionOk() (*TimeseriesWidgetDefinition, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Definition, true
}

// SetDefinition sets field value.
func (o *NotebookTimeseriesCellAttributes) SetDefinition(v TimeseriesWidgetDefinition) {
	o.Definition = v
}

// GetGraphSize returns the GraphSize field value if set, zero value otherwise.
func (o *NotebookTimeseriesCellAttributes) GetGraphSize() NotebookGraphSize {
	if o == nil || o.GraphSize == nil {
		var ret NotebookGraphSize
		return ret
	}
	return *o.GraphSize
}

// GetGraphSizeOk returns a tuple with the GraphSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookTimeseriesCellAttributes) GetGraphSizeOk() (*NotebookGraphSize, bool) {
	if o == nil || o.GraphSize == nil {
		return nil, false
	}
	return o.GraphSize, true
}

// HasGraphSize returns a boolean if a field has been set.
func (o *NotebookTimeseriesCellAttributes) HasGraphSize() bool {
	if o != nil && o.GraphSize != nil {
		return true
	}

	return false
}

// SetGraphSize gets a reference to the given NotebookGraphSize and assigns it to the GraphSize field.
func (o *NotebookTimeseriesCellAttributes) SetGraphSize(v NotebookGraphSize) {
	o.GraphSize = &v
}

// GetSplitBy returns the SplitBy field value if set, zero value otherwise.
func (o *NotebookTimeseriesCellAttributes) GetSplitBy() NotebookSplitBy {
	if o == nil || o.SplitBy == nil {
		var ret NotebookSplitBy
		return ret
	}
	return *o.SplitBy
}

// GetSplitByOk returns a tuple with the SplitBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *NotebookTimeseriesCellAttributes) GetSplitByOk() (*NotebookSplitBy, bool) {
	if o == nil || o.SplitBy == nil {
		return nil, false
	}
	return o.SplitBy, true
}

// HasSplitBy returns a boolean if a field has been set.
func (o *NotebookTimeseriesCellAttributes) HasSplitBy() bool {
	if o != nil && o.SplitBy != nil {
		return true
	}

	return false
}

// SetSplitBy gets a reference to the given NotebookSplitBy and assigns it to the SplitBy field.
func (o *NotebookTimeseriesCellAttributes) SetSplitBy(v NotebookSplitBy) {
	o.SplitBy = &v
}

// GetTime returns the Time field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *NotebookTimeseriesCellAttributes) GetTime() NotebookCellTime {
	if o == nil || o.Time.Get() == nil {
		var ret NotebookCellTime
		return ret
	}
	return *o.Time.Get()
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *NotebookTimeseriesCellAttributes) GetTimeOk() (*NotebookCellTime, bool) {
	if o == nil {
		return nil, false
	}
	return o.Time.Get(), o.Time.IsSet()
}

// HasTime returns a boolean if a field has been set.
func (o *NotebookTimeseriesCellAttributes) HasTime() bool {
	if o != nil && o.Time.IsSet() {
		return true
	}

	return false
}

// SetTime gets a reference to the given NullableNotebookCellTime and assigns it to the Time field.
func (o *NotebookTimeseriesCellAttributes) SetTime(v NotebookCellTime) {
	o.Time.Set(&v)
}

// SetTimeNil sets the value for Time to be an explicit nil.
func (o *NotebookTimeseriesCellAttributes) SetTimeNil() {
	o.Time.Set(nil)
}

// UnsetTime ensures that no value is present for Time, not even an explicit nil.
func (o *NotebookTimeseriesCellAttributes) UnsetTime() {
	o.Time.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookTimeseriesCellAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["definition"] = o.Definition
	if o.GraphSize != nil {
		toSerialize["graph_size"] = o.GraphSize
	}
	if o.SplitBy != nil {
		toSerialize["split_by"] = o.SplitBy
	}
	if o.Time.IsSet() {
		toSerialize["time"] = o.Time.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookTimeseriesCellAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Definition *TimeseriesWidgetDefinition `json:"definition"`
	}{}
	all := struct {
		Definition TimeseriesWidgetDefinition `json:"definition"`
		GraphSize  *NotebookGraphSize         `json:"graph_size,omitempty"`
		SplitBy    *NotebookSplitBy           `json:"split_by,omitempty"`
		Time       NullableNotebookCellTime   `json:"time,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Definition == nil {
		return fmt.Errorf("Required field definition missing")
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
	if v := all.GraphSize; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Definition.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Definition = all.Definition
	o.GraphSize = all.GraphSize
	if all.SplitBy != nil && all.SplitBy.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SplitBy = all.SplitBy
	o.Time = all.Time
	return nil
}

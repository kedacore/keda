// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetEvent Event overlay control options.
//
// See the dedicated [Events JSON schema documentation](https://docs.datadoghq.com/dashboards/graphing_json/widget_json/#events-schema)
// to learn how to build the `<EVENTS_SCHEMA>`.
type WidgetEvent struct {
	// Query definition.
	Q string `json:"q"`
	// The execution method for multi-value filters.
	TagsExecution *string `json:"tags_execution,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetEvent instantiates a new WidgetEvent object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetEvent(q string) *WidgetEvent {
	this := WidgetEvent{}
	this.Q = q
	return &this
}

// NewWidgetEventWithDefaults instantiates a new WidgetEvent object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetEventWithDefaults() *WidgetEvent {
	this := WidgetEvent{}
	return &this
}

// GetQ returns the Q field value.
func (o *WidgetEvent) GetQ() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Q
}

// GetQOk returns a tuple with the Q field value
// and a boolean to check if the value has been set.
func (o *WidgetEvent) GetQOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Q, true
}

// SetQ sets field value.
func (o *WidgetEvent) SetQ(v string) {
	o.Q = v
}

// GetTagsExecution returns the TagsExecution field value if set, zero value otherwise.
func (o *WidgetEvent) GetTagsExecution() string {
	if o == nil || o.TagsExecution == nil {
		var ret string
		return ret
	}
	return *o.TagsExecution
}

// GetTagsExecutionOk returns a tuple with the TagsExecution field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetEvent) GetTagsExecutionOk() (*string, bool) {
	if o == nil || o.TagsExecution == nil {
		return nil, false
	}
	return o.TagsExecution, true
}

// HasTagsExecution returns a boolean if a field has been set.
func (o *WidgetEvent) HasTagsExecution() bool {
	if o != nil && o.TagsExecution != nil {
		return true
	}

	return false
}

// SetTagsExecution gets a reference to the given string and assigns it to the TagsExecution field.
func (o *WidgetEvent) SetTagsExecution(v string) {
	o.TagsExecution = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetEvent) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["q"] = o.Q
	if o.TagsExecution != nil {
		toSerialize["tags_execution"] = o.TagsExecution
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetEvent) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Q *string `json:"q"`
	}{}
	all := struct {
		Q             string  `json:"q"`
		TagsExecution *string `json:"tags_execution,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Q == nil {
		return fmt.Errorf("Required field q missing")
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
	o.Q = all.Q
	o.TagsExecution = all.TagsExecution
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// Log Object describing a log after being processed and stored by Datadog.
type Log struct {
	// JSON object containing all log attributes and their associated values.
	Content *LogContent `json:"content,omitempty"`
	// Unique ID of the Log.
	Id *string `json:"id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLog instantiates a new Log object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLog() *Log {
	this := Log{}
	return &this
}

// NewLogWithDefaults instantiates a new Log object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogWithDefaults() *Log {
	this := Log{}
	return &this
}

// GetContent returns the Content field value if set, zero value otherwise.
func (o *Log) GetContent() LogContent {
	if o == nil || o.Content == nil {
		var ret LogContent
		return ret
	}
	return *o.Content
}

// GetContentOk returns a tuple with the Content field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Log) GetContentOk() (*LogContent, bool) {
	if o == nil || o.Content == nil {
		return nil, false
	}
	return o.Content, true
}

// HasContent returns a boolean if a field has been set.
func (o *Log) HasContent() bool {
	if o != nil && o.Content != nil {
		return true
	}

	return false
}

// SetContent gets a reference to the given LogContent and assigns it to the Content field.
func (o *Log) SetContent(v LogContent) {
	o.Content = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *Log) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Log) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *Log) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *Log) SetId(v string) {
	o.Id = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o Log) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Content != nil {
		toSerialize["content"] = o.Content
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *Log) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Content *LogContent `json:"content,omitempty"`
		Id      *string     `json:"id,omitempty"`
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
	if all.Content != nil && all.Content.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Content = all.Content
	o.Id = all.Id
	return nil
}

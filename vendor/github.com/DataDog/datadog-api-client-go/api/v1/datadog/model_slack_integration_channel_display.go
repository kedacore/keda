// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SlackIntegrationChannelDisplay Configuration options for what is shown in an alert event message.
type SlackIntegrationChannelDisplay struct {
	// Show the main body of the alert event.
	Message *bool `json:"message,omitempty"`
	// Show the list of @-handles in the alert event.
	Notified *bool `json:"notified,omitempty"`
	// Show the alert event's snapshot image.
	Snapshot *bool `json:"snapshot,omitempty"`
	// Show the scopes on which the monitor alerted.
	Tags *bool `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSlackIntegrationChannelDisplay instantiates a new SlackIntegrationChannelDisplay object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSlackIntegrationChannelDisplay() *SlackIntegrationChannelDisplay {
	this := SlackIntegrationChannelDisplay{}
	var message bool = true
	this.Message = &message
	var notified bool = true
	this.Notified = &notified
	var snapshot bool = true
	this.Snapshot = &snapshot
	var tags bool = true
	this.Tags = &tags
	return &this
}

// NewSlackIntegrationChannelDisplayWithDefaults instantiates a new SlackIntegrationChannelDisplay object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSlackIntegrationChannelDisplayWithDefaults() *SlackIntegrationChannelDisplay {
	this := SlackIntegrationChannelDisplay{}
	var message bool = true
	this.Message = &message
	var notified bool = true
	this.Notified = &notified
	var snapshot bool = true
	this.Snapshot = &snapshot
	var tags bool = true
	this.Tags = &tags
	return &this
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SlackIntegrationChannelDisplay) GetMessage() bool {
	if o == nil || o.Message == nil {
		var ret bool
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannelDisplay) GetMessageOk() (*bool, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SlackIntegrationChannelDisplay) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given bool and assigns it to the Message field.
func (o *SlackIntegrationChannelDisplay) SetMessage(v bool) {
	o.Message = &v
}

// GetNotified returns the Notified field value if set, zero value otherwise.
func (o *SlackIntegrationChannelDisplay) GetNotified() bool {
	if o == nil || o.Notified == nil {
		var ret bool
		return ret
	}
	return *o.Notified
}

// GetNotifiedOk returns a tuple with the Notified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannelDisplay) GetNotifiedOk() (*bool, bool) {
	if o == nil || o.Notified == nil {
		return nil, false
	}
	return o.Notified, true
}

// HasNotified returns a boolean if a field has been set.
func (o *SlackIntegrationChannelDisplay) HasNotified() bool {
	if o != nil && o.Notified != nil {
		return true
	}

	return false
}

// SetNotified gets a reference to the given bool and assigns it to the Notified field.
func (o *SlackIntegrationChannelDisplay) SetNotified(v bool) {
	o.Notified = &v
}

// GetSnapshot returns the Snapshot field value if set, zero value otherwise.
func (o *SlackIntegrationChannelDisplay) GetSnapshot() bool {
	if o == nil || o.Snapshot == nil {
		var ret bool
		return ret
	}
	return *o.Snapshot
}

// GetSnapshotOk returns a tuple with the Snapshot field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannelDisplay) GetSnapshotOk() (*bool, bool) {
	if o == nil || o.Snapshot == nil {
		return nil, false
	}
	return o.Snapshot, true
}

// HasSnapshot returns a boolean if a field has been set.
func (o *SlackIntegrationChannelDisplay) HasSnapshot() bool {
	if o != nil && o.Snapshot != nil {
		return true
	}

	return false
}

// SetSnapshot gets a reference to the given bool and assigns it to the Snapshot field.
func (o *SlackIntegrationChannelDisplay) SetSnapshot(v bool) {
	o.Snapshot = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SlackIntegrationChannelDisplay) GetTags() bool {
	if o == nil || o.Tags == nil {
		var ret bool
		return ret
	}
	return *o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannelDisplay) GetTagsOk() (*bool, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SlackIntegrationChannelDisplay) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given bool and assigns it to the Tags field.
func (o *SlackIntegrationChannelDisplay) SetTags(v bool) {
	o.Tags = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SlackIntegrationChannelDisplay) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Notified != nil {
		toSerialize["notified"] = o.Notified
	}
	if o.Snapshot != nil {
		toSerialize["snapshot"] = o.Snapshot
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SlackIntegrationChannelDisplay) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Message  *bool `json:"message,omitempty"`
		Notified *bool `json:"notified,omitempty"`
		Snapshot *bool `json:"snapshot,omitempty"`
		Tags     *bool `json:"tags,omitempty"`
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
	o.Message = all.Message
	o.Notified = all.Notified
	o.Snapshot = all.Snapshot
	o.Tags = all.Tags
	return nil
}

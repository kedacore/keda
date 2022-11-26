// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SlackIntegrationChannel The Slack channel configuration.
type SlackIntegrationChannel struct {
	// Configuration options for what is shown in an alert event message.
	Display *SlackIntegrationChannelDisplay `json:"display,omitempty"`
	// Your channel name.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSlackIntegrationChannel instantiates a new SlackIntegrationChannel object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSlackIntegrationChannel() *SlackIntegrationChannel {
	this := SlackIntegrationChannel{}
	return &this
}

// NewSlackIntegrationChannelWithDefaults instantiates a new SlackIntegrationChannel object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSlackIntegrationChannelWithDefaults() *SlackIntegrationChannel {
	this := SlackIntegrationChannel{}
	return &this
}

// GetDisplay returns the Display field value if set, zero value otherwise.
func (o *SlackIntegrationChannel) GetDisplay() SlackIntegrationChannelDisplay {
	if o == nil || o.Display == nil {
		var ret SlackIntegrationChannelDisplay
		return ret
	}
	return *o.Display
}

// GetDisplayOk returns a tuple with the Display field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannel) GetDisplayOk() (*SlackIntegrationChannelDisplay, bool) {
	if o == nil || o.Display == nil {
		return nil, false
	}
	return o.Display, true
}

// HasDisplay returns a boolean if a field has been set.
func (o *SlackIntegrationChannel) HasDisplay() bool {
	if o != nil && o.Display != nil {
		return true
	}

	return false
}

// SetDisplay gets a reference to the given SlackIntegrationChannelDisplay and assigns it to the Display field.
func (o *SlackIntegrationChannel) SetDisplay(v SlackIntegrationChannelDisplay) {
	o.Display = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SlackIntegrationChannel) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationChannel) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SlackIntegrationChannel) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SlackIntegrationChannel) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SlackIntegrationChannel) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Display != nil {
		toSerialize["display"] = o.Display
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SlackIntegrationChannel) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Display *SlackIntegrationChannelDisplay `json:"display,omitempty"`
		Name    *string                         `json:"name,omitempty"`
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
	if all.Display != nil && all.Display.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Display = all.Display
	o.Name = all.Name
	return nil
}

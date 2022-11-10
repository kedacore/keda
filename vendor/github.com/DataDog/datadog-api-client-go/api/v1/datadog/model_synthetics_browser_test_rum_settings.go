// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserTestRumSettings The RUM data collection settings for the Synthetic browser test.
// **Note:** There are 3 ways to format RUM settings:
//
// `{ isEnabled: false }`
// RUM data is not collected.
//
// `{ isEnabled: true }`
// RUM data is collected from the Synthetic test's default application.
//
// `{ isEnabled: true, applicationId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", clientTokenId: 12345 }`
// RUM data is collected using the specified application.
type SyntheticsBrowserTestRumSettings struct {
	// RUM application ID used to collect RUM data for the browser test.
	ApplicationId *string `json:"applicationId,omitempty"`
	// RUM application API key ID used to collect RUM data for the browser test.
	ClientTokenId *int64 `json:"clientTokenId,omitempty"`
	// Determines whether RUM data is collected during test runs.
	IsEnabled bool `json:"isEnabled"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserTestRumSettings instantiates a new SyntheticsBrowserTestRumSettings object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserTestRumSettings(isEnabled bool) *SyntheticsBrowserTestRumSettings {
	this := SyntheticsBrowserTestRumSettings{}
	this.IsEnabled = isEnabled
	return &this
}

// NewSyntheticsBrowserTestRumSettingsWithDefaults instantiates a new SyntheticsBrowserTestRumSettings object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserTestRumSettingsWithDefaults() *SyntheticsBrowserTestRumSettings {
	this := SyntheticsBrowserTestRumSettings{}
	return &this
}

// GetApplicationId returns the ApplicationId field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestRumSettings) GetApplicationId() string {
	if o == nil || o.ApplicationId == nil {
		var ret string
		return ret
	}
	return *o.ApplicationId
}

// GetApplicationIdOk returns a tuple with the ApplicationId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestRumSettings) GetApplicationIdOk() (*string, bool) {
	if o == nil || o.ApplicationId == nil {
		return nil, false
	}
	return o.ApplicationId, true
}

// HasApplicationId returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestRumSettings) HasApplicationId() bool {
	if o != nil && o.ApplicationId != nil {
		return true
	}

	return false
}

// SetApplicationId gets a reference to the given string and assigns it to the ApplicationId field.
func (o *SyntheticsBrowserTestRumSettings) SetApplicationId(v string) {
	o.ApplicationId = &v
}

// GetClientTokenId returns the ClientTokenId field value if set, zero value otherwise.
func (o *SyntheticsBrowserTestRumSettings) GetClientTokenId() int64 {
	if o == nil || o.ClientTokenId == nil {
		var ret int64
		return ret
	}
	return *o.ClientTokenId
}

// GetClientTokenIdOk returns a tuple with the ClientTokenId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestRumSettings) GetClientTokenIdOk() (*int64, bool) {
	if o == nil || o.ClientTokenId == nil {
		return nil, false
	}
	return o.ClientTokenId, true
}

// HasClientTokenId returns a boolean if a field has been set.
func (o *SyntheticsBrowserTestRumSettings) HasClientTokenId() bool {
	if o != nil && o.ClientTokenId != nil {
		return true
	}

	return false
}

// SetClientTokenId gets a reference to the given int64 and assigns it to the ClientTokenId field.
func (o *SyntheticsBrowserTestRumSettings) SetClientTokenId(v int64) {
	o.ClientTokenId = &v
}

// GetIsEnabled returns the IsEnabled field value.
func (o *SyntheticsBrowserTestRumSettings) GetIsEnabled() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTestRumSettings) GetIsEnabledOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsEnabled, true
}

// SetIsEnabled sets field value.
func (o *SyntheticsBrowserTestRumSettings) SetIsEnabled(v bool) {
	o.IsEnabled = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserTestRumSettings) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApplicationId != nil {
		toSerialize["applicationId"] = o.ApplicationId
	}
	if o.ClientTokenId != nil {
		toSerialize["clientTokenId"] = o.ClientTokenId
	}
	toSerialize["isEnabled"] = o.IsEnabled

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserTestRumSettings) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IsEnabled *bool `json:"isEnabled"`
	}{}
	all := struct {
		ApplicationId *string `json:"applicationId,omitempty"`
		ClientTokenId *int64  `json:"clientTokenId,omitempty"`
		IsEnabled     bool    `json:"isEnabled"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.IsEnabled == nil {
		return fmt.Errorf("Required field isEnabled missing")
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
	o.ApplicationId = all.ApplicationId
	o.ClientTokenId = all.ClientTokenId
	o.IsEnabled = all.IsEnabled
	return nil
}

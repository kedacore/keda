// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// OrganizationSettingsSamlIdpInitiatedLogin Has one property enabled (boolean).
type OrganizationSettingsSamlIdpInitiatedLogin struct {
	// Whether SAML IdP initiated login is enabled, learn more
	// in the [SAML documentation](https://docs.datadoghq.com/account_management/saml/#idp-initiated-login).
	Enabled *bool `json:"enabled,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationSettingsSamlIdpInitiatedLogin instantiates a new OrganizationSettingsSamlIdpInitiatedLogin object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationSettingsSamlIdpInitiatedLogin() *OrganizationSettingsSamlIdpInitiatedLogin {
	this := OrganizationSettingsSamlIdpInitiatedLogin{}
	return &this
}

// NewOrganizationSettingsSamlIdpInitiatedLoginWithDefaults instantiates a new OrganizationSettingsSamlIdpInitiatedLogin object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationSettingsSamlIdpInitiatedLoginWithDefaults() *OrganizationSettingsSamlIdpInitiatedLogin {
	this := OrganizationSettingsSamlIdpInitiatedLogin{}
	return &this
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *OrganizationSettingsSamlIdpInitiatedLogin) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettingsSamlIdpInitiatedLogin) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *OrganizationSettingsSamlIdpInitiatedLogin) HasEnabled() bool {
	if o != nil && o.Enabled != nil {
		return true
	}

	return false
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *OrganizationSettingsSamlIdpInitiatedLogin) SetEnabled(v bool) {
	o.Enabled = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationSettingsSamlIdpInitiatedLogin) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Enabled != nil {
		toSerialize["enabled"] = o.Enabled
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OrganizationSettingsSamlIdpInitiatedLogin) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Enabled *bool `json:"enabled,omitempty"`
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
	o.Enabled = all.Enabled
	return nil
}

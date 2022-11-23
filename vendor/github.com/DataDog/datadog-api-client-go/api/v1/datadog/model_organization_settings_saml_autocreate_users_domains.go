// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// OrganizationSettingsSamlAutocreateUsersDomains Has two properties, `enabled` (boolean) and `domains`, which is a list of domains without the @ symbol.
type OrganizationSettingsSamlAutocreateUsersDomains struct {
	// List of domains where the SAML automated user creation is enabled.
	Domains []string `json:"domains,omitempty"`
	// Whether or not the automated user creation based on SAML domain is enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationSettingsSamlAutocreateUsersDomains instantiates a new OrganizationSettingsSamlAutocreateUsersDomains object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationSettingsSamlAutocreateUsersDomains() *OrganizationSettingsSamlAutocreateUsersDomains {
	this := OrganizationSettingsSamlAutocreateUsersDomains{}
	return &this
}

// NewOrganizationSettingsSamlAutocreateUsersDomainsWithDefaults instantiates a new OrganizationSettingsSamlAutocreateUsersDomains object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationSettingsSamlAutocreateUsersDomainsWithDefaults() *OrganizationSettingsSamlAutocreateUsersDomains {
	this := OrganizationSettingsSamlAutocreateUsersDomains{}
	return &this
}

// GetDomains returns the Domains field value if set, zero value otherwise.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) GetDomains() []string {
	if o == nil || o.Domains == nil {
		var ret []string
		return ret
	}
	return o.Domains
}

// GetDomainsOk returns a tuple with the Domains field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) GetDomainsOk() (*[]string, bool) {
	if o == nil || o.Domains == nil {
		return nil, false
	}
	return &o.Domains, true
}

// HasDomains returns a boolean if a field has been set.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) HasDomains() bool {
	if o != nil && o.Domains != nil {
		return true
	}

	return false
}

// SetDomains gets a reference to the given []string and assigns it to the Domains field.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) SetDomains(v []string) {
	o.Domains = v
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) HasEnabled() bool {
	if o != nil && o.Enabled != nil {
		return true
	}

	return false
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *OrganizationSettingsSamlAutocreateUsersDomains) SetEnabled(v bool) {
	o.Enabled = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationSettingsSamlAutocreateUsersDomains) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Domains != nil {
		toSerialize["domains"] = o.Domains
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
func (o *OrganizationSettingsSamlAutocreateUsersDomains) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Domains []string `json:"domains,omitempty"`
		Enabled *bool    `json:"enabled,omitempty"`
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
	o.Domains = all.Domains
	o.Enabled = all.Enabled
	return nil
}

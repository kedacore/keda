// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// OrganizationSettings A JSON array of settings.
type OrganizationSettings struct {
	// Whether or not the organization users can share widgets outside of Datadog.
	PrivateWidgetShare *bool `json:"private_widget_share,omitempty"`
	// Set the boolean property enabled to enable or disable single sign on with SAML.
	// See the SAML documentation for more information about all SAML settings.
	Saml *OrganizationSettingsSaml `json:"saml,omitempty"`
	// The access role of the user. Options are **st** (standard user), **adm** (admin user), or **ro** (read-only user).
	SamlAutocreateAccessRole *AccessRole `json:"saml_autocreate_access_role,omitempty"`
	// Has two properties, `enabled` (boolean) and `domains`, which is a list of domains without the @ symbol.
	SamlAutocreateUsersDomains *OrganizationSettingsSamlAutocreateUsersDomains `json:"saml_autocreate_users_domains,omitempty"`
	// Whether or not SAML can be enabled for this organization.
	SamlCanBeEnabled *bool `json:"saml_can_be_enabled,omitempty"`
	// Identity provider endpoint for SAML authentication.
	SamlIdpEndpoint *string `json:"saml_idp_endpoint,omitempty"`
	// Has one property enabled (boolean).
	SamlIdpInitiatedLogin *OrganizationSettingsSamlIdpInitiatedLogin `json:"saml_idp_initiated_login,omitempty"`
	// Whether or not a SAML identity provider metadata file was provided to the Datadog organization.
	SamlIdpMetadataUploaded *bool `json:"saml_idp_metadata_uploaded,omitempty"`
	// URL for SAML logging.
	SamlLoginUrl *string `json:"saml_login_url,omitempty"`
	// Has one property enabled (boolean).
	SamlStrictMode *OrganizationSettingsSamlStrictMode `json:"saml_strict_mode,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationSettings instantiates a new OrganizationSettings object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationSettings() *OrganizationSettings {
	this := OrganizationSettings{}
	var samlAutocreateAccessRole AccessRole = ACCESSROLE_STANDARD
	this.SamlAutocreateAccessRole = &samlAutocreateAccessRole
	return &this
}

// NewOrganizationSettingsWithDefaults instantiates a new OrganizationSettings object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationSettingsWithDefaults() *OrganizationSettings {
	this := OrganizationSettings{}
	var samlAutocreateAccessRole AccessRole = ACCESSROLE_STANDARD
	this.SamlAutocreateAccessRole = &samlAutocreateAccessRole
	return &this
}

// GetPrivateWidgetShare returns the PrivateWidgetShare field value if set, zero value otherwise.
func (o *OrganizationSettings) GetPrivateWidgetShare() bool {
	if o == nil || o.PrivateWidgetShare == nil {
		var ret bool
		return ret
	}
	return *o.PrivateWidgetShare
}

// GetPrivateWidgetShareOk returns a tuple with the PrivateWidgetShare field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetPrivateWidgetShareOk() (*bool, bool) {
	if o == nil || o.PrivateWidgetShare == nil {
		return nil, false
	}
	return o.PrivateWidgetShare, true
}

// HasPrivateWidgetShare returns a boolean if a field has been set.
func (o *OrganizationSettings) HasPrivateWidgetShare() bool {
	if o != nil && o.PrivateWidgetShare != nil {
		return true
	}

	return false
}

// SetPrivateWidgetShare gets a reference to the given bool and assigns it to the PrivateWidgetShare field.
func (o *OrganizationSettings) SetPrivateWidgetShare(v bool) {
	o.PrivateWidgetShare = &v
}

// GetSaml returns the Saml field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSaml() OrganizationSettingsSaml {
	if o == nil || o.Saml == nil {
		var ret OrganizationSettingsSaml
		return ret
	}
	return *o.Saml
}

// GetSamlOk returns a tuple with the Saml field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlOk() (*OrganizationSettingsSaml, bool) {
	if o == nil || o.Saml == nil {
		return nil, false
	}
	return o.Saml, true
}

// HasSaml returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSaml() bool {
	if o != nil && o.Saml != nil {
		return true
	}

	return false
}

// SetSaml gets a reference to the given OrganizationSettingsSaml and assigns it to the Saml field.
func (o *OrganizationSettings) SetSaml(v OrganizationSettingsSaml) {
	o.Saml = &v
}

// GetSamlAutocreateAccessRole returns the SamlAutocreateAccessRole field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlAutocreateAccessRole() AccessRole {
	if o == nil || o.SamlAutocreateAccessRole == nil {
		var ret AccessRole
		return ret
	}
	return *o.SamlAutocreateAccessRole
}

// GetSamlAutocreateAccessRoleOk returns a tuple with the SamlAutocreateAccessRole field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlAutocreateAccessRoleOk() (*AccessRole, bool) {
	if o == nil || o.SamlAutocreateAccessRole == nil {
		return nil, false
	}
	return o.SamlAutocreateAccessRole, true
}

// HasSamlAutocreateAccessRole returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlAutocreateAccessRole() bool {
	if o != nil && o.SamlAutocreateAccessRole != nil {
		return true
	}

	return false
}

// SetSamlAutocreateAccessRole gets a reference to the given AccessRole and assigns it to the SamlAutocreateAccessRole field.
func (o *OrganizationSettings) SetSamlAutocreateAccessRole(v AccessRole) {
	o.SamlAutocreateAccessRole = &v
}

// GetSamlAutocreateUsersDomains returns the SamlAutocreateUsersDomains field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlAutocreateUsersDomains() OrganizationSettingsSamlAutocreateUsersDomains {
	if o == nil || o.SamlAutocreateUsersDomains == nil {
		var ret OrganizationSettingsSamlAutocreateUsersDomains
		return ret
	}
	return *o.SamlAutocreateUsersDomains
}

// GetSamlAutocreateUsersDomainsOk returns a tuple with the SamlAutocreateUsersDomains field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlAutocreateUsersDomainsOk() (*OrganizationSettingsSamlAutocreateUsersDomains, bool) {
	if o == nil || o.SamlAutocreateUsersDomains == nil {
		return nil, false
	}
	return o.SamlAutocreateUsersDomains, true
}

// HasSamlAutocreateUsersDomains returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlAutocreateUsersDomains() bool {
	if o != nil && o.SamlAutocreateUsersDomains != nil {
		return true
	}

	return false
}

// SetSamlAutocreateUsersDomains gets a reference to the given OrganizationSettingsSamlAutocreateUsersDomains and assigns it to the SamlAutocreateUsersDomains field.
func (o *OrganizationSettings) SetSamlAutocreateUsersDomains(v OrganizationSettingsSamlAutocreateUsersDomains) {
	o.SamlAutocreateUsersDomains = &v
}

// GetSamlCanBeEnabled returns the SamlCanBeEnabled field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlCanBeEnabled() bool {
	if o == nil || o.SamlCanBeEnabled == nil {
		var ret bool
		return ret
	}
	return *o.SamlCanBeEnabled
}

// GetSamlCanBeEnabledOk returns a tuple with the SamlCanBeEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlCanBeEnabledOk() (*bool, bool) {
	if o == nil || o.SamlCanBeEnabled == nil {
		return nil, false
	}
	return o.SamlCanBeEnabled, true
}

// HasSamlCanBeEnabled returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlCanBeEnabled() bool {
	if o != nil && o.SamlCanBeEnabled != nil {
		return true
	}

	return false
}

// SetSamlCanBeEnabled gets a reference to the given bool and assigns it to the SamlCanBeEnabled field.
func (o *OrganizationSettings) SetSamlCanBeEnabled(v bool) {
	o.SamlCanBeEnabled = &v
}

// GetSamlIdpEndpoint returns the SamlIdpEndpoint field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlIdpEndpoint() string {
	if o == nil || o.SamlIdpEndpoint == nil {
		var ret string
		return ret
	}
	return *o.SamlIdpEndpoint
}

// GetSamlIdpEndpointOk returns a tuple with the SamlIdpEndpoint field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlIdpEndpointOk() (*string, bool) {
	if o == nil || o.SamlIdpEndpoint == nil {
		return nil, false
	}
	return o.SamlIdpEndpoint, true
}

// HasSamlIdpEndpoint returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlIdpEndpoint() bool {
	if o != nil && o.SamlIdpEndpoint != nil {
		return true
	}

	return false
}

// SetSamlIdpEndpoint gets a reference to the given string and assigns it to the SamlIdpEndpoint field.
func (o *OrganizationSettings) SetSamlIdpEndpoint(v string) {
	o.SamlIdpEndpoint = &v
}

// GetSamlIdpInitiatedLogin returns the SamlIdpInitiatedLogin field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlIdpInitiatedLogin() OrganizationSettingsSamlIdpInitiatedLogin {
	if o == nil || o.SamlIdpInitiatedLogin == nil {
		var ret OrganizationSettingsSamlIdpInitiatedLogin
		return ret
	}
	return *o.SamlIdpInitiatedLogin
}

// GetSamlIdpInitiatedLoginOk returns a tuple with the SamlIdpInitiatedLogin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlIdpInitiatedLoginOk() (*OrganizationSettingsSamlIdpInitiatedLogin, bool) {
	if o == nil || o.SamlIdpInitiatedLogin == nil {
		return nil, false
	}
	return o.SamlIdpInitiatedLogin, true
}

// HasSamlIdpInitiatedLogin returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlIdpInitiatedLogin() bool {
	if o != nil && o.SamlIdpInitiatedLogin != nil {
		return true
	}

	return false
}

// SetSamlIdpInitiatedLogin gets a reference to the given OrganizationSettingsSamlIdpInitiatedLogin and assigns it to the SamlIdpInitiatedLogin field.
func (o *OrganizationSettings) SetSamlIdpInitiatedLogin(v OrganizationSettingsSamlIdpInitiatedLogin) {
	o.SamlIdpInitiatedLogin = &v
}

// GetSamlIdpMetadataUploaded returns the SamlIdpMetadataUploaded field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlIdpMetadataUploaded() bool {
	if o == nil || o.SamlIdpMetadataUploaded == nil {
		var ret bool
		return ret
	}
	return *o.SamlIdpMetadataUploaded
}

// GetSamlIdpMetadataUploadedOk returns a tuple with the SamlIdpMetadataUploaded field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlIdpMetadataUploadedOk() (*bool, bool) {
	if o == nil || o.SamlIdpMetadataUploaded == nil {
		return nil, false
	}
	return o.SamlIdpMetadataUploaded, true
}

// HasSamlIdpMetadataUploaded returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlIdpMetadataUploaded() bool {
	if o != nil && o.SamlIdpMetadataUploaded != nil {
		return true
	}

	return false
}

// SetSamlIdpMetadataUploaded gets a reference to the given bool and assigns it to the SamlIdpMetadataUploaded field.
func (o *OrganizationSettings) SetSamlIdpMetadataUploaded(v bool) {
	o.SamlIdpMetadataUploaded = &v
}

// GetSamlLoginUrl returns the SamlLoginUrl field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlLoginUrl() string {
	if o == nil || o.SamlLoginUrl == nil {
		var ret string
		return ret
	}
	return *o.SamlLoginUrl
}

// GetSamlLoginUrlOk returns a tuple with the SamlLoginUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlLoginUrlOk() (*string, bool) {
	if o == nil || o.SamlLoginUrl == nil {
		return nil, false
	}
	return o.SamlLoginUrl, true
}

// HasSamlLoginUrl returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlLoginUrl() bool {
	if o != nil && o.SamlLoginUrl != nil {
		return true
	}

	return false
}

// SetSamlLoginUrl gets a reference to the given string and assigns it to the SamlLoginUrl field.
func (o *OrganizationSettings) SetSamlLoginUrl(v string) {
	o.SamlLoginUrl = &v
}

// GetSamlStrictMode returns the SamlStrictMode field value if set, zero value otherwise.
func (o *OrganizationSettings) GetSamlStrictMode() OrganizationSettingsSamlStrictMode {
	if o == nil || o.SamlStrictMode == nil {
		var ret OrganizationSettingsSamlStrictMode
		return ret
	}
	return *o.SamlStrictMode
}

// GetSamlStrictModeOk returns a tuple with the SamlStrictMode field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationSettings) GetSamlStrictModeOk() (*OrganizationSettingsSamlStrictMode, bool) {
	if o == nil || o.SamlStrictMode == nil {
		return nil, false
	}
	return o.SamlStrictMode, true
}

// HasSamlStrictMode returns a boolean if a field has been set.
func (o *OrganizationSettings) HasSamlStrictMode() bool {
	if o != nil && o.SamlStrictMode != nil {
		return true
	}

	return false
}

// SetSamlStrictMode gets a reference to the given OrganizationSettingsSamlStrictMode and assigns it to the SamlStrictMode field.
func (o *OrganizationSettings) SetSamlStrictMode(v OrganizationSettingsSamlStrictMode) {
	o.SamlStrictMode = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationSettings) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.PrivateWidgetShare != nil {
		toSerialize["private_widget_share"] = o.PrivateWidgetShare
	}
	if o.Saml != nil {
		toSerialize["saml"] = o.Saml
	}
	if o.SamlAutocreateAccessRole != nil {
		toSerialize["saml_autocreate_access_role"] = o.SamlAutocreateAccessRole
	}
	if o.SamlAutocreateUsersDomains != nil {
		toSerialize["saml_autocreate_users_domains"] = o.SamlAutocreateUsersDomains
	}
	if o.SamlCanBeEnabled != nil {
		toSerialize["saml_can_be_enabled"] = o.SamlCanBeEnabled
	}
	if o.SamlIdpEndpoint != nil {
		toSerialize["saml_idp_endpoint"] = o.SamlIdpEndpoint
	}
	if o.SamlIdpInitiatedLogin != nil {
		toSerialize["saml_idp_initiated_login"] = o.SamlIdpInitiatedLogin
	}
	if o.SamlIdpMetadataUploaded != nil {
		toSerialize["saml_idp_metadata_uploaded"] = o.SamlIdpMetadataUploaded
	}
	if o.SamlLoginUrl != nil {
		toSerialize["saml_login_url"] = o.SamlLoginUrl
	}
	if o.SamlStrictMode != nil {
		toSerialize["saml_strict_mode"] = o.SamlStrictMode
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OrganizationSettings) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		PrivateWidgetShare         *bool                                           `json:"private_widget_share,omitempty"`
		Saml                       *OrganizationSettingsSaml                       `json:"saml,omitempty"`
		SamlAutocreateAccessRole   *AccessRole                                     `json:"saml_autocreate_access_role,omitempty"`
		SamlAutocreateUsersDomains *OrganizationSettingsSamlAutocreateUsersDomains `json:"saml_autocreate_users_domains,omitempty"`
		SamlCanBeEnabled           *bool                                           `json:"saml_can_be_enabled,omitempty"`
		SamlIdpEndpoint            *string                                         `json:"saml_idp_endpoint,omitempty"`
		SamlIdpInitiatedLogin      *OrganizationSettingsSamlIdpInitiatedLogin      `json:"saml_idp_initiated_login,omitempty"`
		SamlIdpMetadataUploaded    *bool                                           `json:"saml_idp_metadata_uploaded,omitempty"`
		SamlLoginUrl               *string                                         `json:"saml_login_url,omitempty"`
		SamlStrictMode             *OrganizationSettingsSamlStrictMode             `json:"saml_strict_mode,omitempty"`
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
	if v := all.SamlAutocreateAccessRole; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.PrivateWidgetShare = all.PrivateWidgetShare
	if all.Saml != nil && all.Saml.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Saml = all.Saml
	o.SamlAutocreateAccessRole = all.SamlAutocreateAccessRole
	if all.SamlAutocreateUsersDomains != nil && all.SamlAutocreateUsersDomains.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SamlAutocreateUsersDomains = all.SamlAutocreateUsersDomains
	o.SamlCanBeEnabled = all.SamlCanBeEnabled
	o.SamlIdpEndpoint = all.SamlIdpEndpoint
	if all.SamlIdpInitiatedLogin != nil && all.SamlIdpInitiatedLogin.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SamlIdpInitiatedLogin = all.SamlIdpInitiatedLogin
	o.SamlIdpMetadataUploaded = all.SamlIdpMetadataUploaded
	o.SamlLoginUrl = all.SamlLoginUrl
	if all.SamlStrictMode != nil && all.SamlStrictMode.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SamlStrictMode = all.SamlStrictMode
	return nil
}

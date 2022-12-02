// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// OrganizationCreateResponse Response object for an organization creation.
type OrganizationCreateResponse struct {
	// Datadog API key.
	ApiKey *ApiKey `json:"api_key,omitempty"`
	// An application key with its associated metadata.
	ApplicationKey *ApplicationKey `json:"application_key,omitempty"`
	// Create, edit, and manage organizations.
	Org *Organization `json:"org,omitempty"`
	// Create, edit, and disable users.
	User *User `json:"user,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationCreateResponse instantiates a new OrganizationCreateResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationCreateResponse() *OrganizationCreateResponse {
	this := OrganizationCreateResponse{}
	return &this
}

// NewOrganizationCreateResponseWithDefaults instantiates a new OrganizationCreateResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationCreateResponseWithDefaults() *OrganizationCreateResponse {
	this := OrganizationCreateResponse{}
	return &this
}

// GetApiKey returns the ApiKey field value if set, zero value otherwise.
func (o *OrganizationCreateResponse) GetApiKey() ApiKey {
	if o == nil || o.ApiKey == nil {
		var ret ApiKey
		return ret
	}
	return *o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationCreateResponse) GetApiKeyOk() (*ApiKey, bool) {
	if o == nil || o.ApiKey == nil {
		return nil, false
	}
	return o.ApiKey, true
}

// HasApiKey returns a boolean if a field has been set.
func (o *OrganizationCreateResponse) HasApiKey() bool {
	if o != nil && o.ApiKey != nil {
		return true
	}

	return false
}

// SetApiKey gets a reference to the given ApiKey and assigns it to the ApiKey field.
func (o *OrganizationCreateResponse) SetApiKey(v ApiKey) {
	o.ApiKey = &v
}

// GetApplicationKey returns the ApplicationKey field value if set, zero value otherwise.
func (o *OrganizationCreateResponse) GetApplicationKey() ApplicationKey {
	if o == nil || o.ApplicationKey == nil {
		var ret ApplicationKey
		return ret
	}
	return *o.ApplicationKey
}

// GetApplicationKeyOk returns a tuple with the ApplicationKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationCreateResponse) GetApplicationKeyOk() (*ApplicationKey, bool) {
	if o == nil || o.ApplicationKey == nil {
		return nil, false
	}
	return o.ApplicationKey, true
}

// HasApplicationKey returns a boolean if a field has been set.
func (o *OrganizationCreateResponse) HasApplicationKey() bool {
	if o != nil && o.ApplicationKey != nil {
		return true
	}

	return false
}

// SetApplicationKey gets a reference to the given ApplicationKey and assigns it to the ApplicationKey field.
func (o *OrganizationCreateResponse) SetApplicationKey(v ApplicationKey) {
	o.ApplicationKey = &v
}

// GetOrg returns the Org field value if set, zero value otherwise.
func (o *OrganizationCreateResponse) GetOrg() Organization {
	if o == nil || o.Org == nil {
		var ret Organization
		return ret
	}
	return *o.Org
}

// GetOrgOk returns a tuple with the Org field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationCreateResponse) GetOrgOk() (*Organization, bool) {
	if o == nil || o.Org == nil {
		return nil, false
	}
	return o.Org, true
}

// HasOrg returns a boolean if a field has been set.
func (o *OrganizationCreateResponse) HasOrg() bool {
	if o != nil && o.Org != nil {
		return true
	}

	return false
}

// SetOrg gets a reference to the given Organization and assigns it to the Org field.
func (o *OrganizationCreateResponse) SetOrg(v Organization) {
	o.Org = &v
}

// GetUser returns the User field value if set, zero value otherwise.
func (o *OrganizationCreateResponse) GetUser() User {
	if o == nil || o.User == nil {
		var ret User
		return ret
	}
	return *o.User
}

// GetUserOk returns a tuple with the User field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationCreateResponse) GetUserOk() (*User, bool) {
	if o == nil || o.User == nil {
		return nil, false
	}
	return o.User, true
}

// HasUser returns a boolean if a field has been set.
func (o *OrganizationCreateResponse) HasUser() bool {
	if o != nil && o.User != nil {
		return true
	}

	return false
}

// SetUser gets a reference to the given User and assigns it to the User field.
func (o *OrganizationCreateResponse) SetUser(v User) {
	o.User = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationCreateResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApiKey != nil {
		toSerialize["api_key"] = o.ApiKey
	}
	if o.ApplicationKey != nil {
		toSerialize["application_key"] = o.ApplicationKey
	}
	if o.Org != nil {
		toSerialize["org"] = o.Org
	}
	if o.User != nil {
		toSerialize["user"] = o.User
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OrganizationCreateResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApiKey         *ApiKey         `json:"api_key,omitempty"`
		ApplicationKey *ApplicationKey `json:"application_key,omitempty"`
		Org            *Organization   `json:"org,omitempty"`
		User           *User           `json:"user,omitempty"`
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
	if all.ApiKey != nil && all.ApiKey.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApiKey = all.ApiKey
	if all.ApplicationKey != nil && all.ApplicationKey.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApplicationKey = all.ApplicationKey
	if all.Org != nil && all.Org.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Org = all.Org
	if all.User != nil && all.User.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.User = all.User
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthWeb Object to handle basic authentication when performing the test.
type SyntheticsBasicAuthWeb struct {
	// Password to use for the basic authentication.
	Password string `json:"password"`
	// The type of basic authentication to use when performing the test.
	Type *SyntheticsBasicAuthWebType `json:"type,omitempty"`
	// Username to use for the basic authentication.
	Username string `json:"username"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthWeb instantiates a new SyntheticsBasicAuthWeb object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthWeb(password string, username string) *SyntheticsBasicAuthWeb {
	this := SyntheticsBasicAuthWeb{}
	this.Password = password
	var typeVar SyntheticsBasicAuthWebType = SYNTHETICSBASICAUTHWEBTYPE_WEB
	this.Type = &typeVar
	this.Username = username
	return &this
}

// NewSyntheticsBasicAuthWebWithDefaults instantiates a new SyntheticsBasicAuthWeb object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthWebWithDefaults() *SyntheticsBasicAuthWeb {
	this := SyntheticsBasicAuthWeb{}
	var typeVar SyntheticsBasicAuthWebType = SYNTHETICSBASICAUTHWEBTYPE_WEB
	this.Type = &typeVar
	return &this
}

// GetPassword returns the Password field value.
func (o *SyntheticsBasicAuthWeb) GetPassword() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Password
}

// GetPasswordOk returns a tuple with the Password field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthWeb) GetPasswordOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Password, true
}

// SetPassword sets field value.
func (o *SyntheticsBasicAuthWeb) SetPassword(v string) {
	o.Password = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthWeb) GetType() SyntheticsBasicAuthWebType {
	if o == nil || o.Type == nil {
		var ret SyntheticsBasicAuthWebType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthWeb) GetTypeOk() (*SyntheticsBasicAuthWebType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthWeb) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given SyntheticsBasicAuthWebType and assigns it to the Type field.
func (o *SyntheticsBasicAuthWeb) SetType(v SyntheticsBasicAuthWebType) {
	o.Type = &v
}

// GetUsername returns the Username field value.
func (o *SyntheticsBasicAuthWeb) GetUsername() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Username
}

// GetUsernameOk returns a tuple with the Username field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthWeb) GetUsernameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Username, true
}

// SetUsername sets field value.
func (o *SyntheticsBasicAuthWeb) SetUsername(v string) {
	o.Username = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthWeb) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["password"] = o.Password
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	toSerialize["username"] = o.Username

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBasicAuthWeb) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Password *string `json:"password"`
		Username *string `json:"username"`
	}{}
	all := struct {
		Password string                      `json:"password"`
		Type     *SyntheticsBasicAuthWebType `json:"type,omitempty"`
		Username string                      `json:"username"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Password == nil {
		return fmt.Errorf("Required field password missing")
	}
	if required.Username == nil {
		return fmt.Errorf("Required field username missing")
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Password = all.Password
	o.Type = all.Type
	o.Username = all.Username
	return nil
}

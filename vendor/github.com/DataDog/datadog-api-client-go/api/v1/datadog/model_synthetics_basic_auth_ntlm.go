// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthNTLM Object to handle `NTLM` authentication when performing the test.
type SyntheticsBasicAuthNTLM struct {
	// Domain for the authentication to use when performing the test.
	Domain *string `json:"domain,omitempty"`
	// Password for the authentication to use when performing the test.
	Password *string `json:"password,omitempty"`
	// The type of authentication to use when performing the test.
	Type SyntheticsBasicAuthNTLMType `json:"type"`
	// Username for the authentication to use when performing the test.
	Username *string `json:"username,omitempty"`
	// Workstation for the authentication to use when performing the test.
	Workstation *string `json:"workstation,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthNTLM instantiates a new SyntheticsBasicAuthNTLM object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthNTLM(typeVar SyntheticsBasicAuthNTLMType) *SyntheticsBasicAuthNTLM {
	this := SyntheticsBasicAuthNTLM{}
	this.Type = typeVar
	return &this
}

// NewSyntheticsBasicAuthNTLMWithDefaults instantiates a new SyntheticsBasicAuthNTLM object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthNTLMWithDefaults() *SyntheticsBasicAuthNTLM {
	this := SyntheticsBasicAuthNTLM{}
	var typeVar SyntheticsBasicAuthNTLMType = SYNTHETICSBASICAUTHNTLMTYPE_NTLM
	this.Type = typeVar
	return &this
}

// GetDomain returns the Domain field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthNTLM) GetDomain() string {
	if o == nil || o.Domain == nil {
		var ret string
		return ret
	}
	return *o.Domain
}

// GetDomainOk returns a tuple with the Domain field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthNTLM) GetDomainOk() (*string, bool) {
	if o == nil || o.Domain == nil {
		return nil, false
	}
	return o.Domain, true
}

// HasDomain returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthNTLM) HasDomain() bool {
	if o != nil && o.Domain != nil {
		return true
	}

	return false
}

// SetDomain gets a reference to the given string and assigns it to the Domain field.
func (o *SyntheticsBasicAuthNTLM) SetDomain(v string) {
	o.Domain = &v
}

// GetPassword returns the Password field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthNTLM) GetPassword() string {
	if o == nil || o.Password == nil {
		var ret string
		return ret
	}
	return *o.Password
}

// GetPasswordOk returns a tuple with the Password field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthNTLM) GetPasswordOk() (*string, bool) {
	if o == nil || o.Password == nil {
		return nil, false
	}
	return o.Password, true
}

// HasPassword returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthNTLM) HasPassword() bool {
	if o != nil && o.Password != nil {
		return true
	}

	return false
}

// SetPassword gets a reference to the given string and assigns it to the Password field.
func (o *SyntheticsBasicAuthNTLM) SetPassword(v string) {
	o.Password = &v
}

// GetType returns the Type field value.
func (o *SyntheticsBasicAuthNTLM) GetType() SyntheticsBasicAuthNTLMType {
	if o == nil {
		var ret SyntheticsBasicAuthNTLMType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthNTLM) GetTypeOk() (*SyntheticsBasicAuthNTLMType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsBasicAuthNTLM) SetType(v SyntheticsBasicAuthNTLMType) {
	o.Type = v
}

// GetUsername returns the Username field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthNTLM) GetUsername() string {
	if o == nil || o.Username == nil {
		var ret string
		return ret
	}
	return *o.Username
}

// GetUsernameOk returns a tuple with the Username field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthNTLM) GetUsernameOk() (*string, bool) {
	if o == nil || o.Username == nil {
		return nil, false
	}
	return o.Username, true
}

// HasUsername returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthNTLM) HasUsername() bool {
	if o != nil && o.Username != nil {
		return true
	}

	return false
}

// SetUsername gets a reference to the given string and assigns it to the Username field.
func (o *SyntheticsBasicAuthNTLM) SetUsername(v string) {
	o.Username = &v
}

// GetWorkstation returns the Workstation field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthNTLM) GetWorkstation() string {
	if o == nil || o.Workstation == nil {
		var ret string
		return ret
	}
	return *o.Workstation
}

// GetWorkstationOk returns a tuple with the Workstation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthNTLM) GetWorkstationOk() (*string, bool) {
	if o == nil || o.Workstation == nil {
		return nil, false
	}
	return o.Workstation, true
}

// HasWorkstation returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthNTLM) HasWorkstation() bool {
	if o != nil && o.Workstation != nil {
		return true
	}

	return false
}

// SetWorkstation gets a reference to the given string and assigns it to the Workstation field.
func (o *SyntheticsBasicAuthNTLM) SetWorkstation(v string) {
	o.Workstation = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthNTLM) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Domain != nil {
		toSerialize["domain"] = o.Domain
	}
	if o.Password != nil {
		toSerialize["password"] = o.Password
	}
	toSerialize["type"] = o.Type
	if o.Username != nil {
		toSerialize["username"] = o.Username
	}
	if o.Workstation != nil {
		toSerialize["workstation"] = o.Workstation
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBasicAuthNTLM) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *SyntheticsBasicAuthNTLMType `json:"type"`
	}{}
	all := struct {
		Domain      *string                     `json:"domain,omitempty"`
		Password    *string                     `json:"password,omitempty"`
		Type        SyntheticsBasicAuthNTLMType `json:"type"`
		Username    *string                     `json:"username,omitempty"`
		Workstation *string                     `json:"workstation,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Domain = all.Domain
	o.Password = all.Password
	o.Type = all.Type
	o.Username = all.Username
	o.Workstation = all.Workstation
	return nil
}

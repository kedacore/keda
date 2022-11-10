// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthSigv4 Object to handle `SIGV4` authentication when performing the test.
type SyntheticsBasicAuthSigv4 struct {
	// Access key for the `SIGV4` authentication.
	AccessKey string `json:"accessKey"`
	// Region for the `SIGV4` authentication.
	Region *string `json:"region,omitempty"`
	// Secret key for the `SIGV4` authentication.
	SecretKey string `json:"secretKey"`
	// Service name for the `SIGV4` authentication.
	ServiceName *string `json:"serviceName,omitempty"`
	// Session token for the `SIGV4` authentication.
	SessionToken *string `json:"sessionToken,omitempty"`
	// The type of authentication to use when performing the test.
	Type SyntheticsBasicAuthSigv4Type `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthSigv4 instantiates a new SyntheticsBasicAuthSigv4 object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthSigv4(accessKey string, secretKey string, typeVar SyntheticsBasicAuthSigv4Type) *SyntheticsBasicAuthSigv4 {
	this := SyntheticsBasicAuthSigv4{}
	this.AccessKey = accessKey
	this.SecretKey = secretKey
	this.Type = typeVar
	return &this
}

// NewSyntheticsBasicAuthSigv4WithDefaults instantiates a new SyntheticsBasicAuthSigv4 object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthSigv4WithDefaults() *SyntheticsBasicAuthSigv4 {
	this := SyntheticsBasicAuthSigv4{}
	var typeVar SyntheticsBasicAuthSigv4Type = SYNTHETICSBASICAUTHSIGV4TYPE_SIGV4
	this.Type = typeVar
	return &this
}

// GetAccessKey returns the AccessKey field value.
func (o *SyntheticsBasicAuthSigv4) GetAccessKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccessKey
}

// GetAccessKeyOk returns a tuple with the AccessKey field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetAccessKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccessKey, true
}

// SetAccessKey sets field value.
func (o *SyntheticsBasicAuthSigv4) SetAccessKey(v string) {
	o.AccessKey = v
}

// GetRegion returns the Region field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthSigv4) GetRegion() string {
	if o == nil || o.Region == nil {
		var ret string
		return ret
	}
	return *o.Region
}

// GetRegionOk returns a tuple with the Region field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetRegionOk() (*string, bool) {
	if o == nil || o.Region == nil {
		return nil, false
	}
	return o.Region, true
}

// HasRegion returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthSigv4) HasRegion() bool {
	if o != nil && o.Region != nil {
		return true
	}

	return false
}

// SetRegion gets a reference to the given string and assigns it to the Region field.
func (o *SyntheticsBasicAuthSigv4) SetRegion(v string) {
	o.Region = &v
}

// GetSecretKey returns the SecretKey field value.
func (o *SyntheticsBasicAuthSigv4) GetSecretKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.SecretKey
}

// GetSecretKeyOk returns a tuple with the SecretKey field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetSecretKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SecretKey, true
}

// SetSecretKey sets field value.
func (o *SyntheticsBasicAuthSigv4) SetSecretKey(v string) {
	o.SecretKey = v
}

// GetServiceName returns the ServiceName field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthSigv4) GetServiceName() string {
	if o == nil || o.ServiceName == nil {
		var ret string
		return ret
	}
	return *o.ServiceName
}

// GetServiceNameOk returns a tuple with the ServiceName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetServiceNameOk() (*string, bool) {
	if o == nil || o.ServiceName == nil {
		return nil, false
	}
	return o.ServiceName, true
}

// HasServiceName returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthSigv4) HasServiceName() bool {
	if o != nil && o.ServiceName != nil {
		return true
	}

	return false
}

// SetServiceName gets a reference to the given string and assigns it to the ServiceName field.
func (o *SyntheticsBasicAuthSigv4) SetServiceName(v string) {
	o.ServiceName = &v
}

// GetSessionToken returns the SessionToken field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthSigv4) GetSessionToken() string {
	if o == nil || o.SessionToken == nil {
		var ret string
		return ret
	}
	return *o.SessionToken
}

// GetSessionTokenOk returns a tuple with the SessionToken field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetSessionTokenOk() (*string, bool) {
	if o == nil || o.SessionToken == nil {
		return nil, false
	}
	return o.SessionToken, true
}

// HasSessionToken returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthSigv4) HasSessionToken() bool {
	if o != nil && o.SessionToken != nil {
		return true
	}

	return false
}

// SetSessionToken gets a reference to the given string and assigns it to the SessionToken field.
func (o *SyntheticsBasicAuthSigv4) SetSessionToken(v string) {
	o.SessionToken = &v
}

// GetType returns the Type field value.
func (o *SyntheticsBasicAuthSigv4) GetType() SyntheticsBasicAuthSigv4Type {
	if o == nil {
		var ret SyntheticsBasicAuthSigv4Type
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthSigv4) GetTypeOk() (*SyntheticsBasicAuthSigv4Type, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsBasicAuthSigv4) SetType(v SyntheticsBasicAuthSigv4Type) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthSigv4) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["accessKey"] = o.AccessKey
	if o.Region != nil {
		toSerialize["region"] = o.Region
	}
	toSerialize["secretKey"] = o.SecretKey
	if o.ServiceName != nil {
		toSerialize["serviceName"] = o.ServiceName
	}
	if o.SessionToken != nil {
		toSerialize["sessionToken"] = o.SessionToken
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBasicAuthSigv4) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccessKey *string                       `json:"accessKey"`
		SecretKey *string                       `json:"secretKey"`
		Type      *SyntheticsBasicAuthSigv4Type `json:"type"`
	}{}
	all := struct {
		AccessKey    string                       `json:"accessKey"`
		Region       *string                      `json:"region,omitempty"`
		SecretKey    string                       `json:"secretKey"`
		ServiceName  *string                      `json:"serviceName,omitempty"`
		SessionToken *string                      `json:"sessionToken,omitempty"`
		Type         SyntheticsBasicAuthSigv4Type `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccessKey == nil {
		return fmt.Errorf("Required field accessKey missing")
	}
	if required.SecretKey == nil {
		return fmt.Errorf("Required field secretKey missing")
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
	o.AccessKey = all.AccessKey
	o.Region = all.Region
	o.SecretKey = all.SecretKey
	o.ServiceName = all.ServiceName
	o.SessionToken = all.SessionToken
	o.Type = all.Type
	return nil
}

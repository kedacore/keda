// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"os"
)

// IdpFormData Object describing the IdP configuration.
type IdpFormData struct {
	// The path to the XML metadata file you wish to upload.
	IdpFile *os.File `json:"idp_file"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewIdpFormData instantiates a new IdpFormData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIdpFormData(idpFile *os.File) *IdpFormData {
	this := IdpFormData{}
	this.IdpFile = idpFile
	return &this
}

// NewIdpFormDataWithDefaults instantiates a new IdpFormData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIdpFormDataWithDefaults() *IdpFormData {
	this := IdpFormData{}
	return &this
}

// GetIdpFile returns the IdpFile field value.
func (o *IdpFormData) GetIdpFile() *os.File {
	if o == nil {
		var ret *os.File
		return ret
	}
	return o.IdpFile
}

// GetIdpFileOk returns a tuple with the IdpFile field value
// and a boolean to check if the value has been set.
func (o *IdpFormData) GetIdpFileOk() (**os.File, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IdpFile, true
}

// SetIdpFile sets field value.
func (o *IdpFormData) SetIdpFile(v *os.File) {
	o.IdpFile = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IdpFormData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["idp_file"] = o.IdpFile

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IdpFormData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IdpFile **os.File `json:"idp_file"`
	}{}
	all := struct {
		IdpFile *os.File `json:"idp_file"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.IdpFile == nil {
		return fmt.Errorf("Required field idp_file missing")
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
	o.IdpFile = all.IdpFile
	return nil
}

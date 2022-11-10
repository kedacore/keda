// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsSSLCertificateSubject Object describing the SSL certificate used for the test.
type SyntheticsSSLCertificateSubject struct {
	// Country Name associated with the certificate.
	C *string `json:"C,omitempty"`
	// Common Name that associated with the certificate.
	Cn *string `json:"CN,omitempty"`
	// Locality associated with the certificate.
	L *string `json:"L,omitempty"`
	// Organization associated with the certificate.
	O *string `json:"O,omitempty"`
	// Organizational Unit associated with the certificate.
	Ou *string `json:"OU,omitempty"`
	// State Or Province Name associated with the certificate.
	St *string `json:"ST,omitempty"`
	// Subject Alternative Name associated with the certificate.
	AltName *string `json:"altName,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsSSLCertificateSubject instantiates a new SyntheticsSSLCertificateSubject object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsSSLCertificateSubject() *SyntheticsSSLCertificateSubject {
	this := SyntheticsSSLCertificateSubject{}
	return &this
}

// NewSyntheticsSSLCertificateSubjectWithDefaults instantiates a new SyntheticsSSLCertificateSubject object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsSSLCertificateSubjectWithDefaults() *SyntheticsSSLCertificateSubject {
	this := SyntheticsSSLCertificateSubject{}
	return &this
}

// GetC returns the C field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetC() string {
	if o == nil || o.C == nil {
		var ret string
		return ret
	}
	return *o.C
}

// GetCOk returns a tuple with the C field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetCOk() (*string, bool) {
	if o == nil || o.C == nil {
		return nil, false
	}
	return o.C, true
}

// HasC returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasC() bool {
	if o != nil && o.C != nil {
		return true
	}

	return false
}

// SetC gets a reference to the given string and assigns it to the C field.
func (o *SyntheticsSSLCertificateSubject) SetC(v string) {
	o.C = &v
}

// GetCn returns the Cn field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetCn() string {
	if o == nil || o.Cn == nil {
		var ret string
		return ret
	}
	return *o.Cn
}

// GetCnOk returns a tuple with the Cn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetCnOk() (*string, bool) {
	if o == nil || o.Cn == nil {
		return nil, false
	}
	return o.Cn, true
}

// HasCn returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasCn() bool {
	if o != nil && o.Cn != nil {
		return true
	}

	return false
}

// SetCn gets a reference to the given string and assigns it to the Cn field.
func (o *SyntheticsSSLCertificateSubject) SetCn(v string) {
	o.Cn = &v
}

// GetL returns the L field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetL() string {
	if o == nil || o.L == nil {
		var ret string
		return ret
	}
	return *o.L
}

// GetLOk returns a tuple with the L field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetLOk() (*string, bool) {
	if o == nil || o.L == nil {
		return nil, false
	}
	return o.L, true
}

// HasL returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasL() bool {
	if o != nil && o.L != nil {
		return true
	}

	return false
}

// SetL gets a reference to the given string and assigns it to the L field.
func (o *SyntheticsSSLCertificateSubject) SetL(v string) {
	o.L = &v
}

// GetO returns the O field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetO() string {
	if o == nil || o.O == nil {
		var ret string
		return ret
	}
	return *o.O
}

// GetOOk returns a tuple with the O field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetOOk() (*string, bool) {
	if o == nil || o.O == nil {
		return nil, false
	}
	return o.O, true
}

// HasO returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasO() bool {
	if o != nil && o.O != nil {
		return true
	}

	return false
}

// SetO gets a reference to the given string and assigns it to the O field.
func (o *SyntheticsSSLCertificateSubject) SetO(v string) {
	o.O = &v
}

// GetOu returns the Ou field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetOu() string {
	if o == nil || o.Ou == nil {
		var ret string
		return ret
	}
	return *o.Ou
}

// GetOuOk returns a tuple with the Ou field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetOuOk() (*string, bool) {
	if o == nil || o.Ou == nil {
		return nil, false
	}
	return o.Ou, true
}

// HasOu returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasOu() bool {
	if o != nil && o.Ou != nil {
		return true
	}

	return false
}

// SetOu gets a reference to the given string and assigns it to the Ou field.
func (o *SyntheticsSSLCertificateSubject) SetOu(v string) {
	o.Ou = &v
}

// GetSt returns the St field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetSt() string {
	if o == nil || o.St == nil {
		var ret string
		return ret
	}
	return *o.St
}

// GetStOk returns a tuple with the St field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetStOk() (*string, bool) {
	if o == nil || o.St == nil {
		return nil, false
	}
	return o.St, true
}

// HasSt returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasSt() bool {
	if o != nil && o.St != nil {
		return true
	}

	return false
}

// SetSt gets a reference to the given string and assigns it to the St field.
func (o *SyntheticsSSLCertificateSubject) SetSt(v string) {
	o.St = &v
}

// GetAltName returns the AltName field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateSubject) GetAltName() string {
	if o == nil || o.AltName == nil {
		var ret string
		return ret
	}
	return *o.AltName
}

// GetAltNameOk returns a tuple with the AltName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateSubject) GetAltNameOk() (*string, bool) {
	if o == nil || o.AltName == nil {
		return nil, false
	}
	return o.AltName, true
}

// HasAltName returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateSubject) HasAltName() bool {
	if o != nil && o.AltName != nil {
		return true
	}

	return false
}

// SetAltName gets a reference to the given string and assigns it to the AltName field.
func (o *SyntheticsSSLCertificateSubject) SetAltName(v string) {
	o.AltName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsSSLCertificateSubject) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.C != nil {
		toSerialize["C"] = o.C
	}
	if o.Cn != nil {
		toSerialize["CN"] = o.Cn
	}
	if o.L != nil {
		toSerialize["L"] = o.L
	}
	if o.O != nil {
		toSerialize["O"] = o.O
	}
	if o.Ou != nil {
		toSerialize["OU"] = o.Ou
	}
	if o.St != nil {
		toSerialize["ST"] = o.St
	}
	if o.AltName != nil {
		toSerialize["altName"] = o.AltName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsSSLCertificateSubject) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		C       *string `json:"C,omitempty"`
		Cn      *string `json:"CN,omitempty"`
		L       *string `json:"L,omitempty"`
		O       *string `json:"O,omitempty"`
		Ou      *string `json:"OU,omitempty"`
		St      *string `json:"ST,omitempty"`
		AltName *string `json:"altName,omitempty"`
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
	o.C = all.C
	o.Cn = all.Cn
	o.L = all.L
	o.O = all.O
	o.Ou = all.Ou
	o.St = all.St
	o.AltName = all.AltName
	return nil
}

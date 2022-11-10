// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsSSLCertificateIssuer Object describing the issuer of a SSL certificate.
type SyntheticsSSLCertificateIssuer struct {
	// Country Name that issued the certificate.
	C *string `json:"C,omitempty"`
	// Common Name that issued certificate.
	Cn *string `json:"CN,omitempty"`
	// Locality that issued the certificate.
	L *string `json:"L,omitempty"`
	// Organization that issued the certificate.
	O *string `json:"O,omitempty"`
	// Organizational Unit that issued the certificate.
	Ou *string `json:"OU,omitempty"`
	// State Or Province Name that issued the certificate.
	St *string `json:"ST,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsSSLCertificateIssuer instantiates a new SyntheticsSSLCertificateIssuer object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsSSLCertificateIssuer() *SyntheticsSSLCertificateIssuer {
	this := SyntheticsSSLCertificateIssuer{}
	return &this
}

// NewSyntheticsSSLCertificateIssuerWithDefaults instantiates a new SyntheticsSSLCertificateIssuer object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsSSLCertificateIssuerWithDefaults() *SyntheticsSSLCertificateIssuer {
	this := SyntheticsSSLCertificateIssuer{}
	return &this
}

// GetC returns the C field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetC() string {
	if o == nil || o.C == nil {
		var ret string
		return ret
	}
	return *o.C
}

// GetCOk returns a tuple with the C field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetCOk() (*string, bool) {
	if o == nil || o.C == nil {
		return nil, false
	}
	return o.C, true
}

// HasC returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasC() bool {
	if o != nil && o.C != nil {
		return true
	}

	return false
}

// SetC gets a reference to the given string and assigns it to the C field.
func (o *SyntheticsSSLCertificateIssuer) SetC(v string) {
	o.C = &v
}

// GetCn returns the Cn field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetCn() string {
	if o == nil || o.Cn == nil {
		var ret string
		return ret
	}
	return *o.Cn
}

// GetCnOk returns a tuple with the Cn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetCnOk() (*string, bool) {
	if o == nil || o.Cn == nil {
		return nil, false
	}
	return o.Cn, true
}

// HasCn returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasCn() bool {
	if o != nil && o.Cn != nil {
		return true
	}

	return false
}

// SetCn gets a reference to the given string and assigns it to the Cn field.
func (o *SyntheticsSSLCertificateIssuer) SetCn(v string) {
	o.Cn = &v
}

// GetL returns the L field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetL() string {
	if o == nil || o.L == nil {
		var ret string
		return ret
	}
	return *o.L
}

// GetLOk returns a tuple with the L field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetLOk() (*string, bool) {
	if o == nil || o.L == nil {
		return nil, false
	}
	return o.L, true
}

// HasL returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasL() bool {
	if o != nil && o.L != nil {
		return true
	}

	return false
}

// SetL gets a reference to the given string and assigns it to the L field.
func (o *SyntheticsSSLCertificateIssuer) SetL(v string) {
	o.L = &v
}

// GetO returns the O field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetO() string {
	if o == nil || o.O == nil {
		var ret string
		return ret
	}
	return *o.O
}

// GetOOk returns a tuple with the O field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetOOk() (*string, bool) {
	if o == nil || o.O == nil {
		return nil, false
	}
	return o.O, true
}

// HasO returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasO() bool {
	if o != nil && o.O != nil {
		return true
	}

	return false
}

// SetO gets a reference to the given string and assigns it to the O field.
func (o *SyntheticsSSLCertificateIssuer) SetO(v string) {
	o.O = &v
}

// GetOu returns the Ou field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetOu() string {
	if o == nil || o.Ou == nil {
		var ret string
		return ret
	}
	return *o.Ou
}

// GetOuOk returns a tuple with the Ou field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetOuOk() (*string, bool) {
	if o == nil || o.Ou == nil {
		return nil, false
	}
	return o.Ou, true
}

// HasOu returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasOu() bool {
	if o != nil && o.Ou != nil {
		return true
	}

	return false
}

// SetOu gets a reference to the given string and assigns it to the Ou field.
func (o *SyntheticsSSLCertificateIssuer) SetOu(v string) {
	o.Ou = &v
}

// GetSt returns the St field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificateIssuer) GetSt() string {
	if o == nil || o.St == nil {
		var ret string
		return ret
	}
	return *o.St
}

// GetStOk returns a tuple with the St field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificateIssuer) GetStOk() (*string, bool) {
	if o == nil || o.St == nil {
		return nil, false
	}
	return o.St, true
}

// HasSt returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificateIssuer) HasSt() bool {
	if o != nil && o.St != nil {
		return true
	}

	return false
}

// SetSt gets a reference to the given string and assigns it to the St field.
func (o *SyntheticsSSLCertificateIssuer) SetSt(v string) {
	o.St = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsSSLCertificateIssuer) MarshalJSON() ([]byte, error) {
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

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsSSLCertificateIssuer) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		C  *string `json:"C,omitempty"`
		Cn *string `json:"CN,omitempty"`
		L  *string `json:"L,omitempty"`
		O  *string `json:"O,omitempty"`
		Ou *string `json:"OU,omitempty"`
		St *string `json:"ST,omitempty"`
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
	return nil
}

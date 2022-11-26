// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// SyntheticsSSLCertificate Object describing the SSL certificate used for a Synthetic test.
type SyntheticsSSLCertificate struct {
	// Cipher used for the connection.
	Cipher *string `json:"cipher,omitempty"`
	// Exponent associated to the certificate.
	Exponent *float64 `json:"exponent,omitempty"`
	// Array of extensions and details used for the certificate.
	ExtKeyUsage []string `json:"extKeyUsage,omitempty"`
	// MD5 digest of the DER-encoded Certificate information.
	Fingerprint *string `json:"fingerprint,omitempty"`
	// SHA-1 digest of the DER-encoded Certificate information.
	Fingerprint256 *string `json:"fingerprint256,omitempty"`
	// Object describing the issuer of a SSL certificate.
	Issuer *SyntheticsSSLCertificateIssuer `json:"issuer,omitempty"`
	// Modulus associated to the SSL certificate private key.
	Modulus *string `json:"modulus,omitempty"`
	// TLS protocol used for the test.
	Protocol *string `json:"protocol,omitempty"`
	// Serial Number assigned by Symantec to the SSL certificate.
	SerialNumber *string `json:"serialNumber,omitempty"`
	// Object describing the SSL certificate used for the test.
	Subject *SyntheticsSSLCertificateSubject `json:"subject,omitempty"`
	// Date from which the SSL certificate is valid.
	ValidFrom *time.Time `json:"validFrom,omitempty"`
	// Date until which the SSL certificate is valid.
	ValidTo *time.Time `json:"validTo,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsSSLCertificate instantiates a new SyntheticsSSLCertificate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsSSLCertificate() *SyntheticsSSLCertificate {
	this := SyntheticsSSLCertificate{}
	return &this
}

// NewSyntheticsSSLCertificateWithDefaults instantiates a new SyntheticsSSLCertificate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsSSLCertificateWithDefaults() *SyntheticsSSLCertificate {
	this := SyntheticsSSLCertificate{}
	return &this
}

// GetCipher returns the Cipher field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetCipher() string {
	if o == nil || o.Cipher == nil {
		var ret string
		return ret
	}
	return *o.Cipher
}

// GetCipherOk returns a tuple with the Cipher field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetCipherOk() (*string, bool) {
	if o == nil || o.Cipher == nil {
		return nil, false
	}
	return o.Cipher, true
}

// HasCipher returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasCipher() bool {
	if o != nil && o.Cipher != nil {
		return true
	}

	return false
}

// SetCipher gets a reference to the given string and assigns it to the Cipher field.
func (o *SyntheticsSSLCertificate) SetCipher(v string) {
	o.Cipher = &v
}

// GetExponent returns the Exponent field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetExponent() float64 {
	if o == nil || o.Exponent == nil {
		var ret float64
		return ret
	}
	return *o.Exponent
}

// GetExponentOk returns a tuple with the Exponent field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetExponentOk() (*float64, bool) {
	if o == nil || o.Exponent == nil {
		return nil, false
	}
	return o.Exponent, true
}

// HasExponent returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasExponent() bool {
	if o != nil && o.Exponent != nil {
		return true
	}

	return false
}

// SetExponent gets a reference to the given float64 and assigns it to the Exponent field.
func (o *SyntheticsSSLCertificate) SetExponent(v float64) {
	o.Exponent = &v
}

// GetExtKeyUsage returns the ExtKeyUsage field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetExtKeyUsage() []string {
	if o == nil || o.ExtKeyUsage == nil {
		var ret []string
		return ret
	}
	return o.ExtKeyUsage
}

// GetExtKeyUsageOk returns a tuple with the ExtKeyUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetExtKeyUsageOk() (*[]string, bool) {
	if o == nil || o.ExtKeyUsage == nil {
		return nil, false
	}
	return &o.ExtKeyUsage, true
}

// HasExtKeyUsage returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasExtKeyUsage() bool {
	if o != nil && o.ExtKeyUsage != nil {
		return true
	}

	return false
}

// SetExtKeyUsage gets a reference to the given []string and assigns it to the ExtKeyUsage field.
func (o *SyntheticsSSLCertificate) SetExtKeyUsage(v []string) {
	o.ExtKeyUsage = v
}

// GetFingerprint returns the Fingerprint field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetFingerprint() string {
	if o == nil || o.Fingerprint == nil {
		var ret string
		return ret
	}
	return *o.Fingerprint
}

// GetFingerprintOk returns a tuple with the Fingerprint field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetFingerprintOk() (*string, bool) {
	if o == nil || o.Fingerprint == nil {
		return nil, false
	}
	return o.Fingerprint, true
}

// HasFingerprint returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasFingerprint() bool {
	if o != nil && o.Fingerprint != nil {
		return true
	}

	return false
}

// SetFingerprint gets a reference to the given string and assigns it to the Fingerprint field.
func (o *SyntheticsSSLCertificate) SetFingerprint(v string) {
	o.Fingerprint = &v
}

// GetFingerprint256 returns the Fingerprint256 field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetFingerprint256() string {
	if o == nil || o.Fingerprint256 == nil {
		var ret string
		return ret
	}
	return *o.Fingerprint256
}

// GetFingerprint256Ok returns a tuple with the Fingerprint256 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetFingerprint256Ok() (*string, bool) {
	if o == nil || o.Fingerprint256 == nil {
		return nil, false
	}
	return o.Fingerprint256, true
}

// HasFingerprint256 returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasFingerprint256() bool {
	if o != nil && o.Fingerprint256 != nil {
		return true
	}

	return false
}

// SetFingerprint256 gets a reference to the given string and assigns it to the Fingerprint256 field.
func (o *SyntheticsSSLCertificate) SetFingerprint256(v string) {
	o.Fingerprint256 = &v
}

// GetIssuer returns the Issuer field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetIssuer() SyntheticsSSLCertificateIssuer {
	if o == nil || o.Issuer == nil {
		var ret SyntheticsSSLCertificateIssuer
		return ret
	}
	return *o.Issuer
}

// GetIssuerOk returns a tuple with the Issuer field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetIssuerOk() (*SyntheticsSSLCertificateIssuer, bool) {
	if o == nil || o.Issuer == nil {
		return nil, false
	}
	return o.Issuer, true
}

// HasIssuer returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasIssuer() bool {
	if o != nil && o.Issuer != nil {
		return true
	}

	return false
}

// SetIssuer gets a reference to the given SyntheticsSSLCertificateIssuer and assigns it to the Issuer field.
func (o *SyntheticsSSLCertificate) SetIssuer(v SyntheticsSSLCertificateIssuer) {
	o.Issuer = &v
}

// GetModulus returns the Modulus field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetModulus() string {
	if o == nil || o.Modulus == nil {
		var ret string
		return ret
	}
	return *o.Modulus
}

// GetModulusOk returns a tuple with the Modulus field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetModulusOk() (*string, bool) {
	if o == nil || o.Modulus == nil {
		return nil, false
	}
	return o.Modulus, true
}

// HasModulus returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasModulus() bool {
	if o != nil && o.Modulus != nil {
		return true
	}

	return false
}

// SetModulus gets a reference to the given string and assigns it to the Modulus field.
func (o *SyntheticsSSLCertificate) SetModulus(v string) {
	o.Modulus = &v
}

// GetProtocol returns the Protocol field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetProtocol() string {
	if o == nil || o.Protocol == nil {
		var ret string
		return ret
	}
	return *o.Protocol
}

// GetProtocolOk returns a tuple with the Protocol field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetProtocolOk() (*string, bool) {
	if o == nil || o.Protocol == nil {
		return nil, false
	}
	return o.Protocol, true
}

// HasProtocol returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasProtocol() bool {
	if o != nil && o.Protocol != nil {
		return true
	}

	return false
}

// SetProtocol gets a reference to the given string and assigns it to the Protocol field.
func (o *SyntheticsSSLCertificate) SetProtocol(v string) {
	o.Protocol = &v
}

// GetSerialNumber returns the SerialNumber field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetSerialNumber() string {
	if o == nil || o.SerialNumber == nil {
		var ret string
		return ret
	}
	return *o.SerialNumber
}

// GetSerialNumberOk returns a tuple with the SerialNumber field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetSerialNumberOk() (*string, bool) {
	if o == nil || o.SerialNumber == nil {
		return nil, false
	}
	return o.SerialNumber, true
}

// HasSerialNumber returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasSerialNumber() bool {
	if o != nil && o.SerialNumber != nil {
		return true
	}

	return false
}

// SetSerialNumber gets a reference to the given string and assigns it to the SerialNumber field.
func (o *SyntheticsSSLCertificate) SetSerialNumber(v string) {
	o.SerialNumber = &v
}

// GetSubject returns the Subject field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetSubject() SyntheticsSSLCertificateSubject {
	if o == nil || o.Subject == nil {
		var ret SyntheticsSSLCertificateSubject
		return ret
	}
	return *o.Subject
}

// GetSubjectOk returns a tuple with the Subject field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetSubjectOk() (*SyntheticsSSLCertificateSubject, bool) {
	if o == nil || o.Subject == nil {
		return nil, false
	}
	return o.Subject, true
}

// HasSubject returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasSubject() bool {
	if o != nil && o.Subject != nil {
		return true
	}

	return false
}

// SetSubject gets a reference to the given SyntheticsSSLCertificateSubject and assigns it to the Subject field.
func (o *SyntheticsSSLCertificate) SetSubject(v SyntheticsSSLCertificateSubject) {
	o.Subject = &v
}

// GetValidFrom returns the ValidFrom field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetValidFrom() time.Time {
	if o == nil || o.ValidFrom == nil {
		var ret time.Time
		return ret
	}
	return *o.ValidFrom
}

// GetValidFromOk returns a tuple with the ValidFrom field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetValidFromOk() (*time.Time, bool) {
	if o == nil || o.ValidFrom == nil {
		return nil, false
	}
	return o.ValidFrom, true
}

// HasValidFrom returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasValidFrom() bool {
	if o != nil && o.ValidFrom != nil {
		return true
	}

	return false
}

// SetValidFrom gets a reference to the given time.Time and assigns it to the ValidFrom field.
func (o *SyntheticsSSLCertificate) SetValidFrom(v time.Time) {
	o.ValidFrom = &v
}

// GetValidTo returns the ValidTo field value if set, zero value otherwise.
func (o *SyntheticsSSLCertificate) GetValidTo() time.Time {
	if o == nil || o.ValidTo == nil {
		var ret time.Time
		return ret
	}
	return *o.ValidTo
}

// GetValidToOk returns a tuple with the ValidTo field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsSSLCertificate) GetValidToOk() (*time.Time, bool) {
	if o == nil || o.ValidTo == nil {
		return nil, false
	}
	return o.ValidTo, true
}

// HasValidTo returns a boolean if a field has been set.
func (o *SyntheticsSSLCertificate) HasValidTo() bool {
	if o != nil && o.ValidTo != nil {
		return true
	}

	return false
}

// SetValidTo gets a reference to the given time.Time and assigns it to the ValidTo field.
func (o *SyntheticsSSLCertificate) SetValidTo(v time.Time) {
	o.ValidTo = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsSSLCertificate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cipher != nil {
		toSerialize["cipher"] = o.Cipher
	}
	if o.Exponent != nil {
		toSerialize["exponent"] = o.Exponent
	}
	if o.ExtKeyUsage != nil {
		toSerialize["extKeyUsage"] = o.ExtKeyUsage
	}
	if o.Fingerprint != nil {
		toSerialize["fingerprint"] = o.Fingerprint
	}
	if o.Fingerprint256 != nil {
		toSerialize["fingerprint256"] = o.Fingerprint256
	}
	if o.Issuer != nil {
		toSerialize["issuer"] = o.Issuer
	}
	if o.Modulus != nil {
		toSerialize["modulus"] = o.Modulus
	}
	if o.Protocol != nil {
		toSerialize["protocol"] = o.Protocol
	}
	if o.SerialNumber != nil {
		toSerialize["serialNumber"] = o.SerialNumber
	}
	if o.Subject != nil {
		toSerialize["subject"] = o.Subject
	}
	if o.ValidFrom != nil {
		if o.ValidFrom.Nanosecond() == 0 {
			toSerialize["validFrom"] = o.ValidFrom.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["validFrom"] = o.ValidFrom.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.ValidTo != nil {
		if o.ValidTo.Nanosecond() == 0 {
			toSerialize["validTo"] = o.ValidTo.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["validTo"] = o.ValidTo.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsSSLCertificate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cipher         *string                          `json:"cipher,omitempty"`
		Exponent       *float64                         `json:"exponent,omitempty"`
		ExtKeyUsage    []string                         `json:"extKeyUsage,omitempty"`
		Fingerprint    *string                          `json:"fingerprint,omitempty"`
		Fingerprint256 *string                          `json:"fingerprint256,omitempty"`
		Issuer         *SyntheticsSSLCertificateIssuer  `json:"issuer,omitempty"`
		Modulus        *string                          `json:"modulus,omitempty"`
		Protocol       *string                          `json:"protocol,omitempty"`
		SerialNumber   *string                          `json:"serialNumber,omitempty"`
		Subject        *SyntheticsSSLCertificateSubject `json:"subject,omitempty"`
		ValidFrom      *time.Time                       `json:"validFrom,omitempty"`
		ValidTo        *time.Time                       `json:"validTo,omitempty"`
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
	o.Cipher = all.Cipher
	o.Exponent = all.Exponent
	o.ExtKeyUsage = all.ExtKeyUsage
	o.Fingerprint = all.Fingerprint
	o.Fingerprint256 = all.Fingerprint256
	if all.Issuer != nil && all.Issuer.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Issuer = all.Issuer
	o.Modulus = all.Modulus
	o.Protocol = all.Protocol
	o.SerialNumber = all.SerialNumber
	if all.Subject != nil && all.Subject.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Subject = all.Subject
	o.ValidFrom = all.ValidFrom
	o.ValidTo = all.ValidTo
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTiming Object containing all metrics and their values collected for a Synthetic API test.
// Learn more about those metrics in [Synthetics documentation](https://docs.datadoghq.com/synthetics/#metrics).
type SyntheticsTiming struct {
	// The duration in millisecond of the DNS lookup.
	Dns *float64 `json:"dns,omitempty"`
	// The time in millisecond to download the response.
	Download *float64 `json:"download,omitempty"`
	// The time in millisecond to first byte.
	FirstByte *float64 `json:"firstByte,omitempty"`
	// The duration in millisecond of the TLS handshake.
	Handshake *float64 `json:"handshake,omitempty"`
	// The time in millisecond spent during redirections.
	Redirect *float64 `json:"redirect,omitempty"`
	// The duration in millisecond of the TLS handshake.
	Ssl *float64 `json:"ssl,omitempty"`
	// Time in millisecond to establish the TCP connection.
	Tcp *float64 `json:"tcp,omitempty"`
	// The overall time in millisecond the request took to be processed.
	Total *float64 `json:"total,omitempty"`
	// Time spent in millisecond waiting for a response.
	Wait *float64 `json:"wait,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTiming instantiates a new SyntheticsTiming object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTiming() *SyntheticsTiming {
	this := SyntheticsTiming{}
	return &this
}

// NewSyntheticsTimingWithDefaults instantiates a new SyntheticsTiming object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTimingWithDefaults() *SyntheticsTiming {
	this := SyntheticsTiming{}
	return &this
}

// GetDns returns the Dns field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetDns() float64 {
	if o == nil || o.Dns == nil {
		var ret float64
		return ret
	}
	return *o.Dns
}

// GetDnsOk returns a tuple with the Dns field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetDnsOk() (*float64, bool) {
	if o == nil || o.Dns == nil {
		return nil, false
	}
	return o.Dns, true
}

// HasDns returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasDns() bool {
	if o != nil && o.Dns != nil {
		return true
	}

	return false
}

// SetDns gets a reference to the given float64 and assigns it to the Dns field.
func (o *SyntheticsTiming) SetDns(v float64) {
	o.Dns = &v
}

// GetDownload returns the Download field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetDownload() float64 {
	if o == nil || o.Download == nil {
		var ret float64
		return ret
	}
	return *o.Download
}

// GetDownloadOk returns a tuple with the Download field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetDownloadOk() (*float64, bool) {
	if o == nil || o.Download == nil {
		return nil, false
	}
	return o.Download, true
}

// HasDownload returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasDownload() bool {
	if o != nil && o.Download != nil {
		return true
	}

	return false
}

// SetDownload gets a reference to the given float64 and assigns it to the Download field.
func (o *SyntheticsTiming) SetDownload(v float64) {
	o.Download = &v
}

// GetFirstByte returns the FirstByte field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetFirstByte() float64 {
	if o == nil || o.FirstByte == nil {
		var ret float64
		return ret
	}
	return *o.FirstByte
}

// GetFirstByteOk returns a tuple with the FirstByte field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetFirstByteOk() (*float64, bool) {
	if o == nil || o.FirstByte == nil {
		return nil, false
	}
	return o.FirstByte, true
}

// HasFirstByte returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasFirstByte() bool {
	if o != nil && o.FirstByte != nil {
		return true
	}

	return false
}

// SetFirstByte gets a reference to the given float64 and assigns it to the FirstByte field.
func (o *SyntheticsTiming) SetFirstByte(v float64) {
	o.FirstByte = &v
}

// GetHandshake returns the Handshake field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetHandshake() float64 {
	if o == nil || o.Handshake == nil {
		var ret float64
		return ret
	}
	return *o.Handshake
}

// GetHandshakeOk returns a tuple with the Handshake field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetHandshakeOk() (*float64, bool) {
	if o == nil || o.Handshake == nil {
		return nil, false
	}
	return o.Handshake, true
}

// HasHandshake returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasHandshake() bool {
	if o != nil && o.Handshake != nil {
		return true
	}

	return false
}

// SetHandshake gets a reference to the given float64 and assigns it to the Handshake field.
func (o *SyntheticsTiming) SetHandshake(v float64) {
	o.Handshake = &v
}

// GetRedirect returns the Redirect field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetRedirect() float64 {
	if o == nil || o.Redirect == nil {
		var ret float64
		return ret
	}
	return *o.Redirect
}

// GetRedirectOk returns a tuple with the Redirect field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetRedirectOk() (*float64, bool) {
	if o == nil || o.Redirect == nil {
		return nil, false
	}
	return o.Redirect, true
}

// HasRedirect returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasRedirect() bool {
	if o != nil && o.Redirect != nil {
		return true
	}

	return false
}

// SetRedirect gets a reference to the given float64 and assigns it to the Redirect field.
func (o *SyntheticsTiming) SetRedirect(v float64) {
	o.Redirect = &v
}

// GetSsl returns the Ssl field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetSsl() float64 {
	if o == nil || o.Ssl == nil {
		var ret float64
		return ret
	}
	return *o.Ssl
}

// GetSslOk returns a tuple with the Ssl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetSslOk() (*float64, bool) {
	if o == nil || o.Ssl == nil {
		return nil, false
	}
	return o.Ssl, true
}

// HasSsl returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasSsl() bool {
	if o != nil && o.Ssl != nil {
		return true
	}

	return false
}

// SetSsl gets a reference to the given float64 and assigns it to the Ssl field.
func (o *SyntheticsTiming) SetSsl(v float64) {
	o.Ssl = &v
}

// GetTcp returns the Tcp field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetTcp() float64 {
	if o == nil || o.Tcp == nil {
		var ret float64
		return ret
	}
	return *o.Tcp
}

// GetTcpOk returns a tuple with the Tcp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetTcpOk() (*float64, bool) {
	if o == nil || o.Tcp == nil {
		return nil, false
	}
	return o.Tcp, true
}

// HasTcp returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasTcp() bool {
	if o != nil && o.Tcp != nil {
		return true
	}

	return false
}

// SetTcp gets a reference to the given float64 and assigns it to the Tcp field.
func (o *SyntheticsTiming) SetTcp(v float64) {
	o.Tcp = &v
}

// GetTotal returns the Total field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetTotal() float64 {
	if o == nil || o.Total == nil {
		var ret float64
		return ret
	}
	return *o.Total
}

// GetTotalOk returns a tuple with the Total field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetTotalOk() (*float64, bool) {
	if o == nil || o.Total == nil {
		return nil, false
	}
	return o.Total, true
}

// HasTotal returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasTotal() bool {
	if o != nil && o.Total != nil {
		return true
	}

	return false
}

// SetTotal gets a reference to the given float64 and assigns it to the Total field.
func (o *SyntheticsTiming) SetTotal(v float64) {
	o.Total = &v
}

// GetWait returns the Wait field value if set, zero value otherwise.
func (o *SyntheticsTiming) GetWait() float64 {
	if o == nil || o.Wait == nil {
		var ret float64
		return ret
	}
	return *o.Wait
}

// GetWaitOk returns a tuple with the Wait field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTiming) GetWaitOk() (*float64, bool) {
	if o == nil || o.Wait == nil {
		return nil, false
	}
	return o.Wait, true
}

// HasWait returns a boolean if a field has been set.
func (o *SyntheticsTiming) HasWait() bool {
	if o != nil && o.Wait != nil {
		return true
	}

	return false
}

// SetWait gets a reference to the given float64 and assigns it to the Wait field.
func (o *SyntheticsTiming) SetWait(v float64) {
	o.Wait = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTiming) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Dns != nil {
		toSerialize["dns"] = o.Dns
	}
	if o.Download != nil {
		toSerialize["download"] = o.Download
	}
	if o.FirstByte != nil {
		toSerialize["firstByte"] = o.FirstByte
	}
	if o.Handshake != nil {
		toSerialize["handshake"] = o.Handshake
	}
	if o.Redirect != nil {
		toSerialize["redirect"] = o.Redirect
	}
	if o.Ssl != nil {
		toSerialize["ssl"] = o.Ssl
	}
	if o.Tcp != nil {
		toSerialize["tcp"] = o.Tcp
	}
	if o.Total != nil {
		toSerialize["total"] = o.Total
	}
	if o.Wait != nil {
		toSerialize["wait"] = o.Wait
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTiming) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Dns       *float64 `json:"dns,omitempty"`
		Download  *float64 `json:"download,omitempty"`
		FirstByte *float64 `json:"firstByte,omitempty"`
		Handshake *float64 `json:"handshake,omitempty"`
		Redirect  *float64 `json:"redirect,omitempty"`
		Ssl       *float64 `json:"ssl,omitempty"`
		Tcp       *float64 `json:"tcp,omitempty"`
		Total     *float64 `json:"total,omitempty"`
		Wait      *float64 `json:"wait,omitempty"`
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
	o.Dns = all.Dns
	o.Download = all.Download
	o.FirstByte = all.FirstByte
	o.Handshake = all.Handshake
	o.Redirect = all.Redirect
	o.Ssl = all.Ssl
	o.Tcp = all.Tcp
	o.Total = all.Total
	o.Wait = all.Wait
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestRequestCertificateItem Define a request certificate.
type SyntheticsTestRequestCertificateItem struct {
	// Content of the certificate or key.
	Content *string `json:"content,omitempty"`
	// File name for the certificate or key.
	Filename *string `json:"filename,omitempty"`
	// Date of update of the certificate or key, ISO format.
	UpdatedAt *string `json:"updatedAt,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestRequestCertificateItem instantiates a new SyntheticsTestRequestCertificateItem object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestRequestCertificateItem() *SyntheticsTestRequestCertificateItem {
	this := SyntheticsTestRequestCertificateItem{}
	return &this
}

// NewSyntheticsTestRequestCertificateItemWithDefaults instantiates a new SyntheticsTestRequestCertificateItem object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestRequestCertificateItemWithDefaults() *SyntheticsTestRequestCertificateItem {
	this := SyntheticsTestRequestCertificateItem{}
	return &this
}

// GetContent returns the Content field value if set, zero value otherwise.
func (o *SyntheticsTestRequestCertificateItem) GetContent() string {
	if o == nil || o.Content == nil {
		var ret string
		return ret
	}
	return *o.Content
}

// GetContentOk returns a tuple with the Content field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequestCertificateItem) GetContentOk() (*string, bool) {
	if o == nil || o.Content == nil {
		return nil, false
	}
	return o.Content, true
}

// HasContent returns a boolean if a field has been set.
func (o *SyntheticsTestRequestCertificateItem) HasContent() bool {
	if o != nil && o.Content != nil {
		return true
	}

	return false
}

// SetContent gets a reference to the given string and assigns it to the Content field.
func (o *SyntheticsTestRequestCertificateItem) SetContent(v string) {
	o.Content = &v
}

// GetFilename returns the Filename field value if set, zero value otherwise.
func (o *SyntheticsTestRequestCertificateItem) GetFilename() string {
	if o == nil || o.Filename == nil {
		var ret string
		return ret
	}
	return *o.Filename
}

// GetFilenameOk returns a tuple with the Filename field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequestCertificateItem) GetFilenameOk() (*string, bool) {
	if o == nil || o.Filename == nil {
		return nil, false
	}
	return o.Filename, true
}

// HasFilename returns a boolean if a field has been set.
func (o *SyntheticsTestRequestCertificateItem) HasFilename() bool {
	if o != nil && o.Filename != nil {
		return true
	}

	return false
}

// SetFilename gets a reference to the given string and assigns it to the Filename field.
func (o *SyntheticsTestRequestCertificateItem) SetFilename(v string) {
	o.Filename = &v
}

// GetUpdatedAt returns the UpdatedAt field value if set, zero value otherwise.
func (o *SyntheticsTestRequestCertificateItem) GetUpdatedAt() string {
	if o == nil || o.UpdatedAt == nil {
		var ret string
		return ret
	}
	return *o.UpdatedAt
}

// GetUpdatedAtOk returns a tuple with the UpdatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequestCertificateItem) GetUpdatedAtOk() (*string, bool) {
	if o == nil || o.UpdatedAt == nil {
		return nil, false
	}
	return o.UpdatedAt, true
}

// HasUpdatedAt returns a boolean if a field has been set.
func (o *SyntheticsTestRequestCertificateItem) HasUpdatedAt() bool {
	if o != nil && o.UpdatedAt != nil {
		return true
	}

	return false
}

// SetUpdatedAt gets a reference to the given string and assigns it to the UpdatedAt field.
func (o *SyntheticsTestRequestCertificateItem) SetUpdatedAt(v string) {
	o.UpdatedAt = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestRequestCertificateItem) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Content != nil {
		toSerialize["content"] = o.Content
	}
	if o.Filename != nil {
		toSerialize["filename"] = o.Filename
	}
	if o.UpdatedAt != nil {
		toSerialize["updatedAt"] = o.UpdatedAt
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestRequestCertificateItem) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Content   *string `json:"content,omitempty"`
		Filename  *string `json:"filename,omitempty"`
		UpdatedAt *string `json:"updatedAt,omitempty"`
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
	o.Content = all.Content
	o.Filename = all.Filename
	o.UpdatedAt = all.UpdatedAt
	return nil
}

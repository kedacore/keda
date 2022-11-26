// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// WidgetCustomLink Custom links help you connect a data value to a URL, like a Datadog page or your AWS console.
type WidgetCustomLink struct {
	// The flag for toggling context menu link visibility.
	IsHidden *bool `json:"is_hidden,omitempty"`
	// The label for the custom link URL. Keep the label short and descriptive. Use metrics and tags as variables.
	Label *string `json:"label,omitempty"`
	// The URL of the custom link. URL must include `http` or `https`. A relative URL must start with `/`.
	Link *string `json:"link,omitempty"`
	// The label ID that refers to a context menu link. Can be `logs`, `hosts`, `traces`, `profiles`, `processes`, `containers`, or `rum`.
	OverrideLabel *string `json:"override_label,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetCustomLink instantiates a new WidgetCustomLink object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetCustomLink() *WidgetCustomLink {
	this := WidgetCustomLink{}
	return &this
}

// NewWidgetCustomLinkWithDefaults instantiates a new WidgetCustomLink object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetCustomLinkWithDefaults() *WidgetCustomLink {
	this := WidgetCustomLink{}
	return &this
}

// GetIsHidden returns the IsHidden field value if set, zero value otherwise.
func (o *WidgetCustomLink) GetIsHidden() bool {
	if o == nil || o.IsHidden == nil {
		var ret bool
		return ret
	}
	return *o.IsHidden
}

// GetIsHiddenOk returns a tuple with the IsHidden field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetCustomLink) GetIsHiddenOk() (*bool, bool) {
	if o == nil || o.IsHidden == nil {
		return nil, false
	}
	return o.IsHidden, true
}

// HasIsHidden returns a boolean if a field has been set.
func (o *WidgetCustomLink) HasIsHidden() bool {
	if o != nil && o.IsHidden != nil {
		return true
	}

	return false
}

// SetIsHidden gets a reference to the given bool and assigns it to the IsHidden field.
func (o *WidgetCustomLink) SetIsHidden(v bool) {
	o.IsHidden = &v
}

// GetLabel returns the Label field value if set, zero value otherwise.
func (o *WidgetCustomLink) GetLabel() string {
	if o == nil || o.Label == nil {
		var ret string
		return ret
	}
	return *o.Label
}

// GetLabelOk returns a tuple with the Label field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetCustomLink) GetLabelOk() (*string, bool) {
	if o == nil || o.Label == nil {
		return nil, false
	}
	return o.Label, true
}

// HasLabel returns a boolean if a field has been set.
func (o *WidgetCustomLink) HasLabel() bool {
	if o != nil && o.Label != nil {
		return true
	}

	return false
}

// SetLabel gets a reference to the given string and assigns it to the Label field.
func (o *WidgetCustomLink) SetLabel(v string) {
	o.Label = &v
}

// GetLink returns the Link field value if set, zero value otherwise.
func (o *WidgetCustomLink) GetLink() string {
	if o == nil || o.Link == nil {
		var ret string
		return ret
	}
	return *o.Link
}

// GetLinkOk returns a tuple with the Link field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetCustomLink) GetLinkOk() (*string, bool) {
	if o == nil || o.Link == nil {
		return nil, false
	}
	return o.Link, true
}

// HasLink returns a boolean if a field has been set.
func (o *WidgetCustomLink) HasLink() bool {
	if o != nil && o.Link != nil {
		return true
	}

	return false
}

// SetLink gets a reference to the given string and assigns it to the Link field.
func (o *WidgetCustomLink) SetLink(v string) {
	o.Link = &v
}

// GetOverrideLabel returns the OverrideLabel field value if set, zero value otherwise.
func (o *WidgetCustomLink) GetOverrideLabel() string {
	if o == nil || o.OverrideLabel == nil {
		var ret string
		return ret
	}
	return *o.OverrideLabel
}

// GetOverrideLabelOk returns a tuple with the OverrideLabel field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetCustomLink) GetOverrideLabelOk() (*string, bool) {
	if o == nil || o.OverrideLabel == nil {
		return nil, false
	}
	return o.OverrideLabel, true
}

// HasOverrideLabel returns a boolean if a field has been set.
func (o *WidgetCustomLink) HasOverrideLabel() bool {
	if o != nil && o.OverrideLabel != nil {
		return true
	}

	return false
}

// SetOverrideLabel gets a reference to the given string and assigns it to the OverrideLabel field.
func (o *WidgetCustomLink) SetOverrideLabel(v string) {
	o.OverrideLabel = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetCustomLink) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IsHidden != nil {
		toSerialize["is_hidden"] = o.IsHidden
	}
	if o.Label != nil {
		toSerialize["label"] = o.Label
	}
	if o.Link != nil {
		toSerialize["link"] = o.Link
	}
	if o.OverrideLabel != nil {
		toSerialize["override_label"] = o.OverrideLabel
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetCustomLink) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IsHidden      *bool   `json:"is_hidden,omitempty"`
		Label         *string `json:"label,omitempty"`
		Link          *string `json:"link,omitempty"`
		OverrideLabel *string `json:"override_label,omitempty"`
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
	o.IsHidden = all.IsHidden
	o.Label = all.Label
	o.Link = all.Link
	o.OverrideLabel = all.OverrideLabel
	return nil
}

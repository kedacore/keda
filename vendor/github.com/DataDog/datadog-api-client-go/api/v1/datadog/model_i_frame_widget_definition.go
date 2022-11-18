// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// IFrameWidgetDefinition The iframe widget allows you to embed a portion of any other web page on your dashboard. Only available on FREE layout dashboards.
type IFrameWidgetDefinition struct {
	// Type of the iframe widget.
	Type IFrameWidgetDefinitionType `json:"type"`
	// URL of the iframe.
	Url string `json:"url"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewIFrameWidgetDefinition instantiates a new IFrameWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIFrameWidgetDefinition(typeVar IFrameWidgetDefinitionType, url string) *IFrameWidgetDefinition {
	this := IFrameWidgetDefinition{}
	this.Type = typeVar
	this.Url = url
	return &this
}

// NewIFrameWidgetDefinitionWithDefaults instantiates a new IFrameWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIFrameWidgetDefinitionWithDefaults() *IFrameWidgetDefinition {
	this := IFrameWidgetDefinition{}
	var typeVar IFrameWidgetDefinitionType = IFRAMEWIDGETDEFINITIONTYPE_IFRAME
	this.Type = typeVar
	return &this
}

// GetType returns the Type field value.
func (o *IFrameWidgetDefinition) GetType() IFrameWidgetDefinitionType {
	if o == nil {
		var ret IFrameWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IFrameWidgetDefinition) GetTypeOk() (*IFrameWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IFrameWidgetDefinition) SetType(v IFrameWidgetDefinitionType) {
	o.Type = v
}

// GetUrl returns the Url field value.
func (o *IFrameWidgetDefinition) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *IFrameWidgetDefinition) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value.
func (o *IFrameWidgetDefinition) SetUrl(v string) {
	o.Url = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IFrameWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["type"] = o.Type
	toSerialize["url"] = o.Url

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IFrameWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *IFrameWidgetDefinitionType `json:"type"`
		Url  *string                     `json:"url"`
	}{}
	all := struct {
		Type IFrameWidgetDefinitionType `json:"type"`
		Url  string                     `json:"url"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	if required.Url == nil {
		return fmt.Errorf("Required field url missing")
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
	o.Type = all.Type
	o.Url = all.Url
	return nil
}

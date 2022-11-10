// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// AlertValueWidgetDefinition Alert values are query values showing the current value of the metric in any monitor defined on your system.
type AlertValueWidgetDefinition struct {
	// ID of the alert to use in the widget.
	AlertId string `json:"alert_id"`
	// Number of decimal to show. If not defined, will use the raw value.
	Precision *int64 `json:"precision,omitempty"`
	// How to align the text on the widget.
	TextAlign *WidgetTextAlign `json:"text_align,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of value in the widget.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the alert value widget.
	Type AlertValueWidgetDefinitionType `json:"type"`
	// Unit to display with the value.
	Unit *string `json:"unit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewAlertValueWidgetDefinition instantiates a new AlertValueWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAlertValueWidgetDefinition(alertId string, typeVar AlertValueWidgetDefinitionType) *AlertValueWidgetDefinition {
	this := AlertValueWidgetDefinition{}
	this.AlertId = alertId
	this.Type = typeVar
	return &this
}

// NewAlertValueWidgetDefinitionWithDefaults instantiates a new AlertValueWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAlertValueWidgetDefinitionWithDefaults() *AlertValueWidgetDefinition {
	this := AlertValueWidgetDefinition{}
	var typeVar AlertValueWidgetDefinitionType = ALERTVALUEWIDGETDEFINITIONTYPE_ALERT_VALUE
	this.Type = typeVar
	return &this
}

// GetAlertId returns the AlertId field value.
func (o *AlertValueWidgetDefinition) GetAlertId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AlertId
}

// GetAlertIdOk returns a tuple with the AlertId field value
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetAlertIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AlertId, true
}

// SetAlertId sets field value.
func (o *AlertValueWidgetDefinition) SetAlertId(v string) {
	o.AlertId = v
}

// GetPrecision returns the Precision field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetPrecision() int64 {
	if o == nil || o.Precision == nil {
		var ret int64
		return ret
	}
	return *o.Precision
}

// GetPrecisionOk returns a tuple with the Precision field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetPrecisionOk() (*int64, bool) {
	if o == nil || o.Precision == nil {
		return nil, false
	}
	return o.Precision, true
}

// HasPrecision returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasPrecision() bool {
	if o != nil && o.Precision != nil {
		return true
	}

	return false
}

// SetPrecision gets a reference to the given int64 and assigns it to the Precision field.
func (o *AlertValueWidgetDefinition) SetPrecision(v int64) {
	o.Precision = &v
}

// GetTextAlign returns the TextAlign field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetTextAlign() WidgetTextAlign {
	if o == nil || o.TextAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TextAlign
}

// GetTextAlignOk returns a tuple with the TextAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetTextAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TextAlign == nil {
		return nil, false
	}
	return o.TextAlign, true
}

// HasTextAlign returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasTextAlign() bool {
	if o != nil && o.TextAlign != nil {
		return true
	}

	return false
}

// SetTextAlign gets a reference to the given WidgetTextAlign and assigns it to the TextAlign field.
func (o *AlertValueWidgetDefinition) SetTextAlign(v WidgetTextAlign) {
	o.TextAlign = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *AlertValueWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *AlertValueWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *AlertValueWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *AlertValueWidgetDefinition) GetType() AlertValueWidgetDefinitionType {
	if o == nil {
		var ret AlertValueWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetTypeOk() (*AlertValueWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *AlertValueWidgetDefinition) SetType(v AlertValueWidgetDefinitionType) {
	o.Type = v
}

// GetUnit returns the Unit field value if set, zero value otherwise.
func (o *AlertValueWidgetDefinition) GetUnit() string {
	if o == nil || o.Unit == nil {
		var ret string
		return ret
	}
	return *o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AlertValueWidgetDefinition) GetUnitOk() (*string, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *AlertValueWidgetDefinition) HasUnit() bool {
	if o != nil && o.Unit != nil {
		return true
	}

	return false
}

// SetUnit gets a reference to the given string and assigns it to the Unit field.
func (o *AlertValueWidgetDefinition) SetUnit(v string) {
	o.Unit = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AlertValueWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["alert_id"] = o.AlertId
	if o.Precision != nil {
		toSerialize["precision"] = o.Precision
	}
	if o.TextAlign != nil {
		toSerialize["text_align"] = o.TextAlign
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.TitleAlign != nil {
		toSerialize["title_align"] = o.TitleAlign
	}
	if o.TitleSize != nil {
		toSerialize["title_size"] = o.TitleSize
	}
	toSerialize["type"] = o.Type
	if o.Unit != nil {
		toSerialize["unit"] = o.Unit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AlertValueWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AlertId *string                         `json:"alert_id"`
		Type    *AlertValueWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		AlertId    string                         `json:"alert_id"`
		Precision  *int64                         `json:"precision,omitempty"`
		TextAlign  *WidgetTextAlign               `json:"text_align,omitempty"`
		Title      *string                        `json:"title,omitempty"`
		TitleAlign *WidgetTextAlign               `json:"title_align,omitempty"`
		TitleSize  *string                        `json:"title_size,omitempty"`
		Type       AlertValueWidgetDefinitionType `json:"type"`
		Unit       *string                        `json:"unit,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AlertId == nil {
		return fmt.Errorf("Required field alert_id missing")
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
	if v := all.TextAlign; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.TitleAlign; v != nil && !v.IsValid() {
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
	o.AlertId = all.AlertId
	o.Precision = all.Precision
	o.TextAlign = all.TextAlign
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	o.Unit = all.Unit
	return nil
}

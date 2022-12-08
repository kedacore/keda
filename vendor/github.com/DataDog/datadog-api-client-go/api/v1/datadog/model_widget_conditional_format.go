// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetConditionalFormat Define a conditional format for the widget.
type WidgetConditionalFormat struct {
	// Comparator to apply.
	Comparator WidgetComparator `json:"comparator"`
	// Color palette to apply to the background, same values available as palette.
	CustomBgColor *string `json:"custom_bg_color,omitempty"`
	// Color palette to apply to the foreground, same values available as palette.
	CustomFgColor *string `json:"custom_fg_color,omitempty"`
	// True hides values.
	HideValue *bool `json:"hide_value,omitempty"`
	// Displays an image as the background.
	ImageUrl *string `json:"image_url,omitempty"`
	// Metric from the request to correlate this conditional format with.
	Metric *string `json:"metric,omitempty"`
	// Color palette to apply.
	Palette WidgetPalette `json:"palette"`
	// Defines the displayed timeframe.
	Timeframe *string `json:"timeframe,omitempty"`
	// Value for the comparator.
	Value float64 `json:"value"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetConditionalFormat instantiates a new WidgetConditionalFormat object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetConditionalFormat(comparator WidgetComparator, palette WidgetPalette, value float64) *WidgetConditionalFormat {
	this := WidgetConditionalFormat{}
	this.Comparator = comparator
	this.Palette = palette
	this.Value = value
	return &this
}

// NewWidgetConditionalFormatWithDefaults instantiates a new WidgetConditionalFormat object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetConditionalFormatWithDefaults() *WidgetConditionalFormat {
	this := WidgetConditionalFormat{}
	return &this
}

// GetComparator returns the Comparator field value.
func (o *WidgetConditionalFormat) GetComparator() WidgetComparator {
	if o == nil {
		var ret WidgetComparator
		return ret
	}
	return o.Comparator
}

// GetComparatorOk returns a tuple with the Comparator field value
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetComparatorOk() (*WidgetComparator, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Comparator, true
}

// SetComparator sets field value.
func (o *WidgetConditionalFormat) SetComparator(v WidgetComparator) {
	o.Comparator = v
}

// GetCustomBgColor returns the CustomBgColor field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetCustomBgColor() string {
	if o == nil || o.CustomBgColor == nil {
		var ret string
		return ret
	}
	return *o.CustomBgColor
}

// GetCustomBgColorOk returns a tuple with the CustomBgColor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetCustomBgColorOk() (*string, bool) {
	if o == nil || o.CustomBgColor == nil {
		return nil, false
	}
	return o.CustomBgColor, true
}

// HasCustomBgColor returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasCustomBgColor() bool {
	if o != nil && o.CustomBgColor != nil {
		return true
	}

	return false
}

// SetCustomBgColor gets a reference to the given string and assigns it to the CustomBgColor field.
func (o *WidgetConditionalFormat) SetCustomBgColor(v string) {
	o.CustomBgColor = &v
}

// GetCustomFgColor returns the CustomFgColor field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetCustomFgColor() string {
	if o == nil || o.CustomFgColor == nil {
		var ret string
		return ret
	}
	return *o.CustomFgColor
}

// GetCustomFgColorOk returns a tuple with the CustomFgColor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetCustomFgColorOk() (*string, bool) {
	if o == nil || o.CustomFgColor == nil {
		return nil, false
	}
	return o.CustomFgColor, true
}

// HasCustomFgColor returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasCustomFgColor() bool {
	if o != nil && o.CustomFgColor != nil {
		return true
	}

	return false
}

// SetCustomFgColor gets a reference to the given string and assigns it to the CustomFgColor field.
func (o *WidgetConditionalFormat) SetCustomFgColor(v string) {
	o.CustomFgColor = &v
}

// GetHideValue returns the HideValue field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetHideValue() bool {
	if o == nil || o.HideValue == nil {
		var ret bool
		return ret
	}
	return *o.HideValue
}

// GetHideValueOk returns a tuple with the HideValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetHideValueOk() (*bool, bool) {
	if o == nil || o.HideValue == nil {
		return nil, false
	}
	return o.HideValue, true
}

// HasHideValue returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasHideValue() bool {
	if o != nil && o.HideValue != nil {
		return true
	}

	return false
}

// SetHideValue gets a reference to the given bool and assigns it to the HideValue field.
func (o *WidgetConditionalFormat) SetHideValue(v bool) {
	o.HideValue = &v
}

// GetImageUrl returns the ImageUrl field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetImageUrl() string {
	if o == nil || o.ImageUrl == nil {
		var ret string
		return ret
	}
	return *o.ImageUrl
}

// GetImageUrlOk returns a tuple with the ImageUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetImageUrlOk() (*string, bool) {
	if o == nil || o.ImageUrl == nil {
		return nil, false
	}
	return o.ImageUrl, true
}

// HasImageUrl returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasImageUrl() bool {
	if o != nil && o.ImageUrl != nil {
		return true
	}

	return false
}

// SetImageUrl gets a reference to the given string and assigns it to the ImageUrl field.
func (o *WidgetConditionalFormat) SetImageUrl(v string) {
	o.ImageUrl = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasMetric() bool {
	if o != nil && o.Metric != nil {
		return true
	}

	return false
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *WidgetConditionalFormat) SetMetric(v string) {
	o.Metric = &v
}

// GetPalette returns the Palette field value.
func (o *WidgetConditionalFormat) GetPalette() WidgetPalette {
	if o == nil {
		var ret WidgetPalette
		return ret
	}
	return o.Palette
}

// GetPaletteOk returns a tuple with the Palette field value
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetPaletteOk() (*WidgetPalette, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Palette, true
}

// SetPalette sets field value.
func (o *WidgetConditionalFormat) SetPalette(v WidgetPalette) {
	o.Palette = v
}

// GetTimeframe returns the Timeframe field value if set, zero value otherwise.
func (o *WidgetConditionalFormat) GetTimeframe() string {
	if o == nil || o.Timeframe == nil {
		var ret string
		return ret
	}
	return *o.Timeframe
}

// GetTimeframeOk returns a tuple with the Timeframe field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetTimeframeOk() (*string, bool) {
	if o == nil || o.Timeframe == nil {
		return nil, false
	}
	return o.Timeframe, true
}

// HasTimeframe returns a boolean if a field has been set.
func (o *WidgetConditionalFormat) HasTimeframe() bool {
	if o != nil && o.Timeframe != nil {
		return true
	}

	return false
}

// SetTimeframe gets a reference to the given string and assigns it to the Timeframe field.
func (o *WidgetConditionalFormat) SetTimeframe(v string) {
	o.Timeframe = &v
}

// GetValue returns the Value field value.
func (o *WidgetConditionalFormat) GetValue() float64 {
	if o == nil {
		var ret float64
		return ret
	}
	return o.Value
}

// GetValueOk returns a tuple with the Value field value
// and a boolean to check if the value has been set.
func (o *WidgetConditionalFormat) GetValueOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Value, true
}

// SetValue sets field value.
func (o *WidgetConditionalFormat) SetValue(v float64) {
	o.Value = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetConditionalFormat) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["comparator"] = o.Comparator
	if o.CustomBgColor != nil {
		toSerialize["custom_bg_color"] = o.CustomBgColor
	}
	if o.CustomFgColor != nil {
		toSerialize["custom_fg_color"] = o.CustomFgColor
	}
	if o.HideValue != nil {
		toSerialize["hide_value"] = o.HideValue
	}
	if o.ImageUrl != nil {
		toSerialize["image_url"] = o.ImageUrl
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	toSerialize["palette"] = o.Palette
	if o.Timeframe != nil {
		toSerialize["timeframe"] = o.Timeframe
	}
	toSerialize["value"] = o.Value

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetConditionalFormat) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Comparator *WidgetComparator `json:"comparator"`
		Palette    *WidgetPalette    `json:"palette"`
		Value      *float64          `json:"value"`
	}{}
	all := struct {
		Comparator    WidgetComparator `json:"comparator"`
		CustomBgColor *string          `json:"custom_bg_color,omitempty"`
		CustomFgColor *string          `json:"custom_fg_color,omitempty"`
		HideValue     *bool            `json:"hide_value,omitempty"`
		ImageUrl      *string          `json:"image_url,omitempty"`
		Metric        *string          `json:"metric,omitempty"`
		Palette       WidgetPalette    `json:"palette"`
		Timeframe     *string          `json:"timeframe,omitempty"`
		Value         float64          `json:"value"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Comparator == nil {
		return fmt.Errorf("Required field comparator missing")
	}
	if required.Palette == nil {
		return fmt.Errorf("Required field palette missing")
	}
	if required.Value == nil {
		return fmt.Errorf("Required field value missing")
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
	if v := all.Comparator; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Palette; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Comparator = all.Comparator
	o.CustomBgColor = all.CustomBgColor
	o.CustomFgColor = all.CustomFgColor
	o.HideValue = all.HideValue
	o.ImageUrl = all.ImageUrl
	o.Metric = all.Metric
	o.Palette = all.Palette
	o.Timeframe = all.Timeframe
	o.Value = all.Value
	return nil
}

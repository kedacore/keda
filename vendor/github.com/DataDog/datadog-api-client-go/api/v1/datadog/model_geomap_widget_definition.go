// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// GeomapWidgetDefinition This visualization displays a series of values by country on a world map.
type GeomapWidgetDefinition struct {
	// A list of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Array of one request object to display in the widget. The request must contain a `group-by` tag whose value is a country ISO code.
	//
	// See the [Request JSON schema documentation](https://docs.datadoghq.com/dashboards/graphing_json/request_json)
	// for information about building the `REQUEST_SCHEMA`.
	Requests []GeomapWidgetRequest `json:"requests"`
	// The style to apply to the widget.
	Style GeomapWidgetDefinitionStyle `json:"style"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// The title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// The size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the geomap widget.
	Type GeomapWidgetDefinitionType `json:"type"`
	// The view of the world that the map should render.
	View GeomapWidgetDefinitionView `json:"view"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewGeomapWidgetDefinition instantiates a new GeomapWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewGeomapWidgetDefinition(requests []GeomapWidgetRequest, style GeomapWidgetDefinitionStyle, typeVar GeomapWidgetDefinitionType, view GeomapWidgetDefinitionView) *GeomapWidgetDefinition {
	this := GeomapWidgetDefinition{}
	this.Requests = requests
	this.Style = style
	this.Type = typeVar
	this.View = view
	return &this
}

// NewGeomapWidgetDefinitionWithDefaults instantiates a new GeomapWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewGeomapWidgetDefinitionWithDefaults() *GeomapWidgetDefinition {
	this := GeomapWidgetDefinition{}
	var typeVar GeomapWidgetDefinitionType = GEOMAPWIDGETDEFINITIONTYPE_GEOMAP
	this.Type = typeVar
	return &this
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *GeomapWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *GeomapWidgetDefinition) HasCustomLinks() bool {
	if o != nil && o.CustomLinks != nil {
		return true
	}

	return false
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *GeomapWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetRequests returns the Requests field value.
func (o *GeomapWidgetDefinition) GetRequests() []GeomapWidgetRequest {
	if o == nil {
		var ret []GeomapWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetRequestsOk() (*[]GeomapWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *GeomapWidgetDefinition) SetRequests(v []GeomapWidgetRequest) {
	o.Requests = v
}

// GetStyle returns the Style field value.
func (o *GeomapWidgetDefinition) GetStyle() GeomapWidgetDefinitionStyle {
	if o == nil {
		var ret GeomapWidgetDefinitionStyle
		return ret
	}
	return o.Style
}

// GetStyleOk returns a tuple with the Style field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetStyleOk() (*GeomapWidgetDefinitionStyle, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Style, true
}

// SetStyle sets field value.
func (o *GeomapWidgetDefinition) SetStyle(v GeomapWidgetDefinitionStyle) {
	o.Style = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *GeomapWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *GeomapWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *GeomapWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *GeomapWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *GeomapWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *GeomapWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *GeomapWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *GeomapWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *GeomapWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *GeomapWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *GeomapWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *GeomapWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *GeomapWidgetDefinition) GetType() GeomapWidgetDefinitionType {
	if o == nil {
		var ret GeomapWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetTypeOk() (*GeomapWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *GeomapWidgetDefinition) SetType(v GeomapWidgetDefinitionType) {
	o.Type = v
}

// GetView returns the View field value.
func (o *GeomapWidgetDefinition) GetView() GeomapWidgetDefinitionView {
	if o == nil {
		var ret GeomapWidgetDefinitionView
		return ret
	}
	return o.View
}

// GetViewOk returns a tuple with the View field value
// and a boolean to check if the value has been set.
func (o *GeomapWidgetDefinition) GetViewOk() (*GeomapWidgetDefinitionView, bool) {
	if o == nil {
		return nil, false
	}
	return &o.View, true
}

// SetView sets field value.
func (o *GeomapWidgetDefinition) SetView(v GeomapWidgetDefinitionView) {
	o.View = v
}

// MarshalJSON serializes the struct using spec logic.
func (o GeomapWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
	}
	toSerialize["requests"] = o.Requests
	toSerialize["style"] = o.Style
	if o.Time != nil {
		toSerialize["time"] = o.Time
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
	toSerialize["view"] = o.View

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *GeomapWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]GeomapWidgetRequest       `json:"requests"`
		Style    *GeomapWidgetDefinitionStyle `json:"style"`
		Type     *GeomapWidgetDefinitionType  `json:"type"`
		View     *GeomapWidgetDefinitionView  `json:"view"`
	}{}
	all := struct {
		CustomLinks []WidgetCustomLink          `json:"custom_links,omitempty"`
		Requests    []GeomapWidgetRequest       `json:"requests"`
		Style       GeomapWidgetDefinitionStyle `json:"style"`
		Time        *WidgetTime                 `json:"time,omitempty"`
		Title       *string                     `json:"title,omitempty"`
		TitleAlign  *WidgetTextAlign            `json:"title_align,omitempty"`
		TitleSize   *string                     `json:"title_size,omitempty"`
		Type        GeomapWidgetDefinitionType  `json:"type"`
		View        GeomapWidgetDefinitionView  `json:"view"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Requests == nil {
		return fmt.Errorf("Required field requests missing")
	}
	if required.Style == nil {
		return fmt.Errorf("Required field style missing")
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	if required.View == nil {
		return fmt.Errorf("Required field view missing")
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
	o.CustomLinks = all.CustomLinks
	o.Requests = all.Requests
	if all.Style.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Style = all.Style
	if all.Time != nil && all.Time.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Time = all.Time
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	if all.View.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.View = all.View
	return nil
}

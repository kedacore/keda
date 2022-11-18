// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ServiceSummaryWidgetDefinition The service summary displays the graphs of a chosen service in your screenboard. Only available on FREE layout dashboards.
type ServiceSummaryWidgetDefinition struct {
	// Number of columns to display.
	DisplayFormat *WidgetServiceSummaryDisplayFormat `json:"display_format,omitempty"`
	// APM environment.
	Env string `json:"env"`
	// APM service.
	Service string `json:"service"`
	// Whether to show the latency breakdown or not.
	ShowBreakdown *bool `json:"show_breakdown,omitempty"`
	// Whether to show the latency distribution or not.
	ShowDistribution *bool `json:"show_distribution,omitempty"`
	// Whether to show the error metrics or not.
	ShowErrors *bool `json:"show_errors,omitempty"`
	// Whether to show the hits metrics or not.
	ShowHits *bool `json:"show_hits,omitempty"`
	// Whether to show the latency metrics or not.
	ShowLatency *bool `json:"show_latency,omitempty"`
	// Whether to show the resource list or not.
	ShowResourceList *bool `json:"show_resource_list,omitempty"`
	// Size of the widget.
	SizeFormat *WidgetSizeFormat `json:"size_format,omitempty"`
	// APM span name.
	SpanName string `json:"span_name"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the service summary widget.
	Type ServiceSummaryWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewServiceSummaryWidgetDefinition instantiates a new ServiceSummaryWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceSummaryWidgetDefinition(env string, service string, spanName string, typeVar ServiceSummaryWidgetDefinitionType) *ServiceSummaryWidgetDefinition {
	this := ServiceSummaryWidgetDefinition{}
	this.Env = env
	this.Service = service
	this.SpanName = spanName
	this.Type = typeVar
	return &this
}

// NewServiceSummaryWidgetDefinitionWithDefaults instantiates a new ServiceSummaryWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceSummaryWidgetDefinitionWithDefaults() *ServiceSummaryWidgetDefinition {
	this := ServiceSummaryWidgetDefinition{}
	var typeVar ServiceSummaryWidgetDefinitionType = SERVICESUMMARYWIDGETDEFINITIONTYPE_TRACE_SERVICE
	this.Type = typeVar
	return &this
}

// GetDisplayFormat returns the DisplayFormat field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetDisplayFormat() WidgetServiceSummaryDisplayFormat {
	if o == nil || o.DisplayFormat == nil {
		var ret WidgetServiceSummaryDisplayFormat
		return ret
	}
	return *o.DisplayFormat
}

// GetDisplayFormatOk returns a tuple with the DisplayFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetDisplayFormatOk() (*WidgetServiceSummaryDisplayFormat, bool) {
	if o == nil || o.DisplayFormat == nil {
		return nil, false
	}
	return o.DisplayFormat, true
}

// HasDisplayFormat returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasDisplayFormat() bool {
	if o != nil && o.DisplayFormat != nil {
		return true
	}

	return false
}

// SetDisplayFormat gets a reference to the given WidgetServiceSummaryDisplayFormat and assigns it to the DisplayFormat field.
func (o *ServiceSummaryWidgetDefinition) SetDisplayFormat(v WidgetServiceSummaryDisplayFormat) {
	o.DisplayFormat = &v
}

// GetEnv returns the Env field value.
func (o *ServiceSummaryWidgetDefinition) GetEnv() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Env
}

// GetEnvOk returns a tuple with the Env field value
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetEnvOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Env, true
}

// SetEnv sets field value.
func (o *ServiceSummaryWidgetDefinition) SetEnv(v string) {
	o.Env = v
}

// GetService returns the Service field value.
func (o *ServiceSummaryWidgetDefinition) GetService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Service
}

// GetServiceOk returns a tuple with the Service field value
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Service, true
}

// SetService sets field value.
func (o *ServiceSummaryWidgetDefinition) SetService(v string) {
	o.Service = v
}

// GetShowBreakdown returns the ShowBreakdown field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowBreakdown() bool {
	if o == nil || o.ShowBreakdown == nil {
		var ret bool
		return ret
	}
	return *o.ShowBreakdown
}

// GetShowBreakdownOk returns a tuple with the ShowBreakdown field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowBreakdownOk() (*bool, bool) {
	if o == nil || o.ShowBreakdown == nil {
		return nil, false
	}
	return o.ShowBreakdown, true
}

// HasShowBreakdown returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowBreakdown() bool {
	if o != nil && o.ShowBreakdown != nil {
		return true
	}

	return false
}

// SetShowBreakdown gets a reference to the given bool and assigns it to the ShowBreakdown field.
func (o *ServiceSummaryWidgetDefinition) SetShowBreakdown(v bool) {
	o.ShowBreakdown = &v
}

// GetShowDistribution returns the ShowDistribution field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowDistribution() bool {
	if o == nil || o.ShowDistribution == nil {
		var ret bool
		return ret
	}
	return *o.ShowDistribution
}

// GetShowDistributionOk returns a tuple with the ShowDistribution field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowDistributionOk() (*bool, bool) {
	if o == nil || o.ShowDistribution == nil {
		return nil, false
	}
	return o.ShowDistribution, true
}

// HasShowDistribution returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowDistribution() bool {
	if o != nil && o.ShowDistribution != nil {
		return true
	}

	return false
}

// SetShowDistribution gets a reference to the given bool and assigns it to the ShowDistribution field.
func (o *ServiceSummaryWidgetDefinition) SetShowDistribution(v bool) {
	o.ShowDistribution = &v
}

// GetShowErrors returns the ShowErrors field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowErrors() bool {
	if o == nil || o.ShowErrors == nil {
		var ret bool
		return ret
	}
	return *o.ShowErrors
}

// GetShowErrorsOk returns a tuple with the ShowErrors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowErrorsOk() (*bool, bool) {
	if o == nil || o.ShowErrors == nil {
		return nil, false
	}
	return o.ShowErrors, true
}

// HasShowErrors returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowErrors() bool {
	if o != nil && o.ShowErrors != nil {
		return true
	}

	return false
}

// SetShowErrors gets a reference to the given bool and assigns it to the ShowErrors field.
func (o *ServiceSummaryWidgetDefinition) SetShowErrors(v bool) {
	o.ShowErrors = &v
}

// GetShowHits returns the ShowHits field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowHits() bool {
	if o == nil || o.ShowHits == nil {
		var ret bool
		return ret
	}
	return *o.ShowHits
}

// GetShowHitsOk returns a tuple with the ShowHits field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowHitsOk() (*bool, bool) {
	if o == nil || o.ShowHits == nil {
		return nil, false
	}
	return o.ShowHits, true
}

// HasShowHits returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowHits() bool {
	if o != nil && o.ShowHits != nil {
		return true
	}

	return false
}

// SetShowHits gets a reference to the given bool and assigns it to the ShowHits field.
func (o *ServiceSummaryWidgetDefinition) SetShowHits(v bool) {
	o.ShowHits = &v
}

// GetShowLatency returns the ShowLatency field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowLatency() bool {
	if o == nil || o.ShowLatency == nil {
		var ret bool
		return ret
	}
	return *o.ShowLatency
}

// GetShowLatencyOk returns a tuple with the ShowLatency field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowLatencyOk() (*bool, bool) {
	if o == nil || o.ShowLatency == nil {
		return nil, false
	}
	return o.ShowLatency, true
}

// HasShowLatency returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowLatency() bool {
	if o != nil && o.ShowLatency != nil {
		return true
	}

	return false
}

// SetShowLatency gets a reference to the given bool and assigns it to the ShowLatency field.
func (o *ServiceSummaryWidgetDefinition) SetShowLatency(v bool) {
	o.ShowLatency = &v
}

// GetShowResourceList returns the ShowResourceList field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetShowResourceList() bool {
	if o == nil || o.ShowResourceList == nil {
		var ret bool
		return ret
	}
	return *o.ShowResourceList
}

// GetShowResourceListOk returns a tuple with the ShowResourceList field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetShowResourceListOk() (*bool, bool) {
	if o == nil || o.ShowResourceList == nil {
		return nil, false
	}
	return o.ShowResourceList, true
}

// HasShowResourceList returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasShowResourceList() bool {
	if o != nil && o.ShowResourceList != nil {
		return true
	}

	return false
}

// SetShowResourceList gets a reference to the given bool and assigns it to the ShowResourceList field.
func (o *ServiceSummaryWidgetDefinition) SetShowResourceList(v bool) {
	o.ShowResourceList = &v
}

// GetSizeFormat returns the SizeFormat field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetSizeFormat() WidgetSizeFormat {
	if o == nil || o.SizeFormat == nil {
		var ret WidgetSizeFormat
		return ret
	}
	return *o.SizeFormat
}

// GetSizeFormatOk returns a tuple with the SizeFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetSizeFormatOk() (*WidgetSizeFormat, bool) {
	if o == nil || o.SizeFormat == nil {
		return nil, false
	}
	return o.SizeFormat, true
}

// HasSizeFormat returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasSizeFormat() bool {
	if o != nil && o.SizeFormat != nil {
		return true
	}

	return false
}

// SetSizeFormat gets a reference to the given WidgetSizeFormat and assigns it to the SizeFormat field.
func (o *ServiceSummaryWidgetDefinition) SetSizeFormat(v WidgetSizeFormat) {
	o.SizeFormat = &v
}

// GetSpanName returns the SpanName field value.
func (o *ServiceSummaryWidgetDefinition) GetSpanName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.SpanName
}

// GetSpanNameOk returns a tuple with the SpanName field value
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetSpanNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SpanName, true
}

// SetSpanName sets field value.
func (o *ServiceSummaryWidgetDefinition) SetSpanName(v string) {
	o.SpanName = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *ServiceSummaryWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *ServiceSummaryWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasTitleAlign() bool {
	if o != nil && o.TitleAlign != nil {
		return true
	}

	return false
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *ServiceSummaryWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *ServiceSummaryWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *ServiceSummaryWidgetDefinition) HasTitleSize() bool {
	if o != nil && o.TitleSize != nil {
		return true
	}

	return false
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *ServiceSummaryWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *ServiceSummaryWidgetDefinition) GetType() ServiceSummaryWidgetDefinitionType {
	if o == nil {
		var ret ServiceSummaryWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceSummaryWidgetDefinition) GetTypeOk() (*ServiceSummaryWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceSummaryWidgetDefinition) SetType(v ServiceSummaryWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceSummaryWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DisplayFormat != nil {
		toSerialize["display_format"] = o.DisplayFormat
	}
	toSerialize["env"] = o.Env
	toSerialize["service"] = o.Service
	if o.ShowBreakdown != nil {
		toSerialize["show_breakdown"] = o.ShowBreakdown
	}
	if o.ShowDistribution != nil {
		toSerialize["show_distribution"] = o.ShowDistribution
	}
	if o.ShowErrors != nil {
		toSerialize["show_errors"] = o.ShowErrors
	}
	if o.ShowHits != nil {
		toSerialize["show_hits"] = o.ShowHits
	}
	if o.ShowLatency != nil {
		toSerialize["show_latency"] = o.ShowLatency
	}
	if o.ShowResourceList != nil {
		toSerialize["show_resource_list"] = o.ShowResourceList
	}
	if o.SizeFormat != nil {
		toSerialize["size_format"] = o.SizeFormat
	}
	toSerialize["span_name"] = o.SpanName
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

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceSummaryWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Env      *string                             `json:"env"`
		Service  *string                             `json:"service"`
		SpanName *string                             `json:"span_name"`
		Type     *ServiceSummaryWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		DisplayFormat    *WidgetServiceSummaryDisplayFormat `json:"display_format,omitempty"`
		Env              string                             `json:"env"`
		Service          string                             `json:"service"`
		ShowBreakdown    *bool                              `json:"show_breakdown,omitempty"`
		ShowDistribution *bool                              `json:"show_distribution,omitempty"`
		ShowErrors       *bool                              `json:"show_errors,omitempty"`
		ShowHits         *bool                              `json:"show_hits,omitempty"`
		ShowLatency      *bool                              `json:"show_latency,omitempty"`
		ShowResourceList *bool                              `json:"show_resource_list,omitempty"`
		SizeFormat       *WidgetSizeFormat                  `json:"size_format,omitempty"`
		SpanName         string                             `json:"span_name"`
		Time             *WidgetTime                        `json:"time,omitempty"`
		Title            *string                            `json:"title,omitempty"`
		TitleAlign       *WidgetTextAlign                   `json:"title_align,omitempty"`
		TitleSize        *string                            `json:"title_size,omitempty"`
		Type             ServiceSummaryWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Env == nil {
		return fmt.Errorf("Required field env missing")
	}
	if required.Service == nil {
		return fmt.Errorf("Required field service missing")
	}
	if required.SpanName == nil {
		return fmt.Errorf("Required field span_name missing")
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
	if v := all.DisplayFormat; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.SizeFormat; v != nil && !v.IsValid() {
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
	o.DisplayFormat = all.DisplayFormat
	o.Env = all.Env
	o.Service = all.Service
	o.ShowBreakdown = all.ShowBreakdown
	o.ShowDistribution = all.ShowDistribution
	o.ShowErrors = all.ShowErrors
	o.ShowHits = all.ShowHits
	o.ShowLatency = all.ShowLatency
	o.ShowResourceList = all.ShowResourceList
	o.SizeFormat = all.SizeFormat
	o.SpanName = all.SpanName
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
	return nil
}

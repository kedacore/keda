// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"time"
)

// Dashboard A dashboard is Datadog’s tool for visually tracking, analyzing, and displaying
// key performance metrics, which enable you to monitor the health of your infrastructure.
type Dashboard struct {
	// Identifier of the dashboard author.
	AuthorHandle *string `json:"author_handle,omitempty"`
	// Name of the dashboard author.
	AuthorName NullableString `json:"author_name,omitempty"`
	// Creation date of the dashboard.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Description of the dashboard.
	Description NullableString `json:"description,omitempty"`
	// ID of the dashboard.
	Id *string `json:"id,omitempty"`
	// Whether this dashboard is read-only. If True, only the author and admins can make changes to it. Prefer using `restricted_roles` to manage write authorization.
	// Deprecated
	IsReadOnly *bool `json:"is_read_only,omitempty"`
	// Layout type of the dashboard.
	LayoutType DashboardLayoutType `json:"layout_type"`
	// Modification date of the dashboard.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// List of handles of users to notify when changes are made to this dashboard.
	NotifyList []string `json:"notify_list,omitempty"`
	// Reflow type for a **new dashboard layout** dashboard. Set this only when layout type is 'ordered'.
	// If set to 'fixed', the dashboard expects all widgets to have a layout, and if it's set to 'auto',
	// widgets should not have layouts.
	ReflowType *DashboardReflowType `json:"reflow_type,omitempty"`
	// A list of role identifiers. Only the author and users associated with at least one of these roles can edit this dashboard.
	RestrictedRoles []string `json:"restricted_roles,omitempty"`
	// Array of template variables saved views.
	TemplateVariablePresets []DashboardTemplateVariablePreset `json:"template_variable_presets,omitempty"`
	// List of template variables for this dashboard.
	TemplateVariables []DashboardTemplateVariable `json:"template_variables,omitempty"`
	// Title of the dashboard.
	Title string `json:"title"`
	// The URL of the dashboard.
	Url *string `json:"url,omitempty"`
	// List of widgets to display on the dashboard.
	Widgets []Widget `json:"widgets"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboard instantiates a new Dashboard object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboard(layoutType DashboardLayoutType, title string, widgets []Widget) *Dashboard {
	this := Dashboard{}
	var isReadOnly bool = false
	this.IsReadOnly = &isReadOnly
	this.LayoutType = layoutType
	this.Title = title
	this.Widgets = widgets
	return &this
}

// NewDashboardWithDefaults instantiates a new Dashboard object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardWithDefaults() *Dashboard {
	this := Dashboard{}
	var isReadOnly bool = false
	this.IsReadOnly = &isReadOnly
	return &this
}

// GetAuthorHandle returns the AuthorHandle field value if set, zero value otherwise.
func (o *Dashboard) GetAuthorHandle() string {
	if o == nil || o.AuthorHandle == nil {
		var ret string
		return ret
	}
	return *o.AuthorHandle
}

// GetAuthorHandleOk returns a tuple with the AuthorHandle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetAuthorHandleOk() (*string, bool) {
	if o == nil || o.AuthorHandle == nil {
		return nil, false
	}
	return o.AuthorHandle, true
}

// HasAuthorHandle returns a boolean if a field has been set.
func (o *Dashboard) HasAuthorHandle() bool {
	if o != nil && o.AuthorHandle != nil {
		return true
	}

	return false
}

// SetAuthorHandle gets a reference to the given string and assigns it to the AuthorHandle field.
func (o *Dashboard) SetAuthorHandle(v string) {
	o.AuthorHandle = &v
}

// GetAuthorName returns the AuthorName field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Dashboard) GetAuthorName() string {
	if o == nil || o.AuthorName.Get() == nil {
		var ret string
		return ret
	}
	return *o.AuthorName.Get()
}

// GetAuthorNameOk returns a tuple with the AuthorName field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Dashboard) GetAuthorNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.AuthorName.Get(), o.AuthorName.IsSet()
}

// HasAuthorName returns a boolean if a field has been set.
func (o *Dashboard) HasAuthorName() bool {
	if o != nil && o.AuthorName.IsSet() {
		return true
	}

	return false
}

// SetAuthorName gets a reference to the given NullableString and assigns it to the AuthorName field.
func (o *Dashboard) SetAuthorName(v string) {
	o.AuthorName.Set(&v)
}

// SetAuthorNameNil sets the value for AuthorName to be an explicit nil.
func (o *Dashboard) SetAuthorNameNil() {
	o.AuthorName.Set(nil)
}

// UnsetAuthorName ensures that no value is present for AuthorName, not even an explicit nil.
func (o *Dashboard) UnsetAuthorName() {
	o.AuthorName.Unset()
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *Dashboard) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *Dashboard) HasCreatedAt() bool {
	if o != nil && o.CreatedAt != nil {
		return true
	}

	return false
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *Dashboard) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetDescription returns the Description field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Dashboard) GetDescription() string {
	if o == nil || o.Description.Get() == nil {
		var ret string
		return ret
	}
	return *o.Description.Get()
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Dashboard) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Description.Get(), o.Description.IsSet()
}

// HasDescription returns a boolean if a field has been set.
func (o *Dashboard) HasDescription() bool {
	if o != nil && o.Description.IsSet() {
		return true
	}

	return false
}

// SetDescription gets a reference to the given NullableString and assigns it to the Description field.
func (o *Dashboard) SetDescription(v string) {
	o.Description.Set(&v)
}

// SetDescriptionNil sets the value for Description to be an explicit nil.
func (o *Dashboard) SetDescriptionNil() {
	o.Description.Set(nil)
}

// UnsetDescription ensures that no value is present for Description, not even an explicit nil.
func (o *Dashboard) UnsetDescription() {
	o.Description.Unset()
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *Dashboard) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *Dashboard) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *Dashboard) SetId(v string) {
	o.Id = &v
}

// GetIsReadOnly returns the IsReadOnly field value if set, zero value otherwise.
// Deprecated
func (o *Dashboard) GetIsReadOnly() bool {
	if o == nil || o.IsReadOnly == nil {
		var ret bool
		return ret
	}
	return *o.IsReadOnly
}

// GetIsReadOnlyOk returns a tuple with the IsReadOnly field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *Dashboard) GetIsReadOnlyOk() (*bool, bool) {
	if o == nil || o.IsReadOnly == nil {
		return nil, false
	}
	return o.IsReadOnly, true
}

// HasIsReadOnly returns a boolean if a field has been set.
func (o *Dashboard) HasIsReadOnly() bool {
	if o != nil && o.IsReadOnly != nil {
		return true
	}

	return false
}

// SetIsReadOnly gets a reference to the given bool and assigns it to the IsReadOnly field.
// Deprecated
func (o *Dashboard) SetIsReadOnly(v bool) {
	o.IsReadOnly = &v
}

// GetLayoutType returns the LayoutType field value.
func (o *Dashboard) GetLayoutType() DashboardLayoutType {
	if o == nil {
		var ret DashboardLayoutType
		return ret
	}
	return o.LayoutType
}

// GetLayoutTypeOk returns a tuple with the LayoutType field value
// and a boolean to check if the value has been set.
func (o *Dashboard) GetLayoutTypeOk() (*DashboardLayoutType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LayoutType, true
}

// SetLayoutType sets field value.
func (o *Dashboard) SetLayoutType(v DashboardLayoutType) {
	o.LayoutType = v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *Dashboard) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *Dashboard) HasModifiedAt() bool {
	if o != nil && o.ModifiedAt != nil {
		return true
	}

	return false
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *Dashboard) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetNotifyList returns the NotifyList field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Dashboard) GetNotifyList() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.NotifyList
}

// GetNotifyListOk returns a tuple with the NotifyList field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Dashboard) GetNotifyListOk() (*[]string, bool) {
	if o == nil || o.NotifyList == nil {
		return nil, false
	}
	return &o.NotifyList, true
}

// HasNotifyList returns a boolean if a field has been set.
func (o *Dashboard) HasNotifyList() bool {
	if o != nil && o.NotifyList != nil {
		return true
	}

	return false
}

// SetNotifyList gets a reference to the given []string and assigns it to the NotifyList field.
func (o *Dashboard) SetNotifyList(v []string) {
	o.NotifyList = v
}

// GetReflowType returns the ReflowType field value if set, zero value otherwise.
func (o *Dashboard) GetReflowType() DashboardReflowType {
	if o == nil || o.ReflowType == nil {
		var ret DashboardReflowType
		return ret
	}
	return *o.ReflowType
}

// GetReflowTypeOk returns a tuple with the ReflowType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetReflowTypeOk() (*DashboardReflowType, bool) {
	if o == nil || o.ReflowType == nil {
		return nil, false
	}
	return o.ReflowType, true
}

// HasReflowType returns a boolean if a field has been set.
func (o *Dashboard) HasReflowType() bool {
	if o != nil && o.ReflowType != nil {
		return true
	}

	return false
}

// SetReflowType gets a reference to the given DashboardReflowType and assigns it to the ReflowType field.
func (o *Dashboard) SetReflowType(v DashboardReflowType) {
	o.ReflowType = &v
}

// GetRestrictedRoles returns the RestrictedRoles field value if set, zero value otherwise.
func (o *Dashboard) GetRestrictedRoles() []string {
	if o == nil || o.RestrictedRoles == nil {
		var ret []string
		return ret
	}
	return o.RestrictedRoles
}

// GetRestrictedRolesOk returns a tuple with the RestrictedRoles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetRestrictedRolesOk() (*[]string, bool) {
	if o == nil || o.RestrictedRoles == nil {
		return nil, false
	}
	return &o.RestrictedRoles, true
}

// HasRestrictedRoles returns a boolean if a field has been set.
func (o *Dashboard) HasRestrictedRoles() bool {
	if o != nil && o.RestrictedRoles != nil {
		return true
	}

	return false
}

// SetRestrictedRoles gets a reference to the given []string and assigns it to the RestrictedRoles field.
func (o *Dashboard) SetRestrictedRoles(v []string) {
	o.RestrictedRoles = v
}

// GetTemplateVariablePresets returns the TemplateVariablePresets field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Dashboard) GetTemplateVariablePresets() []DashboardTemplateVariablePreset {
	if o == nil {
		var ret []DashboardTemplateVariablePreset
		return ret
	}
	return o.TemplateVariablePresets
}

// GetTemplateVariablePresetsOk returns a tuple with the TemplateVariablePresets field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Dashboard) GetTemplateVariablePresetsOk() (*[]DashboardTemplateVariablePreset, bool) {
	if o == nil || o.TemplateVariablePresets == nil {
		return nil, false
	}
	return &o.TemplateVariablePresets, true
}

// HasTemplateVariablePresets returns a boolean if a field has been set.
func (o *Dashboard) HasTemplateVariablePresets() bool {
	if o != nil && o.TemplateVariablePresets != nil {
		return true
	}

	return false
}

// SetTemplateVariablePresets gets a reference to the given []DashboardTemplateVariablePreset and assigns it to the TemplateVariablePresets field.
func (o *Dashboard) SetTemplateVariablePresets(v []DashboardTemplateVariablePreset) {
	o.TemplateVariablePresets = v
}

// GetTemplateVariables returns the TemplateVariables field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Dashboard) GetTemplateVariables() []DashboardTemplateVariable {
	if o == nil {
		var ret []DashboardTemplateVariable
		return ret
	}
	return o.TemplateVariables
}

// GetTemplateVariablesOk returns a tuple with the TemplateVariables field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Dashboard) GetTemplateVariablesOk() (*[]DashboardTemplateVariable, bool) {
	if o == nil || o.TemplateVariables == nil {
		return nil, false
	}
	return &o.TemplateVariables, true
}

// HasTemplateVariables returns a boolean if a field has been set.
func (o *Dashboard) HasTemplateVariables() bool {
	if o != nil && o.TemplateVariables != nil {
		return true
	}

	return false
}

// SetTemplateVariables gets a reference to the given []DashboardTemplateVariable and assigns it to the TemplateVariables field.
func (o *Dashboard) SetTemplateVariables(v []DashboardTemplateVariable) {
	o.TemplateVariables = v
}

// GetTitle returns the Title field value.
func (o *Dashboard) GetTitle() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Title
}

// GetTitleOk returns a tuple with the Title field value
// and a boolean to check if the value has been set.
func (o *Dashboard) GetTitleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Title, true
}

// SetTitle sets field value.
func (o *Dashboard) SetTitle(v string) {
	o.Title = v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *Dashboard) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Dashboard) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *Dashboard) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *Dashboard) SetUrl(v string) {
	o.Url = &v
}

// GetWidgets returns the Widgets field value.
func (o *Dashboard) GetWidgets() []Widget {
	if o == nil {
		var ret []Widget
		return ret
	}
	return o.Widgets
}

// GetWidgetsOk returns a tuple with the Widgets field value
// and a boolean to check if the value has been set.
func (o *Dashboard) GetWidgetsOk() (*[]Widget, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Widgets, true
}

// SetWidgets sets field value.
func (o *Dashboard) SetWidgets(v []Widget) {
	o.Widgets = v
}

// MarshalJSON serializes the struct using spec logic.
func (o Dashboard) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AuthorHandle != nil {
		toSerialize["author_handle"] = o.AuthorHandle
	}
	if o.AuthorName.IsSet() {
		toSerialize["author_name"] = o.AuthorName.Get()
	}
	if o.CreatedAt != nil {
		if o.CreatedAt.Nanosecond() == 0 {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Description.IsSet() {
		toSerialize["description"] = o.Description.Get()
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IsReadOnly != nil {
		toSerialize["is_read_only"] = o.IsReadOnly
	}
	toSerialize["layout_type"] = o.LayoutType
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.NotifyList != nil {
		toSerialize["notify_list"] = o.NotifyList
	}
	if o.ReflowType != nil {
		toSerialize["reflow_type"] = o.ReflowType
	}
	if o.RestrictedRoles != nil {
		toSerialize["restricted_roles"] = o.RestrictedRoles
	}
	if o.TemplateVariablePresets != nil {
		toSerialize["template_variable_presets"] = o.TemplateVariablePresets
	}
	if o.TemplateVariables != nil {
		toSerialize["template_variables"] = o.TemplateVariables
	}
	toSerialize["title"] = o.Title
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}
	toSerialize["widgets"] = o.Widgets

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *Dashboard) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		LayoutType *DashboardLayoutType `json:"layout_type"`
		Title      *string              `json:"title"`
		Widgets    *[]Widget            `json:"widgets"`
	}{}
	all := struct {
		AuthorHandle            *string                           `json:"author_handle,omitempty"`
		AuthorName              NullableString                    `json:"author_name,omitempty"`
		CreatedAt               *time.Time                        `json:"created_at,omitempty"`
		Description             NullableString                    `json:"description,omitempty"`
		Id                      *string                           `json:"id,omitempty"`
		IsReadOnly              *bool                             `json:"is_read_only,omitempty"`
		LayoutType              DashboardLayoutType               `json:"layout_type"`
		ModifiedAt              *time.Time                        `json:"modified_at,omitempty"`
		NotifyList              []string                          `json:"notify_list,omitempty"`
		ReflowType              *DashboardReflowType              `json:"reflow_type,omitempty"`
		RestrictedRoles         []string                          `json:"restricted_roles,omitempty"`
		TemplateVariablePresets []DashboardTemplateVariablePreset `json:"template_variable_presets,omitempty"`
		TemplateVariables       []DashboardTemplateVariable       `json:"template_variables,omitempty"`
		Title                   string                            `json:"title"`
		Url                     *string                           `json:"url,omitempty"`
		Widgets                 []Widget                          `json:"widgets"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.LayoutType == nil {
		return fmt.Errorf("Required field layout_type missing")
	}
	if required.Title == nil {
		return fmt.Errorf("Required field title missing")
	}
	if required.Widgets == nil {
		return fmt.Errorf("Required field widgets missing")
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
	if v := all.LayoutType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.ReflowType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AuthorHandle = all.AuthorHandle
	o.AuthorName = all.AuthorName
	o.CreatedAt = all.CreatedAt
	o.Description = all.Description
	o.Id = all.Id
	o.IsReadOnly = all.IsReadOnly
	o.LayoutType = all.LayoutType
	o.ModifiedAt = all.ModifiedAt
	o.NotifyList = all.NotifyList
	o.ReflowType = all.ReflowType
	o.RestrictedRoles = all.RestrictedRoles
	o.TemplateVariablePresets = all.TemplateVariablePresets
	o.TemplateVariables = all.TemplateVariables
	o.Title = all.Title
	o.Url = all.Url
	o.Widgets = all.Widgets
	return nil
}

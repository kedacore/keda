// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// DashboardSummaryDefinition Dashboard definition.
type DashboardSummaryDefinition struct {
	// Identifier of the dashboard author.
	AuthorHandle *string `json:"author_handle,omitempty"`
	// Creation date of the dashboard.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Description of the dashboard.
	Description NullableString `json:"description,omitempty"`
	// Dashboard identifier.
	Id *string `json:"id,omitempty"`
	// Whether this dashboard is read-only. If True, only the author and admins can make changes to it.
	IsReadOnly *bool `json:"is_read_only,omitempty"`
	// Layout type of the dashboard.
	LayoutType *DashboardLayoutType `json:"layout_type,omitempty"`
	// Modification date of the dashboard.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// Title of the dashboard.
	Title *string `json:"title,omitempty"`
	// URL of the dashboard.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardSummaryDefinition instantiates a new DashboardSummaryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardSummaryDefinition() *DashboardSummaryDefinition {
	this := DashboardSummaryDefinition{}
	return &this
}

// NewDashboardSummaryDefinitionWithDefaults instantiates a new DashboardSummaryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardSummaryDefinitionWithDefaults() *DashboardSummaryDefinition {
	this := DashboardSummaryDefinition{}
	return &this
}

// GetAuthorHandle returns the AuthorHandle field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetAuthorHandle() string {
	if o == nil || o.AuthorHandle == nil {
		var ret string
		return ret
	}
	return *o.AuthorHandle
}

// GetAuthorHandleOk returns a tuple with the AuthorHandle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetAuthorHandleOk() (*string, bool) {
	if o == nil || o.AuthorHandle == nil {
		return nil, false
	}
	return o.AuthorHandle, true
}

// HasAuthorHandle returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasAuthorHandle() bool {
	if o != nil && o.AuthorHandle != nil {
		return true
	}

	return false
}

// SetAuthorHandle gets a reference to the given string and assigns it to the AuthorHandle field.
func (o *DashboardSummaryDefinition) SetAuthorHandle(v string) {
	o.AuthorHandle = &v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasCreatedAt() bool {
	if o != nil && o.CreatedAt != nil {
		return true
	}

	return false
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *DashboardSummaryDefinition) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetDescription returns the Description field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DashboardSummaryDefinition) GetDescription() string {
	if o == nil || o.Description.Get() == nil {
		var ret string
		return ret
	}
	return *o.Description.Get()
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DashboardSummaryDefinition) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Description.Get(), o.Description.IsSet()
}

// HasDescription returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasDescription() bool {
	if o != nil && o.Description.IsSet() {
		return true
	}

	return false
}

// SetDescription gets a reference to the given NullableString and assigns it to the Description field.
func (o *DashboardSummaryDefinition) SetDescription(v string) {
	o.Description.Set(&v)
}

// SetDescriptionNil sets the value for Description to be an explicit nil.
func (o *DashboardSummaryDefinition) SetDescriptionNil() {
	o.Description.Set(nil)
}

// UnsetDescription ensures that no value is present for Description, not even an explicit nil.
func (o *DashboardSummaryDefinition) UnsetDescription() {
	o.Description.Unset()
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *DashboardSummaryDefinition) SetId(v string) {
	o.Id = &v
}

// GetIsReadOnly returns the IsReadOnly field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetIsReadOnly() bool {
	if o == nil || o.IsReadOnly == nil {
		var ret bool
		return ret
	}
	return *o.IsReadOnly
}

// GetIsReadOnlyOk returns a tuple with the IsReadOnly field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetIsReadOnlyOk() (*bool, bool) {
	if o == nil || o.IsReadOnly == nil {
		return nil, false
	}
	return o.IsReadOnly, true
}

// HasIsReadOnly returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasIsReadOnly() bool {
	if o != nil && o.IsReadOnly != nil {
		return true
	}

	return false
}

// SetIsReadOnly gets a reference to the given bool and assigns it to the IsReadOnly field.
func (o *DashboardSummaryDefinition) SetIsReadOnly(v bool) {
	o.IsReadOnly = &v
}

// GetLayoutType returns the LayoutType field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetLayoutType() DashboardLayoutType {
	if o == nil || o.LayoutType == nil {
		var ret DashboardLayoutType
		return ret
	}
	return *o.LayoutType
}

// GetLayoutTypeOk returns a tuple with the LayoutType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetLayoutTypeOk() (*DashboardLayoutType, bool) {
	if o == nil || o.LayoutType == nil {
		return nil, false
	}
	return o.LayoutType, true
}

// HasLayoutType returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasLayoutType() bool {
	if o != nil && o.LayoutType != nil {
		return true
	}

	return false
}

// SetLayoutType gets a reference to the given DashboardLayoutType and assigns it to the LayoutType field.
func (o *DashboardSummaryDefinition) SetLayoutType(v DashboardLayoutType) {
	o.LayoutType = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasModifiedAt() bool {
	if o != nil && o.ModifiedAt != nil {
		return true
	}

	return false
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *DashboardSummaryDefinition) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *DashboardSummaryDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *DashboardSummaryDefinition) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummaryDefinition) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *DashboardSummaryDefinition) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *DashboardSummaryDefinition) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardSummaryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AuthorHandle != nil {
		toSerialize["author_handle"] = o.AuthorHandle
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
	if o.LayoutType != nil {
		toSerialize["layout_type"] = o.LayoutType
	}
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardSummaryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AuthorHandle *string              `json:"author_handle,omitempty"`
		CreatedAt    *time.Time           `json:"created_at,omitempty"`
		Description  NullableString       `json:"description,omitempty"`
		Id           *string              `json:"id,omitempty"`
		IsReadOnly   *bool                `json:"is_read_only,omitempty"`
		LayoutType   *DashboardLayoutType `json:"layout_type,omitempty"`
		ModifiedAt   *time.Time           `json:"modified_at,omitempty"`
		Title        *string              `json:"title,omitempty"`
		Url          *string              `json:"url,omitempty"`
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
	if v := all.LayoutType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AuthorHandle = all.AuthorHandle
	o.CreatedAt = all.CreatedAt
	o.Description = all.Description
	o.Id = all.Id
	o.IsReadOnly = all.IsReadOnly
	o.LayoutType = all.LayoutType
	o.ModifiedAt = all.ModifiedAt
	o.Title = all.Title
	o.Url = all.Url
	return nil
}

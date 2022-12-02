// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"time"
)

// DashboardList Your Datadog Dashboards.
type DashboardList struct {
	// Object describing the creator of the shared element.
	Author *Creator `json:"author,omitempty"`
	// Date of creation of the dashboard list.
	Created *time.Time `json:"created,omitempty"`
	// The number of dashboards in the list.
	DashboardCount *int64 `json:"dashboard_count,omitempty"`
	// The ID of the dashboard list.
	Id *int64 `json:"id,omitempty"`
	// Whether or not the list is in the favorites.
	IsFavorite *bool `json:"is_favorite,omitempty"`
	// Date of last edition of the dashboard list.
	Modified *time.Time `json:"modified,omitempty"`
	// The name of the dashboard list.
	Name string `json:"name"`
	// The type of dashboard list.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardList instantiates a new DashboardList object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardList(name string) *DashboardList {
	this := DashboardList{}
	this.Name = name
	return &this
}

// NewDashboardListWithDefaults instantiates a new DashboardList object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardListWithDefaults() *DashboardList {
	this := DashboardList{}
	return &this
}

// GetAuthor returns the Author field value if set, zero value otherwise.
func (o *DashboardList) GetAuthor() Creator {
	if o == nil || o.Author == nil {
		var ret Creator
		return ret
	}
	return *o.Author
}

// GetAuthorOk returns a tuple with the Author field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetAuthorOk() (*Creator, bool) {
	if o == nil || o.Author == nil {
		return nil, false
	}
	return o.Author, true
}

// HasAuthor returns a boolean if a field has been set.
func (o *DashboardList) HasAuthor() bool {
	if o != nil && o.Author != nil {
		return true
	}

	return false
}

// SetAuthor gets a reference to the given Creator and assigns it to the Author field.
func (o *DashboardList) SetAuthor(v Creator) {
	o.Author = &v
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *DashboardList) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *DashboardList) HasCreated() bool {
	if o != nil && o.Created != nil {
		return true
	}

	return false
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *DashboardList) SetCreated(v time.Time) {
	o.Created = &v
}

// GetDashboardCount returns the DashboardCount field value if set, zero value otherwise.
func (o *DashboardList) GetDashboardCount() int64 {
	if o == nil || o.DashboardCount == nil {
		var ret int64
		return ret
	}
	return *o.DashboardCount
}

// GetDashboardCountOk returns a tuple with the DashboardCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetDashboardCountOk() (*int64, bool) {
	if o == nil || o.DashboardCount == nil {
		return nil, false
	}
	return o.DashboardCount, true
}

// HasDashboardCount returns a boolean if a field has been set.
func (o *DashboardList) HasDashboardCount() bool {
	if o != nil && o.DashboardCount != nil {
		return true
	}

	return false
}

// SetDashboardCount gets a reference to the given int64 and assigns it to the DashboardCount field.
func (o *DashboardList) SetDashboardCount(v int64) {
	o.DashboardCount = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *DashboardList) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *DashboardList) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *DashboardList) SetId(v int64) {
	o.Id = &v
}

// GetIsFavorite returns the IsFavorite field value if set, zero value otherwise.
func (o *DashboardList) GetIsFavorite() bool {
	if o == nil || o.IsFavorite == nil {
		var ret bool
		return ret
	}
	return *o.IsFavorite
}

// GetIsFavoriteOk returns a tuple with the IsFavorite field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetIsFavoriteOk() (*bool, bool) {
	if o == nil || o.IsFavorite == nil {
		return nil, false
	}
	return o.IsFavorite, true
}

// HasIsFavorite returns a boolean if a field has been set.
func (o *DashboardList) HasIsFavorite() bool {
	if o != nil && o.IsFavorite != nil {
		return true
	}

	return false
}

// SetIsFavorite gets a reference to the given bool and assigns it to the IsFavorite field.
func (o *DashboardList) SetIsFavorite(v bool) {
	o.IsFavorite = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *DashboardList) GetModified() time.Time {
	if o == nil || o.Modified == nil {
		var ret time.Time
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetModifiedOk() (*time.Time, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *DashboardList) HasModified() bool {
	if o != nil && o.Modified != nil {
		return true
	}

	return false
}

// SetModified gets a reference to the given time.Time and assigns it to the Modified field.
func (o *DashboardList) SetModified(v time.Time) {
	o.Modified = &v
}

// GetName returns the Name field value.
func (o *DashboardList) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *DashboardList) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *DashboardList) SetName(v string) {
	o.Name = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *DashboardList) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardList) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *DashboardList) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *DashboardList) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardList) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Author != nil {
		toSerialize["author"] = o.Author
	}
	if o.Created != nil {
		if o.Created.Nanosecond() == 0 {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.DashboardCount != nil {
		toSerialize["dashboard_count"] = o.DashboardCount
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IsFavorite != nil {
		toSerialize["is_favorite"] = o.IsFavorite
	}
	if o.Modified != nil {
		if o.Modified.Nanosecond() == 0 {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	toSerialize["name"] = o.Name
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardList) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Author         *Creator   `json:"author,omitempty"`
		Created        *time.Time `json:"created,omitempty"`
		DashboardCount *int64     `json:"dashboard_count,omitempty"`
		Id             *int64     `json:"id,omitempty"`
		IsFavorite     *bool      `json:"is_favorite,omitempty"`
		Modified       *time.Time `json:"modified,omitempty"`
		Name           string     `json:"name"`
		Type           *string    `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if all.Author != nil && all.Author.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Author = all.Author
	o.Created = all.Created
	o.DashboardCount = all.DashboardCount
	o.Id = all.Id
	o.IsFavorite = all.IsFavorite
	o.Modified = all.Modified
	o.Name = all.Name
	o.Type = all.Type
	return nil
}

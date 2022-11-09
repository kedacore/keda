// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// MonitorUpdateRequest Object describing a monitor update request.
type MonitorUpdateRequest struct {
	// Timestamp of the monitor creation.
	Created *time.Time `json:"created,omitempty"`
	// Object describing the creator of the shared element.
	Creator *Creator `json:"creator,omitempty"`
	// Whether or not the monitor is deleted. (Always `null`)
	Deleted NullableTime `json:"deleted,omitempty"`
	// ID of this monitor.
	Id *int64 `json:"id,omitempty"`
	// A message to include with notifications for this monitor.
	Message *string `json:"message,omitempty"`
	// Last timestamp when the monitor was edited.
	Modified *time.Time `json:"modified,omitempty"`
	// Whether or not the monitor is broken down on different groups.
	Multi *bool `json:"multi,omitempty"`
	// The monitor name.
	Name *string `json:"name,omitempty"`
	// List of options associated with your monitor.
	Options *MonitorOptions `json:"options,omitempty"`
	// The different states your monitor can be in.
	OverallState *MonitorOverallStates `json:"overall_state,omitempty"`
	// Integer from 1 (high) to 5 (low) indicating alert severity.
	Priority *int64 `json:"priority,omitempty"`
	// The monitor query.
	Query *string `json:"query,omitempty"`
	// A list of unique role identifiers to define which roles are allowed to edit the monitor. The unique identifiers for all roles can be pulled from the [Roles API](https://docs.datadoghq.com/api/latest/roles/#list-roles) and are located in the `data.id` field. Editing a monitor includes any updates to the monitor configuration, monitor deletion, and muting of the monitor for any amount of time. `restricted_roles` is the successor of `locked`. For more information about `locked` and `restricted_roles`, see the [monitor options docs](https://docs.datadoghq.com/monitors/guide/monitor_api_options/#permissions-options).
	RestrictedRoles []string `json:"restricted_roles,omitempty"`
	// Wrapper object with the different monitor states.
	State *MonitorState `json:"state,omitempty"`
	// Tags associated to your monitor.
	Tags []string `json:"tags,omitempty"`
	// The type of the monitor. For more information about `type`, see the [monitor options](https://docs.datadoghq.com/monitors/guide/monitor_api_options/) docs.
	Type *MonitorType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorUpdateRequest instantiates a new MonitorUpdateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorUpdateRequest() *MonitorUpdateRequest {
	this := MonitorUpdateRequest{}
	return &this
}

// NewMonitorUpdateRequestWithDefaults instantiates a new MonitorUpdateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorUpdateRequestWithDefaults() *MonitorUpdateRequest {
	this := MonitorUpdateRequest{}
	return &this
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasCreated() bool {
	if o != nil && o.Created != nil {
		return true
	}

	return false
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *MonitorUpdateRequest) SetCreated(v time.Time) {
	o.Created = &v
}

// GetCreator returns the Creator field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetCreator() Creator {
	if o == nil || o.Creator == nil {
		var ret Creator
		return ret
	}
	return *o.Creator
}

// GetCreatorOk returns a tuple with the Creator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetCreatorOk() (*Creator, bool) {
	if o == nil || o.Creator == nil {
		return nil, false
	}
	return o.Creator, true
}

// HasCreator returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasCreator() bool {
	if o != nil && o.Creator != nil {
		return true
	}

	return false
}

// SetCreator gets a reference to the given Creator and assigns it to the Creator field.
func (o *MonitorUpdateRequest) SetCreator(v Creator) {
	o.Creator = &v
}

// GetDeleted returns the Deleted field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorUpdateRequest) GetDeleted() time.Time {
	if o == nil || o.Deleted.Get() == nil {
		var ret time.Time
		return ret
	}
	return *o.Deleted.Get()
}

// GetDeletedOk returns a tuple with the Deleted field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorUpdateRequest) GetDeletedOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return o.Deleted.Get(), o.Deleted.IsSet()
}

// HasDeleted returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasDeleted() bool {
	if o != nil && o.Deleted.IsSet() {
		return true
	}

	return false
}

// SetDeleted gets a reference to the given NullableTime and assigns it to the Deleted field.
func (o *MonitorUpdateRequest) SetDeleted(v time.Time) {
	o.Deleted.Set(&v)
}

// SetDeletedNil sets the value for Deleted to be an explicit nil.
func (o *MonitorUpdateRequest) SetDeletedNil() {
	o.Deleted.Set(nil)
}

// UnsetDeleted ensures that no value is present for Deleted, not even an explicit nil.
func (o *MonitorUpdateRequest) UnsetDeleted() {
	o.Deleted.Unset()
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *MonitorUpdateRequest) SetId(v int64) {
	o.Id = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *MonitorUpdateRequest) SetMessage(v string) {
	o.Message = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetModified() time.Time {
	if o == nil || o.Modified == nil {
		var ret time.Time
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetModifiedOk() (*time.Time, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasModified() bool {
	if o != nil && o.Modified != nil {
		return true
	}

	return false
}

// SetModified gets a reference to the given time.Time and assigns it to the Modified field.
func (o *MonitorUpdateRequest) SetModified(v time.Time) {
	o.Modified = &v
}

// GetMulti returns the Multi field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetMulti() bool {
	if o == nil || o.Multi == nil {
		var ret bool
		return ret
	}
	return *o.Multi
}

// GetMultiOk returns a tuple with the Multi field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetMultiOk() (*bool, bool) {
	if o == nil || o.Multi == nil {
		return nil, false
	}
	return o.Multi, true
}

// HasMulti returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasMulti() bool {
	if o != nil && o.Multi != nil {
		return true
	}

	return false
}

// SetMulti gets a reference to the given bool and assigns it to the Multi field.
func (o *MonitorUpdateRequest) SetMulti(v bool) {
	o.Multi = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *MonitorUpdateRequest) SetName(v string) {
	o.Name = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetOptions() MonitorOptions {
	if o == nil || o.Options == nil {
		var ret MonitorOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetOptionsOk() (*MonitorOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasOptions() bool {
	if o != nil && o.Options != nil {
		return true
	}

	return false
}

// SetOptions gets a reference to the given MonitorOptions and assigns it to the Options field.
func (o *MonitorUpdateRequest) SetOptions(v MonitorOptions) {
	o.Options = &v
}

// GetOverallState returns the OverallState field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetOverallState() MonitorOverallStates {
	if o == nil || o.OverallState == nil {
		var ret MonitorOverallStates
		return ret
	}
	return *o.OverallState
}

// GetOverallStateOk returns a tuple with the OverallState field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetOverallStateOk() (*MonitorOverallStates, bool) {
	if o == nil || o.OverallState == nil {
		return nil, false
	}
	return o.OverallState, true
}

// HasOverallState returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasOverallState() bool {
	if o != nil && o.OverallState != nil {
		return true
	}

	return false
}

// SetOverallState gets a reference to the given MonitorOverallStates and assigns it to the OverallState field.
func (o *MonitorUpdateRequest) SetOverallState(v MonitorOverallStates) {
	o.OverallState = &v
}

// GetPriority returns the Priority field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetPriority() int64 {
	if o == nil || o.Priority == nil {
		var ret int64
		return ret
	}
	return *o.Priority
}

// GetPriorityOk returns a tuple with the Priority field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetPriorityOk() (*int64, bool) {
	if o == nil || o.Priority == nil {
		return nil, false
	}
	return o.Priority, true
}

// HasPriority returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasPriority() bool {
	if o != nil && o.Priority != nil {
		return true
	}

	return false
}

// SetPriority gets a reference to the given int64 and assigns it to the Priority field.
func (o *MonitorUpdateRequest) SetPriority(v int64) {
	o.Priority = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *MonitorUpdateRequest) SetQuery(v string) {
	o.Query = &v
}

// GetRestrictedRoles returns the RestrictedRoles field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetRestrictedRoles() []string {
	if o == nil || o.RestrictedRoles == nil {
		var ret []string
		return ret
	}
	return o.RestrictedRoles
}

// GetRestrictedRolesOk returns a tuple with the RestrictedRoles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetRestrictedRolesOk() (*[]string, bool) {
	if o == nil || o.RestrictedRoles == nil {
		return nil, false
	}
	return &o.RestrictedRoles, true
}

// HasRestrictedRoles returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasRestrictedRoles() bool {
	if o != nil && o.RestrictedRoles != nil {
		return true
	}

	return false
}

// SetRestrictedRoles gets a reference to the given []string and assigns it to the RestrictedRoles field.
func (o *MonitorUpdateRequest) SetRestrictedRoles(v []string) {
	o.RestrictedRoles = v
}

// GetState returns the State field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetState() MonitorState {
	if o == nil || o.State == nil {
		var ret MonitorState
		return ret
	}
	return *o.State
}

// GetStateOk returns a tuple with the State field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetStateOk() (*MonitorState, bool) {
	if o == nil || o.State == nil {
		return nil, false
	}
	return o.State, true
}

// HasState returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasState() bool {
	if o != nil && o.State != nil {
		return true
	}

	return false
}

// SetState gets a reference to the given MonitorState and assigns it to the State field.
func (o *MonitorUpdateRequest) SetState(v MonitorState) {
	o.State = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *MonitorUpdateRequest) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MonitorUpdateRequest) GetType() MonitorType {
	if o == nil || o.Type == nil {
		var ret MonitorType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorUpdateRequest) GetTypeOk() (*MonitorType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MonitorUpdateRequest) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given MonitorType and assigns it to the Type field.
func (o *MonitorUpdateRequest) SetType(v MonitorType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorUpdateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Created != nil {
		if o.Created.Nanosecond() == 0 {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Creator != nil {
		toSerialize["creator"] = o.Creator
	}
	if o.Deleted.IsSet() {
		toSerialize["deleted"] = o.Deleted.Get()
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Modified != nil {
		if o.Modified.Nanosecond() == 0 {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Multi != nil {
		toSerialize["multi"] = o.Multi
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Options != nil {
		toSerialize["options"] = o.Options
	}
	if o.OverallState != nil {
		toSerialize["overall_state"] = o.OverallState
	}
	if o.Priority != nil {
		toSerialize["priority"] = o.Priority
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.RestrictedRoles != nil {
		toSerialize["restricted_roles"] = o.RestrictedRoles
	}
	if o.State != nil {
		toSerialize["state"] = o.State
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorUpdateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Created         *time.Time            `json:"created,omitempty"`
		Creator         *Creator              `json:"creator,omitempty"`
		Deleted         NullableTime          `json:"deleted,omitempty"`
		Id              *int64                `json:"id,omitempty"`
		Message         *string               `json:"message,omitempty"`
		Modified        *time.Time            `json:"modified,omitempty"`
		Multi           *bool                 `json:"multi,omitempty"`
		Name            *string               `json:"name,omitempty"`
		Options         *MonitorOptions       `json:"options,omitempty"`
		OverallState    *MonitorOverallStates `json:"overall_state,omitempty"`
		Priority        *int64                `json:"priority,omitempty"`
		Query           *string               `json:"query,omitempty"`
		RestrictedRoles []string              `json:"restricted_roles,omitempty"`
		State           *MonitorState         `json:"state,omitempty"`
		Tags            []string              `json:"tags,omitempty"`
		Type            *MonitorType          `json:"type,omitempty"`
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
	if v := all.OverallState; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Created = all.Created
	if all.Creator != nil && all.Creator.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Creator = all.Creator
	o.Deleted = all.Deleted
	o.Id = all.Id
	o.Message = all.Message
	o.Modified = all.Modified
	o.Multi = all.Multi
	o.Name = all.Name
	if all.Options != nil && all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	o.OverallState = all.OverallState
	o.Priority = all.Priority
	o.Query = all.Query
	o.RestrictedRoles = all.RestrictedRoles
	if all.State != nil && all.State.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.State = all.State
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}

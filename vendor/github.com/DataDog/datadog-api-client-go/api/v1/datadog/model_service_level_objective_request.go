// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ServiceLevelObjectiveRequest A service level objective object includes a service level indicator, thresholds
// for one or more timeframes, and metadata (`name`, `description`, `tags`, etc.).
type ServiceLevelObjectiveRequest struct {
	// A user-defined description of the service level objective.
	//
	// Always included in service level objective responses (but may be `null`).
	// Optional in create/update requests.
	Description NullableString `json:"description,omitempty"`
	// A list of (up to 100) monitor groups that narrow the scope of a monitor service level objective.
	//
	// Included in service level objective responses if it is not empty. Optional in
	// create/update requests for monitor service level objectives, but may only be
	// used when then length of the `monitor_ids` field is one.
	Groups []string `json:"groups,omitempty"`
	// A list of monitor IDs that defines the scope of a monitor service level
	// objective. **Required if type is `monitor`**.
	MonitorIds []int64 `json:"monitor_ids,omitempty"`
	// The name of the service level objective object.
	Name string `json:"name"`
	// A metric-based SLO. **Required if type is `metric`**. Note that Datadog only allows the sum by aggregator
	// to be used because this will sum up all request counts instead of averaging them, or taking the max or
	// min of all of those requests.
	Query *ServiceLevelObjectiveQuery `json:"query,omitempty"`
	// A list of tags associated with this service level objective.
	// Always included in service level objective responses (but may be empty).
	// Optional in create/update requests.
	Tags []string `json:"tags,omitempty"`
	// The thresholds (timeframes and associated targets) for this service level
	// objective object.
	Thresholds []SLOThreshold `json:"thresholds"`
	// The type of the service level objective.
	Type SLOType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewServiceLevelObjectiveRequest instantiates a new ServiceLevelObjectiveRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceLevelObjectiveRequest(name string, thresholds []SLOThreshold, typeVar SLOType) *ServiceLevelObjectiveRequest {
	this := ServiceLevelObjectiveRequest{}
	this.Name = name
	this.Thresholds = thresholds
	this.Type = typeVar
	return &this
}

// NewServiceLevelObjectiveRequestWithDefaults instantiates a new ServiceLevelObjectiveRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceLevelObjectiveRequestWithDefaults() *ServiceLevelObjectiveRequest {
	this := ServiceLevelObjectiveRequest{}
	return &this
}

// GetDescription returns the Description field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ServiceLevelObjectiveRequest) GetDescription() string {
	if o == nil || o.Description.Get() == nil {
		var ret string
		return ret
	}
	return *o.Description.Get()
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *ServiceLevelObjectiveRequest) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Description.Get(), o.Description.IsSet()
}

// HasDescription returns a boolean if a field has been set.
func (o *ServiceLevelObjectiveRequest) HasDescription() bool {
	if o != nil && o.Description.IsSet() {
		return true
	}

	return false
}

// SetDescription gets a reference to the given NullableString and assigns it to the Description field.
func (o *ServiceLevelObjectiveRequest) SetDescription(v string) {
	o.Description.Set(&v)
}

// SetDescriptionNil sets the value for Description to be an explicit nil.
func (o *ServiceLevelObjectiveRequest) SetDescriptionNil() {
	o.Description.Set(nil)
}

// UnsetDescription ensures that no value is present for Description, not even an explicit nil.
func (o *ServiceLevelObjectiveRequest) UnsetDescription() {
	o.Description.Unset()
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *ServiceLevelObjectiveRequest) GetGroups() []string {
	if o == nil || o.Groups == nil {
		var ret []string
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetGroupsOk() (*[]string, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *ServiceLevelObjectiveRequest) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given []string and assigns it to the Groups field.
func (o *ServiceLevelObjectiveRequest) SetGroups(v []string) {
	o.Groups = v
}

// GetMonitorIds returns the MonitorIds field value if set, zero value otherwise.
func (o *ServiceLevelObjectiveRequest) GetMonitorIds() []int64 {
	if o == nil || o.MonitorIds == nil {
		var ret []int64
		return ret
	}
	return o.MonitorIds
}

// GetMonitorIdsOk returns a tuple with the MonitorIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetMonitorIdsOk() (*[]int64, bool) {
	if o == nil || o.MonitorIds == nil {
		return nil, false
	}
	return &o.MonitorIds, true
}

// HasMonitorIds returns a boolean if a field has been set.
func (o *ServiceLevelObjectiveRequest) HasMonitorIds() bool {
	if o != nil && o.MonitorIds != nil {
		return true
	}

	return false
}

// SetMonitorIds gets a reference to the given []int64 and assigns it to the MonitorIds field.
func (o *ServiceLevelObjectiveRequest) SetMonitorIds(v []int64) {
	o.MonitorIds = v
}

// GetName returns the Name field value.
func (o *ServiceLevelObjectiveRequest) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *ServiceLevelObjectiveRequest) SetName(v string) {
	o.Name = v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *ServiceLevelObjectiveRequest) GetQuery() ServiceLevelObjectiveQuery {
	if o == nil || o.Query == nil {
		var ret ServiceLevelObjectiveQuery
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetQueryOk() (*ServiceLevelObjectiveQuery, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *ServiceLevelObjectiveRequest) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given ServiceLevelObjectiveQuery and assigns it to the Query field.
func (o *ServiceLevelObjectiveRequest) SetQuery(v ServiceLevelObjectiveQuery) {
	o.Query = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ServiceLevelObjectiveRequest) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ServiceLevelObjectiveRequest) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ServiceLevelObjectiveRequest) SetTags(v []string) {
	o.Tags = v
}

// GetThresholds returns the Thresholds field value.
func (o *ServiceLevelObjectiveRequest) GetThresholds() []SLOThreshold {
	if o == nil {
		var ret []SLOThreshold
		return ret
	}
	return o.Thresholds
}

// GetThresholdsOk returns a tuple with the Thresholds field value
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetThresholdsOk() (*[]SLOThreshold, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Thresholds, true
}

// SetThresholds sets field value.
func (o *ServiceLevelObjectiveRequest) SetThresholds(v []SLOThreshold) {
	o.Thresholds = v
}

// GetType returns the Type field value.
func (o *ServiceLevelObjectiveRequest) GetType() SLOType {
	if o == nil {
		var ret SLOType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceLevelObjectiveRequest) GetTypeOk() (*SLOType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceLevelObjectiveRequest) SetType(v SLOType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceLevelObjectiveRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Description.IsSet() {
		toSerialize["description"] = o.Description.Get()
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if o.MonitorIds != nil {
		toSerialize["monitor_ids"] = o.MonitorIds
	}
	toSerialize["name"] = o.Name
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	toSerialize["thresholds"] = o.Thresholds
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceLevelObjectiveRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name       *string         `json:"name"`
		Thresholds *[]SLOThreshold `json:"thresholds"`
		Type       *SLOType        `json:"type"`
	}{}
	all := struct {
		Description NullableString              `json:"description,omitempty"`
		Groups      []string                    `json:"groups,omitempty"`
		MonitorIds  []int64                     `json:"monitor_ids,omitempty"`
		Name        string                      `json:"name"`
		Query       *ServiceLevelObjectiveQuery `json:"query,omitempty"`
		Tags        []string                    `json:"tags,omitempty"`
		Thresholds  []SLOThreshold              `json:"thresholds"`
		Type        SLOType                     `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Thresholds == nil {
		return fmt.Errorf("Required field thresholds missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Description = all.Description
	o.Groups = all.Groups
	o.MonitorIds = all.MonitorIds
	o.Name = all.Name
	if all.Query != nil && all.Query.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Query = all.Query
	o.Tags = all.Tags
	o.Thresholds = all.Thresholds
	o.Type = all.Type
	return nil
}

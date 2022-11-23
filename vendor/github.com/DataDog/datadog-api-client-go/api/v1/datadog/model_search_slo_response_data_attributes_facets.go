// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SearchSLOResponseDataAttributesFacets Facets
type SearchSLOResponseDataAttributesFacets struct {
	// All tags associated with an SLO.
	AllTags []SearchSLOResponseDataAttributesFacetsObjectString `json:"all_tags,omitempty"`
	// Creator of an SLO.
	CreatorName []SearchSLOResponseDataAttributesFacetsObjectString `json:"creator_name,omitempty"`
	// Tags with the `env` tag key.
	EnvTags []SearchSLOResponseDataAttributesFacetsObjectString `json:"env_tags,omitempty"`
	// Tags with the `service` tag key.
	ServiceTags []SearchSLOResponseDataAttributesFacetsObjectString `json:"service_tags,omitempty"`
	// Type of SLO.
	SloType []SearchSLOResponseDataAttributesFacetsObjectInt `json:"slo_type,omitempty"`
	// SLO Target
	Target []SearchSLOResponseDataAttributesFacetsObjectInt `json:"target,omitempty"`
	// Tags with the `team` tag key.
	TeamTags []SearchSLOResponseDataAttributesFacetsObjectString `json:"team_tags,omitempty"`
	// Timeframes of SLOs.
	Timeframe []SearchSLOResponseDataAttributesFacetsObjectString `json:"timeframe,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponseDataAttributesFacets instantiates a new SearchSLOResponseDataAttributesFacets object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponseDataAttributesFacets() *SearchSLOResponseDataAttributesFacets {
	this := SearchSLOResponseDataAttributesFacets{}
	return &this
}

// NewSearchSLOResponseDataAttributesFacetsWithDefaults instantiates a new SearchSLOResponseDataAttributesFacets object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseDataAttributesFacetsWithDefaults() *SearchSLOResponseDataAttributesFacets {
	this := SearchSLOResponseDataAttributesFacets{}
	return &this
}

// GetAllTags returns the AllTags field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetAllTags() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.AllTags == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.AllTags
}

// GetAllTagsOk returns a tuple with the AllTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetAllTagsOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.AllTags == nil {
		return nil, false
	}
	return &o.AllTags, true
}

// HasAllTags returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasAllTags() bool {
	if o != nil && o.AllTags != nil {
		return true
	}

	return false
}

// SetAllTags gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the AllTags field.
func (o *SearchSLOResponseDataAttributesFacets) SetAllTags(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.AllTags = v
}

// GetCreatorName returns the CreatorName field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetCreatorName() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.CreatorName == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.CreatorName
}

// GetCreatorNameOk returns a tuple with the CreatorName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetCreatorNameOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.CreatorName == nil {
		return nil, false
	}
	return &o.CreatorName, true
}

// HasCreatorName returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasCreatorName() bool {
	if o != nil && o.CreatorName != nil {
		return true
	}

	return false
}

// SetCreatorName gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the CreatorName field.
func (o *SearchSLOResponseDataAttributesFacets) SetCreatorName(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.CreatorName = v
}

// GetEnvTags returns the EnvTags field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetEnvTags() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.EnvTags == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.EnvTags
}

// GetEnvTagsOk returns a tuple with the EnvTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetEnvTagsOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.EnvTags == nil {
		return nil, false
	}
	return &o.EnvTags, true
}

// HasEnvTags returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasEnvTags() bool {
	if o != nil && o.EnvTags != nil {
		return true
	}

	return false
}

// SetEnvTags gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the EnvTags field.
func (o *SearchSLOResponseDataAttributesFacets) SetEnvTags(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.EnvTags = v
}

// GetServiceTags returns the ServiceTags field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetServiceTags() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.ServiceTags == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.ServiceTags
}

// GetServiceTagsOk returns a tuple with the ServiceTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetServiceTagsOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.ServiceTags == nil {
		return nil, false
	}
	return &o.ServiceTags, true
}

// HasServiceTags returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasServiceTags() bool {
	if o != nil && o.ServiceTags != nil {
		return true
	}

	return false
}

// SetServiceTags gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the ServiceTags field.
func (o *SearchSLOResponseDataAttributesFacets) SetServiceTags(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.ServiceTags = v
}

// GetSloType returns the SloType field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetSloType() []SearchSLOResponseDataAttributesFacetsObjectInt {
	if o == nil || o.SloType == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectInt
		return ret
	}
	return o.SloType
}

// GetSloTypeOk returns a tuple with the SloType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetSloTypeOk() (*[]SearchSLOResponseDataAttributesFacetsObjectInt, bool) {
	if o == nil || o.SloType == nil {
		return nil, false
	}
	return &o.SloType, true
}

// HasSloType returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasSloType() bool {
	if o != nil && o.SloType != nil {
		return true
	}

	return false
}

// SetSloType gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectInt and assigns it to the SloType field.
func (o *SearchSLOResponseDataAttributesFacets) SetSloType(v []SearchSLOResponseDataAttributesFacetsObjectInt) {
	o.SloType = v
}

// GetTarget returns the Target field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetTarget() []SearchSLOResponseDataAttributesFacetsObjectInt {
	if o == nil || o.Target == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectInt
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetTargetOk() (*[]SearchSLOResponseDataAttributesFacetsObjectInt, bool) {
	if o == nil || o.Target == nil {
		return nil, false
	}
	return &o.Target, true
}

// HasTarget returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasTarget() bool {
	if o != nil && o.Target != nil {
		return true
	}

	return false
}

// SetTarget gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectInt and assigns it to the Target field.
func (o *SearchSLOResponseDataAttributesFacets) SetTarget(v []SearchSLOResponseDataAttributesFacetsObjectInt) {
	o.Target = v
}

// GetTeamTags returns the TeamTags field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetTeamTags() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.TeamTags == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.TeamTags
}

// GetTeamTagsOk returns a tuple with the TeamTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetTeamTagsOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.TeamTags == nil {
		return nil, false
	}
	return &o.TeamTags, true
}

// HasTeamTags returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasTeamTags() bool {
	if o != nil && o.TeamTags != nil {
		return true
	}

	return false
}

// SetTeamTags gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the TeamTags field.
func (o *SearchSLOResponseDataAttributesFacets) SetTeamTags(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.TeamTags = v
}

// GetTimeframe returns the Timeframe field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributesFacets) GetTimeframe() []SearchSLOResponseDataAttributesFacetsObjectString {
	if o == nil || o.Timeframe == nil {
		var ret []SearchSLOResponseDataAttributesFacetsObjectString
		return ret
	}
	return o.Timeframe
}

// GetTimeframeOk returns a tuple with the Timeframe field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributesFacets) GetTimeframeOk() (*[]SearchSLOResponseDataAttributesFacetsObjectString, bool) {
	if o == nil || o.Timeframe == nil {
		return nil, false
	}
	return &o.Timeframe, true
}

// HasTimeframe returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributesFacets) HasTimeframe() bool {
	if o != nil && o.Timeframe != nil {
		return true
	}

	return false
}

// SetTimeframe gets a reference to the given []SearchSLOResponseDataAttributesFacetsObjectString and assigns it to the Timeframe field.
func (o *SearchSLOResponseDataAttributesFacets) SetTimeframe(v []SearchSLOResponseDataAttributesFacetsObjectString) {
	o.Timeframe = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponseDataAttributesFacets) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AllTags != nil {
		toSerialize["all_tags"] = o.AllTags
	}
	if o.CreatorName != nil {
		toSerialize["creator_name"] = o.CreatorName
	}
	if o.EnvTags != nil {
		toSerialize["env_tags"] = o.EnvTags
	}
	if o.ServiceTags != nil {
		toSerialize["service_tags"] = o.ServiceTags
	}
	if o.SloType != nil {
		toSerialize["slo_type"] = o.SloType
	}
	if o.Target != nil {
		toSerialize["target"] = o.Target
	}
	if o.TeamTags != nil {
		toSerialize["team_tags"] = o.TeamTags
	}
	if o.Timeframe != nil {
		toSerialize["timeframe"] = o.Timeframe
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponseDataAttributesFacets) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AllTags     []SearchSLOResponseDataAttributesFacetsObjectString `json:"all_tags,omitempty"`
		CreatorName []SearchSLOResponseDataAttributesFacetsObjectString `json:"creator_name,omitempty"`
		EnvTags     []SearchSLOResponseDataAttributesFacetsObjectString `json:"env_tags,omitempty"`
		ServiceTags []SearchSLOResponseDataAttributesFacetsObjectString `json:"service_tags,omitempty"`
		SloType     []SearchSLOResponseDataAttributesFacetsObjectInt    `json:"slo_type,omitempty"`
		Target      []SearchSLOResponseDataAttributesFacetsObjectInt    `json:"target,omitempty"`
		TeamTags    []SearchSLOResponseDataAttributesFacetsObjectString `json:"team_tags,omitempty"`
		Timeframe   []SearchSLOResponseDataAttributesFacetsObjectString `json:"timeframe,omitempty"`
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
	o.AllTags = all.AllTags
	o.CreatorName = all.CreatorName
	o.EnvTags = all.EnvTags
	o.ServiceTags = all.ServiceTags
	o.SloType = all.SloType
	o.Target = all.Target
	o.TeamTags = all.TeamTags
	o.Timeframe = all.Timeframe
	return nil
}

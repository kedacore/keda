// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorGroupSearchResponse The response of a monitor group search.
type MonitorGroupSearchResponse struct {
	// The counts of monitor groups per different criteria.
	Counts *MonitorGroupSearchResponseCounts `json:"counts,omitempty"`
	// The list of found monitor groups.
	Groups []MonitorGroupSearchResult `json:"groups,omitempty"`
	// Metadata about the response.
	Metadata *MonitorSearchResponseMetadata `json:"metadata,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorGroupSearchResponse instantiates a new MonitorGroupSearchResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorGroupSearchResponse() *MonitorGroupSearchResponse {
	this := MonitorGroupSearchResponse{}
	return &this
}

// NewMonitorGroupSearchResponseWithDefaults instantiates a new MonitorGroupSearchResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorGroupSearchResponseWithDefaults() *MonitorGroupSearchResponse {
	this := MonitorGroupSearchResponse{}
	return &this
}

// GetCounts returns the Counts field value if set, zero value otherwise.
func (o *MonitorGroupSearchResponse) GetCounts() MonitorGroupSearchResponseCounts {
	if o == nil || o.Counts == nil {
		var ret MonitorGroupSearchResponseCounts
		return ret
	}
	return *o.Counts
}

// GetCountsOk returns a tuple with the Counts field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResponse) GetCountsOk() (*MonitorGroupSearchResponseCounts, bool) {
	if o == nil || o.Counts == nil {
		return nil, false
	}
	return o.Counts, true
}

// HasCounts returns a boolean if a field has been set.
func (o *MonitorGroupSearchResponse) HasCounts() bool {
	if o != nil && o.Counts != nil {
		return true
	}

	return false
}

// SetCounts gets a reference to the given MonitorGroupSearchResponseCounts and assigns it to the Counts field.
func (o *MonitorGroupSearchResponse) SetCounts(v MonitorGroupSearchResponseCounts) {
	o.Counts = &v
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *MonitorGroupSearchResponse) GetGroups() []MonitorGroupSearchResult {
	if o == nil || o.Groups == nil {
		var ret []MonitorGroupSearchResult
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResponse) GetGroupsOk() (*[]MonitorGroupSearchResult, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *MonitorGroupSearchResponse) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given []MonitorGroupSearchResult and assigns it to the Groups field.
func (o *MonitorGroupSearchResponse) SetGroups(v []MonitorGroupSearchResult) {
	o.Groups = v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *MonitorGroupSearchResponse) GetMetadata() MonitorSearchResponseMetadata {
	if o == nil || o.Metadata == nil {
		var ret MonitorSearchResponseMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorGroupSearchResponse) GetMetadataOk() (*MonitorSearchResponseMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *MonitorGroupSearchResponse) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given MonitorSearchResponseMetadata and assigns it to the Metadata field.
func (o *MonitorGroupSearchResponse) SetMetadata(v MonitorSearchResponseMetadata) {
	o.Metadata = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorGroupSearchResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Counts != nil {
		toSerialize["counts"] = o.Counts
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorGroupSearchResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Counts   *MonitorGroupSearchResponseCounts `json:"counts,omitempty"`
		Groups   []MonitorGroupSearchResult        `json:"groups,omitempty"`
		Metadata *MonitorSearchResponseMetadata    `json:"metadata,omitempty"`
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
	if all.Counts != nil && all.Counts.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Counts = all.Counts
	o.Groups = all.Groups
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	return nil
}

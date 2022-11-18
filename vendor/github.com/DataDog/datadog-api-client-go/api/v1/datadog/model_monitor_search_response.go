// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonitorSearchResponse The response form a monitor search.
type MonitorSearchResponse struct {
	// The counts of monitors per different criteria.
	Counts *MonitorSearchResponseCounts `json:"counts,omitempty"`
	// Metadata about the response.
	Metadata *MonitorSearchResponseMetadata `json:"metadata,omitempty"`
	// The list of found monitors.
	Monitors []MonitorSearchResult `json:"monitors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorSearchResponse instantiates a new MonitorSearchResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorSearchResponse() *MonitorSearchResponse {
	this := MonitorSearchResponse{}
	return &this
}

// NewMonitorSearchResponseWithDefaults instantiates a new MonitorSearchResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorSearchResponseWithDefaults() *MonitorSearchResponse {
	this := MonitorSearchResponse{}
	return &this
}

// GetCounts returns the Counts field value if set, zero value otherwise.
func (o *MonitorSearchResponse) GetCounts() MonitorSearchResponseCounts {
	if o == nil || o.Counts == nil {
		var ret MonitorSearchResponseCounts
		return ret
	}
	return *o.Counts
}

// GetCountsOk returns a tuple with the Counts field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponse) GetCountsOk() (*MonitorSearchResponseCounts, bool) {
	if o == nil || o.Counts == nil {
		return nil, false
	}
	return o.Counts, true
}

// HasCounts returns a boolean if a field has been set.
func (o *MonitorSearchResponse) HasCounts() bool {
	if o != nil && o.Counts != nil {
		return true
	}

	return false
}

// SetCounts gets a reference to the given MonitorSearchResponseCounts and assigns it to the Counts field.
func (o *MonitorSearchResponse) SetCounts(v MonitorSearchResponseCounts) {
	o.Counts = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *MonitorSearchResponse) GetMetadata() MonitorSearchResponseMetadata {
	if o == nil || o.Metadata == nil {
		var ret MonitorSearchResponseMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponse) GetMetadataOk() (*MonitorSearchResponseMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *MonitorSearchResponse) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given MonitorSearchResponseMetadata and assigns it to the Metadata field.
func (o *MonitorSearchResponse) SetMetadata(v MonitorSearchResponseMetadata) {
	o.Metadata = &v
}

// GetMonitors returns the Monitors field value if set, zero value otherwise.
func (o *MonitorSearchResponse) GetMonitors() []MonitorSearchResult {
	if o == nil || o.Monitors == nil {
		var ret []MonitorSearchResult
		return ret
	}
	return o.Monitors
}

// GetMonitorsOk returns a tuple with the Monitors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorSearchResponse) GetMonitorsOk() (*[]MonitorSearchResult, bool) {
	if o == nil || o.Monitors == nil {
		return nil, false
	}
	return &o.Monitors, true
}

// HasMonitors returns a boolean if a field has been set.
func (o *MonitorSearchResponse) HasMonitors() bool {
	if o != nil && o.Monitors != nil {
		return true
	}

	return false
}

// SetMonitors gets a reference to the given []MonitorSearchResult and assigns it to the Monitors field.
func (o *MonitorSearchResponse) SetMonitors(v []MonitorSearchResult) {
	o.Monitors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorSearchResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Counts != nil {
		toSerialize["counts"] = o.Counts
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.Monitors != nil {
		toSerialize["monitors"] = o.Monitors
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorSearchResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Counts   *MonitorSearchResponseCounts   `json:"counts,omitempty"`
		Metadata *MonitorSearchResponseMetadata `json:"metadata,omitempty"`
		Monitors []MonitorSearchResult          `json:"monitors,omitempty"`
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
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Monitors = all.Monitors
	return nil
}

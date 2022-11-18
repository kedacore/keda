// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// Host Object representing a host.
type Host struct {
	// Host aliases collected by Datadog.
	Aliases []string `json:"aliases,omitempty"`
	// The Datadog integrations reporting metrics for the host.
	Apps []string `json:"apps,omitempty"`
	// AWS name of your host.
	AwsName *string `json:"aws_name,omitempty"`
	// The host name.
	HostName *string `json:"host_name,omitempty"`
	// The host ID.
	Id *int64 `json:"id,omitempty"`
	// If a host is muted or unmuted.
	IsMuted *bool `json:"is_muted,omitempty"`
	// Last time the host reported a metric data point.
	LastReportedTime *int64 `json:"last_reported_time,omitempty"`
	// Metadata associated with your host.
	Meta *HostMeta `json:"meta,omitempty"`
	// Host Metrics collected.
	Metrics *HostMetrics `json:"metrics,omitempty"`
	// Timeout of the mute applied to your host.
	MuteTimeout *int64 `json:"mute_timeout,omitempty"`
	// The host name.
	Name *string `json:"name,omitempty"`
	// Source or cloud provider associated with your host.
	Sources []string `json:"sources,omitempty"`
	// List of tags for each source (AWS, Datadog Agent, Chef..).
	TagsBySource map[string][]string `json:"tags_by_source,omitempty"`
	// Displays UP when the expected metrics are received and displays `???` if no metrics are received.
	Up *bool `json:"up,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHost instantiates a new Host object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHost() *Host {
	this := Host{}
	return &this
}

// NewHostWithDefaults instantiates a new Host object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostWithDefaults() *Host {
	this := Host{}
	return &this
}

// GetAliases returns the Aliases field value if set, zero value otherwise.
func (o *Host) GetAliases() []string {
	if o == nil || o.Aliases == nil {
		var ret []string
		return ret
	}
	return o.Aliases
}

// GetAliasesOk returns a tuple with the Aliases field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetAliasesOk() (*[]string, bool) {
	if o == nil || o.Aliases == nil {
		return nil, false
	}
	return &o.Aliases, true
}

// HasAliases returns a boolean if a field has been set.
func (o *Host) HasAliases() bool {
	if o != nil && o.Aliases != nil {
		return true
	}

	return false
}

// SetAliases gets a reference to the given []string and assigns it to the Aliases field.
func (o *Host) SetAliases(v []string) {
	o.Aliases = v
}

// GetApps returns the Apps field value if set, zero value otherwise.
func (o *Host) GetApps() []string {
	if o == nil || o.Apps == nil {
		var ret []string
		return ret
	}
	return o.Apps
}

// GetAppsOk returns a tuple with the Apps field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetAppsOk() (*[]string, bool) {
	if o == nil || o.Apps == nil {
		return nil, false
	}
	return &o.Apps, true
}

// HasApps returns a boolean if a field has been set.
func (o *Host) HasApps() bool {
	if o != nil && o.Apps != nil {
		return true
	}

	return false
}

// SetApps gets a reference to the given []string and assigns it to the Apps field.
func (o *Host) SetApps(v []string) {
	o.Apps = v
}

// GetAwsName returns the AwsName field value if set, zero value otherwise.
func (o *Host) GetAwsName() string {
	if o == nil || o.AwsName == nil {
		var ret string
		return ret
	}
	return *o.AwsName
}

// GetAwsNameOk returns a tuple with the AwsName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetAwsNameOk() (*string, bool) {
	if o == nil || o.AwsName == nil {
		return nil, false
	}
	return o.AwsName, true
}

// HasAwsName returns a boolean if a field has been set.
func (o *Host) HasAwsName() bool {
	if o != nil && o.AwsName != nil {
		return true
	}

	return false
}

// SetAwsName gets a reference to the given string and assigns it to the AwsName field.
func (o *Host) SetAwsName(v string) {
	o.AwsName = &v
}

// GetHostName returns the HostName field value if set, zero value otherwise.
func (o *Host) GetHostName() string {
	if o == nil || o.HostName == nil {
		var ret string
		return ret
	}
	return *o.HostName
}

// GetHostNameOk returns a tuple with the HostName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetHostNameOk() (*string, bool) {
	if o == nil || o.HostName == nil {
		return nil, false
	}
	return o.HostName, true
}

// HasHostName returns a boolean if a field has been set.
func (o *Host) HasHostName() bool {
	if o != nil && o.HostName != nil {
		return true
	}

	return false
}

// SetHostName gets a reference to the given string and assigns it to the HostName field.
func (o *Host) SetHostName(v string) {
	o.HostName = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *Host) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *Host) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *Host) SetId(v int64) {
	o.Id = &v
}

// GetIsMuted returns the IsMuted field value if set, zero value otherwise.
func (o *Host) GetIsMuted() bool {
	if o == nil || o.IsMuted == nil {
		var ret bool
		return ret
	}
	return *o.IsMuted
}

// GetIsMutedOk returns a tuple with the IsMuted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetIsMutedOk() (*bool, bool) {
	if o == nil || o.IsMuted == nil {
		return nil, false
	}
	return o.IsMuted, true
}

// HasIsMuted returns a boolean if a field has been set.
func (o *Host) HasIsMuted() bool {
	if o != nil && o.IsMuted != nil {
		return true
	}

	return false
}

// SetIsMuted gets a reference to the given bool and assigns it to the IsMuted field.
func (o *Host) SetIsMuted(v bool) {
	o.IsMuted = &v
}

// GetLastReportedTime returns the LastReportedTime field value if set, zero value otherwise.
func (o *Host) GetLastReportedTime() int64 {
	if o == nil || o.LastReportedTime == nil {
		var ret int64
		return ret
	}
	return *o.LastReportedTime
}

// GetLastReportedTimeOk returns a tuple with the LastReportedTime field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetLastReportedTimeOk() (*int64, bool) {
	if o == nil || o.LastReportedTime == nil {
		return nil, false
	}
	return o.LastReportedTime, true
}

// HasLastReportedTime returns a boolean if a field has been set.
func (o *Host) HasLastReportedTime() bool {
	if o != nil && o.LastReportedTime != nil {
		return true
	}

	return false
}

// SetLastReportedTime gets a reference to the given int64 and assigns it to the LastReportedTime field.
func (o *Host) SetLastReportedTime(v int64) {
	o.LastReportedTime = &v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *Host) GetMeta() HostMeta {
	if o == nil || o.Meta == nil {
		var ret HostMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetMetaOk() (*HostMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *Host) HasMeta() bool {
	if o != nil && o.Meta != nil {
		return true
	}

	return false
}

// SetMeta gets a reference to the given HostMeta and assigns it to the Meta field.
func (o *Host) SetMeta(v HostMeta) {
	o.Meta = &v
}

// GetMetrics returns the Metrics field value if set, zero value otherwise.
func (o *Host) GetMetrics() HostMetrics {
	if o == nil || o.Metrics == nil {
		var ret HostMetrics
		return ret
	}
	return *o.Metrics
}

// GetMetricsOk returns a tuple with the Metrics field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetMetricsOk() (*HostMetrics, bool) {
	if o == nil || o.Metrics == nil {
		return nil, false
	}
	return o.Metrics, true
}

// HasMetrics returns a boolean if a field has been set.
func (o *Host) HasMetrics() bool {
	if o != nil && o.Metrics != nil {
		return true
	}

	return false
}

// SetMetrics gets a reference to the given HostMetrics and assigns it to the Metrics field.
func (o *Host) SetMetrics(v HostMetrics) {
	o.Metrics = &v
}

// GetMuteTimeout returns the MuteTimeout field value if set, zero value otherwise.
func (o *Host) GetMuteTimeout() int64 {
	if o == nil || o.MuteTimeout == nil {
		var ret int64
		return ret
	}
	return *o.MuteTimeout
}

// GetMuteTimeoutOk returns a tuple with the MuteTimeout field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetMuteTimeoutOk() (*int64, bool) {
	if o == nil || o.MuteTimeout == nil {
		return nil, false
	}
	return o.MuteTimeout, true
}

// HasMuteTimeout returns a boolean if a field has been set.
func (o *Host) HasMuteTimeout() bool {
	if o != nil && o.MuteTimeout != nil {
		return true
	}

	return false
}

// SetMuteTimeout gets a reference to the given int64 and assigns it to the MuteTimeout field.
func (o *Host) SetMuteTimeout(v int64) {
	o.MuteTimeout = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *Host) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *Host) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *Host) SetName(v string) {
	o.Name = &v
}

// GetSources returns the Sources field value if set, zero value otherwise.
func (o *Host) GetSources() []string {
	if o == nil || o.Sources == nil {
		var ret []string
		return ret
	}
	return o.Sources
}

// GetSourcesOk returns a tuple with the Sources field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetSourcesOk() (*[]string, bool) {
	if o == nil || o.Sources == nil {
		return nil, false
	}
	return &o.Sources, true
}

// HasSources returns a boolean if a field has been set.
func (o *Host) HasSources() bool {
	if o != nil && o.Sources != nil {
		return true
	}

	return false
}

// SetSources gets a reference to the given []string and assigns it to the Sources field.
func (o *Host) SetSources(v []string) {
	o.Sources = v
}

// GetTagsBySource returns the TagsBySource field value if set, zero value otherwise.
func (o *Host) GetTagsBySource() map[string][]string {
	if o == nil || o.TagsBySource == nil {
		var ret map[string][]string
		return ret
	}
	return o.TagsBySource
}

// GetTagsBySourceOk returns a tuple with the TagsBySource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetTagsBySourceOk() (*map[string][]string, bool) {
	if o == nil || o.TagsBySource == nil {
		return nil, false
	}
	return &o.TagsBySource, true
}

// HasTagsBySource returns a boolean if a field has been set.
func (o *Host) HasTagsBySource() bool {
	if o != nil && o.TagsBySource != nil {
		return true
	}

	return false
}

// SetTagsBySource gets a reference to the given map[string][]string and assigns it to the TagsBySource field.
func (o *Host) SetTagsBySource(v map[string][]string) {
	o.TagsBySource = v
}

// GetUp returns the Up field value if set, zero value otherwise.
func (o *Host) GetUp() bool {
	if o == nil || o.Up == nil {
		var ret bool
		return ret
	}
	return *o.Up
}

// GetUpOk returns a tuple with the Up field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Host) GetUpOk() (*bool, bool) {
	if o == nil || o.Up == nil {
		return nil, false
	}
	return o.Up, true
}

// HasUp returns a boolean if a field has been set.
func (o *Host) HasUp() bool {
	if o != nil && o.Up != nil {
		return true
	}

	return false
}

// SetUp gets a reference to the given bool and assigns it to the Up field.
func (o *Host) SetUp(v bool) {
	o.Up = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o Host) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aliases != nil {
		toSerialize["aliases"] = o.Aliases
	}
	if o.Apps != nil {
		toSerialize["apps"] = o.Apps
	}
	if o.AwsName != nil {
		toSerialize["aws_name"] = o.AwsName
	}
	if o.HostName != nil {
		toSerialize["host_name"] = o.HostName
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IsMuted != nil {
		toSerialize["is_muted"] = o.IsMuted
	}
	if o.LastReportedTime != nil {
		toSerialize["last_reported_time"] = o.LastReportedTime
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}
	if o.Metrics != nil {
		toSerialize["metrics"] = o.Metrics
	}
	if o.MuteTimeout != nil {
		toSerialize["mute_timeout"] = o.MuteTimeout
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Sources != nil {
		toSerialize["sources"] = o.Sources
	}
	if o.TagsBySource != nil {
		toSerialize["tags_by_source"] = o.TagsBySource
	}
	if o.Up != nil {
		toSerialize["up"] = o.Up
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *Host) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aliases          []string            `json:"aliases,omitempty"`
		Apps             []string            `json:"apps,omitempty"`
		AwsName          *string             `json:"aws_name,omitempty"`
		HostName         *string             `json:"host_name,omitempty"`
		Id               *int64              `json:"id,omitempty"`
		IsMuted          *bool               `json:"is_muted,omitempty"`
		LastReportedTime *int64              `json:"last_reported_time,omitempty"`
		Meta             *HostMeta           `json:"meta,omitempty"`
		Metrics          *HostMetrics        `json:"metrics,omitempty"`
		MuteTimeout      *int64              `json:"mute_timeout,omitempty"`
		Name             *string             `json:"name,omitempty"`
		Sources          []string            `json:"sources,omitempty"`
		TagsBySource     map[string][]string `json:"tags_by_source,omitempty"`
		Up               *bool               `json:"up,omitempty"`
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
	o.Aliases = all.Aliases
	o.Apps = all.Apps
	o.AwsName = all.AwsName
	o.HostName = all.HostName
	o.Id = all.Id
	o.IsMuted = all.IsMuted
	o.LastReportedTime = all.LastReportedTime
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	if all.Metrics != nil && all.Metrics.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metrics = all.Metrics
	o.MuteTimeout = all.MuteTimeout
	o.Name = all.Name
	o.Sources = all.Sources
	o.TagsBySource = all.TagsBySource
	o.Up = all.Up
	return nil
}

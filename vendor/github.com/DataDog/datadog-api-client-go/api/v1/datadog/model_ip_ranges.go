// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// IPRanges IP ranges.
type IPRanges struct {
	// Available prefix information for the Agent endpoints.
	Agents *IPPrefixesAgents `json:"agents,omitempty"`
	// Available prefix information for the API endpoints.
	Api *IPPrefixesAPI `json:"api,omitempty"`
	// Available prefix information for the APM endpoints.
	Apm *IPPrefixesAPM `json:"apm,omitempty"`
	// Available prefix information for the Logs endpoints.
	Logs *IPPrefixesLogs `json:"logs,omitempty"`
	// Date when last updated, in the form `YYYY-MM-DD-hh-mm-ss`.
	Modified *string `json:"modified,omitempty"`
	// Available prefix information for the Process endpoints.
	Process *IPPrefixesProcess `json:"process,omitempty"`
	// Available prefix information for the Synthetics endpoints.
	Synthetics *IPPrefixesSynthetics `json:"synthetics,omitempty"`
	// Available prefix information for the Synthetics Private Locations endpoints.
	SyntheticsPrivateLocations *IPPrefixesSyntheticsPrivateLocations `json:"synthetics-private-locations,omitempty"`
	// Version of the IP list.
	Version *int64 `json:"version,omitempty"`
	// Available prefix information for the Webhook endpoints.
	Webhooks *IPPrefixesWebhooks `json:"webhooks,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewIPRanges instantiates a new IPRanges object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPRanges() *IPRanges {
	this := IPRanges{}
	return &this
}

// NewIPRangesWithDefaults instantiates a new IPRanges object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPRangesWithDefaults() *IPRanges {
	this := IPRanges{}
	return &this
}

// GetAgents returns the Agents field value if set, zero value otherwise.
func (o *IPRanges) GetAgents() IPPrefixesAgents {
	if o == nil || o.Agents == nil {
		var ret IPPrefixesAgents
		return ret
	}
	return *o.Agents
}

// GetAgentsOk returns a tuple with the Agents field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetAgentsOk() (*IPPrefixesAgents, bool) {
	if o == nil || o.Agents == nil {
		return nil, false
	}
	return o.Agents, true
}

// HasAgents returns a boolean if a field has been set.
func (o *IPRanges) HasAgents() bool {
	if o != nil && o.Agents != nil {
		return true
	}

	return false
}

// SetAgents gets a reference to the given IPPrefixesAgents and assigns it to the Agents field.
func (o *IPRanges) SetAgents(v IPPrefixesAgents) {
	o.Agents = &v
}

// GetApi returns the Api field value if set, zero value otherwise.
func (o *IPRanges) GetApi() IPPrefixesAPI {
	if o == nil || o.Api == nil {
		var ret IPPrefixesAPI
		return ret
	}
	return *o.Api
}

// GetApiOk returns a tuple with the Api field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetApiOk() (*IPPrefixesAPI, bool) {
	if o == nil || o.Api == nil {
		return nil, false
	}
	return o.Api, true
}

// HasApi returns a boolean if a field has been set.
func (o *IPRanges) HasApi() bool {
	if o != nil && o.Api != nil {
		return true
	}

	return false
}

// SetApi gets a reference to the given IPPrefixesAPI and assigns it to the Api field.
func (o *IPRanges) SetApi(v IPPrefixesAPI) {
	o.Api = &v
}

// GetApm returns the Apm field value if set, zero value otherwise.
func (o *IPRanges) GetApm() IPPrefixesAPM {
	if o == nil || o.Apm == nil {
		var ret IPPrefixesAPM
		return ret
	}
	return *o.Apm
}

// GetApmOk returns a tuple with the Apm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetApmOk() (*IPPrefixesAPM, bool) {
	if o == nil || o.Apm == nil {
		return nil, false
	}
	return o.Apm, true
}

// HasApm returns a boolean if a field has been set.
func (o *IPRanges) HasApm() bool {
	if o != nil && o.Apm != nil {
		return true
	}

	return false
}

// SetApm gets a reference to the given IPPrefixesAPM and assigns it to the Apm field.
func (o *IPRanges) SetApm(v IPPrefixesAPM) {
	o.Apm = &v
}

// GetLogs returns the Logs field value if set, zero value otherwise.
func (o *IPRanges) GetLogs() IPPrefixesLogs {
	if o == nil || o.Logs == nil {
		var ret IPPrefixesLogs
		return ret
	}
	return *o.Logs
}

// GetLogsOk returns a tuple with the Logs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetLogsOk() (*IPPrefixesLogs, bool) {
	if o == nil || o.Logs == nil {
		return nil, false
	}
	return o.Logs, true
}

// HasLogs returns a boolean if a field has been set.
func (o *IPRanges) HasLogs() bool {
	if o != nil && o.Logs != nil {
		return true
	}

	return false
}

// SetLogs gets a reference to the given IPPrefixesLogs and assigns it to the Logs field.
func (o *IPRanges) SetLogs(v IPPrefixesLogs) {
	o.Logs = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *IPRanges) GetModified() string {
	if o == nil || o.Modified == nil {
		var ret string
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetModifiedOk() (*string, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *IPRanges) HasModified() bool {
	if o != nil && o.Modified != nil {
		return true
	}

	return false
}

// SetModified gets a reference to the given string and assigns it to the Modified field.
func (o *IPRanges) SetModified(v string) {
	o.Modified = &v
}

// GetProcess returns the Process field value if set, zero value otherwise.
func (o *IPRanges) GetProcess() IPPrefixesProcess {
	if o == nil || o.Process == nil {
		var ret IPPrefixesProcess
		return ret
	}
	return *o.Process
}

// GetProcessOk returns a tuple with the Process field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetProcessOk() (*IPPrefixesProcess, bool) {
	if o == nil || o.Process == nil {
		return nil, false
	}
	return o.Process, true
}

// HasProcess returns a boolean if a field has been set.
func (o *IPRanges) HasProcess() bool {
	if o != nil && o.Process != nil {
		return true
	}

	return false
}

// SetProcess gets a reference to the given IPPrefixesProcess and assigns it to the Process field.
func (o *IPRanges) SetProcess(v IPPrefixesProcess) {
	o.Process = &v
}

// GetSynthetics returns the Synthetics field value if set, zero value otherwise.
func (o *IPRanges) GetSynthetics() IPPrefixesSynthetics {
	if o == nil || o.Synthetics == nil {
		var ret IPPrefixesSynthetics
		return ret
	}
	return *o.Synthetics
}

// GetSyntheticsOk returns a tuple with the Synthetics field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetSyntheticsOk() (*IPPrefixesSynthetics, bool) {
	if o == nil || o.Synthetics == nil {
		return nil, false
	}
	return o.Synthetics, true
}

// HasSynthetics returns a boolean if a field has been set.
func (o *IPRanges) HasSynthetics() bool {
	if o != nil && o.Synthetics != nil {
		return true
	}

	return false
}

// SetSynthetics gets a reference to the given IPPrefixesSynthetics and assigns it to the Synthetics field.
func (o *IPRanges) SetSynthetics(v IPPrefixesSynthetics) {
	o.Synthetics = &v
}

// GetSyntheticsPrivateLocations returns the SyntheticsPrivateLocations field value if set, zero value otherwise.
func (o *IPRanges) GetSyntheticsPrivateLocations() IPPrefixesSyntheticsPrivateLocations {
	if o == nil || o.SyntheticsPrivateLocations == nil {
		var ret IPPrefixesSyntheticsPrivateLocations
		return ret
	}
	return *o.SyntheticsPrivateLocations
}

// GetSyntheticsPrivateLocationsOk returns a tuple with the SyntheticsPrivateLocations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetSyntheticsPrivateLocationsOk() (*IPPrefixesSyntheticsPrivateLocations, bool) {
	if o == nil || o.SyntheticsPrivateLocations == nil {
		return nil, false
	}
	return o.SyntheticsPrivateLocations, true
}

// HasSyntheticsPrivateLocations returns a boolean if a field has been set.
func (o *IPRanges) HasSyntheticsPrivateLocations() bool {
	if o != nil && o.SyntheticsPrivateLocations != nil {
		return true
	}

	return false
}

// SetSyntheticsPrivateLocations gets a reference to the given IPPrefixesSyntheticsPrivateLocations and assigns it to the SyntheticsPrivateLocations field.
func (o *IPRanges) SetSyntheticsPrivateLocations(v IPPrefixesSyntheticsPrivateLocations) {
	o.SyntheticsPrivateLocations = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *IPRanges) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *IPRanges) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *IPRanges) SetVersion(v int64) {
	o.Version = &v
}

// GetWebhooks returns the Webhooks field value if set, zero value otherwise.
func (o *IPRanges) GetWebhooks() IPPrefixesWebhooks {
	if o == nil || o.Webhooks == nil {
		var ret IPPrefixesWebhooks
		return ret
	}
	return *o.Webhooks
}

// GetWebhooksOk returns a tuple with the Webhooks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPRanges) GetWebhooksOk() (*IPPrefixesWebhooks, bool) {
	if o == nil || o.Webhooks == nil {
		return nil, false
	}
	return o.Webhooks, true
}

// HasWebhooks returns a boolean if a field has been set.
func (o *IPRanges) HasWebhooks() bool {
	if o != nil && o.Webhooks != nil {
		return true
	}

	return false
}

// SetWebhooks gets a reference to the given IPPrefixesWebhooks and assigns it to the Webhooks field.
func (o *IPRanges) SetWebhooks(v IPPrefixesWebhooks) {
	o.Webhooks = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPRanges) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Agents != nil {
		toSerialize["agents"] = o.Agents
	}
	if o.Api != nil {
		toSerialize["api"] = o.Api
	}
	if o.Apm != nil {
		toSerialize["apm"] = o.Apm
	}
	if o.Logs != nil {
		toSerialize["logs"] = o.Logs
	}
	if o.Modified != nil {
		toSerialize["modified"] = o.Modified
	}
	if o.Process != nil {
		toSerialize["process"] = o.Process
	}
	if o.Synthetics != nil {
		toSerialize["synthetics"] = o.Synthetics
	}
	if o.SyntheticsPrivateLocations != nil {
		toSerialize["synthetics-private-locations"] = o.SyntheticsPrivateLocations
	}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}
	if o.Webhooks != nil {
		toSerialize["webhooks"] = o.Webhooks
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IPRanges) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Agents                     *IPPrefixesAgents                     `json:"agents,omitempty"`
		Api                        *IPPrefixesAPI                        `json:"api,omitempty"`
		Apm                        *IPPrefixesAPM                        `json:"apm,omitempty"`
		Logs                       *IPPrefixesLogs                       `json:"logs,omitempty"`
		Modified                   *string                               `json:"modified,omitempty"`
		Process                    *IPPrefixesProcess                    `json:"process,omitempty"`
		Synthetics                 *IPPrefixesSynthetics                 `json:"synthetics,omitempty"`
		SyntheticsPrivateLocations *IPPrefixesSyntheticsPrivateLocations `json:"synthetics-private-locations,omitempty"`
		Version                    *int64                                `json:"version,omitempty"`
		Webhooks                   *IPPrefixesWebhooks                   `json:"webhooks,omitempty"`
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
	if all.Agents != nil && all.Agents.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Agents = all.Agents
	if all.Api != nil && all.Api.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Api = all.Api
	if all.Apm != nil && all.Apm.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Apm = all.Apm
	if all.Logs != nil && all.Logs.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Logs = all.Logs
	o.Modified = all.Modified
	if all.Process != nil && all.Process.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Process = all.Process
	if all.Synthetics != nil && all.Synthetics.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Synthetics = all.Synthetics
	if all.SyntheticsPrivateLocations != nil && all.SyntheticsPrivateLocations.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SyntheticsPrivateLocations = all.SyntheticsPrivateLocations
	o.Version = all.Version
	if all.Webhooks != nil && all.Webhooks.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Webhooks = all.Webhooks
	return nil
}

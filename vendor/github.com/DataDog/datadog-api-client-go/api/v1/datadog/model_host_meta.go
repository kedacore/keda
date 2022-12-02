// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostMeta Metadata associated with your host.
type HostMeta struct {
	// A list of Agent checks running on the host.
	AgentChecks [][]interface{} `json:"agent_checks,omitempty"`
	// The Datadog Agent version.
	AgentVersion *string `json:"agent_version,omitempty"`
	// The number of cores.
	CpuCores *int64 `json:"cpuCores,omitempty"`
	// An array of Mac versions.
	FbsdV []string `json:"fbsdV,omitempty"`
	// JSON string containing system information.
	Gohai *string `json:"gohai,omitempty"`
	// Agent install method.
	InstallMethod *HostMetaInstallMethod `json:"install_method,omitempty"`
	// An array of Mac versions.
	MacV []string `json:"macV,omitempty"`
	// The machine architecture.
	Machine *string `json:"machine,omitempty"`
	// Array of Unix versions.
	NixV []string `json:"nixV,omitempty"`
	// The OS platform.
	Platform *string `json:"platform,omitempty"`
	// The processor.
	Processor *string `json:"processor,omitempty"`
	// The Python version.
	PythonV *string `json:"pythonV,omitempty"`
	// The socket fqdn.
	SocketFqdn *string `json:"socket-fqdn,omitempty"`
	// The socket hostname.
	SocketHostname *string `json:"socket-hostname,omitempty"`
	// An array of Windows versions.
	WinV []string `json:"winV,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostMeta instantiates a new HostMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostMeta() *HostMeta {
	this := HostMeta{}
	return &this
}

// NewHostMetaWithDefaults instantiates a new HostMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostMetaWithDefaults() *HostMeta {
	this := HostMeta{}
	return &this
}

// GetAgentChecks returns the AgentChecks field value if set, zero value otherwise.
func (o *HostMeta) GetAgentChecks() [][]interface{} {
	if o == nil || o.AgentChecks == nil {
		var ret [][]interface{}
		return ret
	}
	return o.AgentChecks
}

// GetAgentChecksOk returns a tuple with the AgentChecks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetAgentChecksOk() (*[][]interface{}, bool) {
	if o == nil || o.AgentChecks == nil {
		return nil, false
	}
	return &o.AgentChecks, true
}

// HasAgentChecks returns a boolean if a field has been set.
func (o *HostMeta) HasAgentChecks() bool {
	if o != nil && o.AgentChecks != nil {
		return true
	}

	return false
}

// SetAgentChecks gets a reference to the given [][]interface{} and assigns it to the AgentChecks field.
func (o *HostMeta) SetAgentChecks(v [][]interface{}) {
	o.AgentChecks = v
}

// GetAgentVersion returns the AgentVersion field value if set, zero value otherwise.
func (o *HostMeta) GetAgentVersion() string {
	if o == nil || o.AgentVersion == nil {
		var ret string
		return ret
	}
	return *o.AgentVersion
}

// GetAgentVersionOk returns a tuple with the AgentVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetAgentVersionOk() (*string, bool) {
	if o == nil || o.AgentVersion == nil {
		return nil, false
	}
	return o.AgentVersion, true
}

// HasAgentVersion returns a boolean if a field has been set.
func (o *HostMeta) HasAgentVersion() bool {
	if o != nil && o.AgentVersion != nil {
		return true
	}

	return false
}

// SetAgentVersion gets a reference to the given string and assigns it to the AgentVersion field.
func (o *HostMeta) SetAgentVersion(v string) {
	o.AgentVersion = &v
}

// GetCpuCores returns the CpuCores field value if set, zero value otherwise.
func (o *HostMeta) GetCpuCores() int64 {
	if o == nil || o.CpuCores == nil {
		var ret int64
		return ret
	}
	return *o.CpuCores
}

// GetCpuCoresOk returns a tuple with the CpuCores field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetCpuCoresOk() (*int64, bool) {
	if o == nil || o.CpuCores == nil {
		return nil, false
	}
	return o.CpuCores, true
}

// HasCpuCores returns a boolean if a field has been set.
func (o *HostMeta) HasCpuCores() bool {
	if o != nil && o.CpuCores != nil {
		return true
	}

	return false
}

// SetCpuCores gets a reference to the given int64 and assigns it to the CpuCores field.
func (o *HostMeta) SetCpuCores(v int64) {
	o.CpuCores = &v
}

// GetFbsdV returns the FbsdV field value if set, zero value otherwise.
func (o *HostMeta) GetFbsdV() []string {
	if o == nil || o.FbsdV == nil {
		var ret []string
		return ret
	}
	return o.FbsdV
}

// GetFbsdVOk returns a tuple with the FbsdV field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetFbsdVOk() (*[]string, bool) {
	if o == nil || o.FbsdV == nil {
		return nil, false
	}
	return &o.FbsdV, true
}

// HasFbsdV returns a boolean if a field has been set.
func (o *HostMeta) HasFbsdV() bool {
	if o != nil && o.FbsdV != nil {
		return true
	}

	return false
}

// SetFbsdV gets a reference to the given []string and assigns it to the FbsdV field.
func (o *HostMeta) SetFbsdV(v []string) {
	o.FbsdV = v
}

// GetGohai returns the Gohai field value if set, zero value otherwise.
func (o *HostMeta) GetGohai() string {
	if o == nil || o.Gohai == nil {
		var ret string
		return ret
	}
	return *o.Gohai
}

// GetGohaiOk returns a tuple with the Gohai field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetGohaiOk() (*string, bool) {
	if o == nil || o.Gohai == nil {
		return nil, false
	}
	return o.Gohai, true
}

// HasGohai returns a boolean if a field has been set.
func (o *HostMeta) HasGohai() bool {
	if o != nil && o.Gohai != nil {
		return true
	}

	return false
}

// SetGohai gets a reference to the given string and assigns it to the Gohai field.
func (o *HostMeta) SetGohai(v string) {
	o.Gohai = &v
}

// GetInstallMethod returns the InstallMethod field value if set, zero value otherwise.
func (o *HostMeta) GetInstallMethod() HostMetaInstallMethod {
	if o == nil || o.InstallMethod == nil {
		var ret HostMetaInstallMethod
		return ret
	}
	return *o.InstallMethod
}

// GetInstallMethodOk returns a tuple with the InstallMethod field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetInstallMethodOk() (*HostMetaInstallMethod, bool) {
	if o == nil || o.InstallMethod == nil {
		return nil, false
	}
	return o.InstallMethod, true
}

// HasInstallMethod returns a boolean if a field has been set.
func (o *HostMeta) HasInstallMethod() bool {
	if o != nil && o.InstallMethod != nil {
		return true
	}

	return false
}

// SetInstallMethod gets a reference to the given HostMetaInstallMethod and assigns it to the InstallMethod field.
func (o *HostMeta) SetInstallMethod(v HostMetaInstallMethod) {
	o.InstallMethod = &v
}

// GetMacV returns the MacV field value if set, zero value otherwise.
func (o *HostMeta) GetMacV() []string {
	if o == nil || o.MacV == nil {
		var ret []string
		return ret
	}
	return o.MacV
}

// GetMacVOk returns a tuple with the MacV field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetMacVOk() (*[]string, bool) {
	if o == nil || o.MacV == nil {
		return nil, false
	}
	return &o.MacV, true
}

// HasMacV returns a boolean if a field has been set.
func (o *HostMeta) HasMacV() bool {
	if o != nil && o.MacV != nil {
		return true
	}

	return false
}

// SetMacV gets a reference to the given []string and assigns it to the MacV field.
func (o *HostMeta) SetMacV(v []string) {
	o.MacV = v
}

// GetMachine returns the Machine field value if set, zero value otherwise.
func (o *HostMeta) GetMachine() string {
	if o == nil || o.Machine == nil {
		var ret string
		return ret
	}
	return *o.Machine
}

// GetMachineOk returns a tuple with the Machine field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetMachineOk() (*string, bool) {
	if o == nil || o.Machine == nil {
		return nil, false
	}
	return o.Machine, true
}

// HasMachine returns a boolean if a field has been set.
func (o *HostMeta) HasMachine() bool {
	if o != nil && o.Machine != nil {
		return true
	}

	return false
}

// SetMachine gets a reference to the given string and assigns it to the Machine field.
func (o *HostMeta) SetMachine(v string) {
	o.Machine = &v
}

// GetNixV returns the NixV field value if set, zero value otherwise.
func (o *HostMeta) GetNixV() []string {
	if o == nil || o.NixV == nil {
		var ret []string
		return ret
	}
	return o.NixV
}

// GetNixVOk returns a tuple with the NixV field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetNixVOk() (*[]string, bool) {
	if o == nil || o.NixV == nil {
		return nil, false
	}
	return &o.NixV, true
}

// HasNixV returns a boolean if a field has been set.
func (o *HostMeta) HasNixV() bool {
	if o != nil && o.NixV != nil {
		return true
	}

	return false
}

// SetNixV gets a reference to the given []string and assigns it to the NixV field.
func (o *HostMeta) SetNixV(v []string) {
	o.NixV = v
}

// GetPlatform returns the Platform field value if set, zero value otherwise.
func (o *HostMeta) GetPlatform() string {
	if o == nil || o.Platform == nil {
		var ret string
		return ret
	}
	return *o.Platform
}

// GetPlatformOk returns a tuple with the Platform field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetPlatformOk() (*string, bool) {
	if o == nil || o.Platform == nil {
		return nil, false
	}
	return o.Platform, true
}

// HasPlatform returns a boolean if a field has been set.
func (o *HostMeta) HasPlatform() bool {
	if o != nil && o.Platform != nil {
		return true
	}

	return false
}

// SetPlatform gets a reference to the given string and assigns it to the Platform field.
func (o *HostMeta) SetPlatform(v string) {
	o.Platform = &v
}

// GetProcessor returns the Processor field value if set, zero value otherwise.
func (o *HostMeta) GetProcessor() string {
	if o == nil || o.Processor == nil {
		var ret string
		return ret
	}
	return *o.Processor
}

// GetProcessorOk returns a tuple with the Processor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetProcessorOk() (*string, bool) {
	if o == nil || o.Processor == nil {
		return nil, false
	}
	return o.Processor, true
}

// HasProcessor returns a boolean if a field has been set.
func (o *HostMeta) HasProcessor() bool {
	if o != nil && o.Processor != nil {
		return true
	}

	return false
}

// SetProcessor gets a reference to the given string and assigns it to the Processor field.
func (o *HostMeta) SetProcessor(v string) {
	o.Processor = &v
}

// GetPythonV returns the PythonV field value if set, zero value otherwise.
func (o *HostMeta) GetPythonV() string {
	if o == nil || o.PythonV == nil {
		var ret string
		return ret
	}
	return *o.PythonV
}

// GetPythonVOk returns a tuple with the PythonV field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetPythonVOk() (*string, bool) {
	if o == nil || o.PythonV == nil {
		return nil, false
	}
	return o.PythonV, true
}

// HasPythonV returns a boolean if a field has been set.
func (o *HostMeta) HasPythonV() bool {
	if o != nil && o.PythonV != nil {
		return true
	}

	return false
}

// SetPythonV gets a reference to the given string and assigns it to the PythonV field.
func (o *HostMeta) SetPythonV(v string) {
	o.PythonV = &v
}

// GetSocketFqdn returns the SocketFqdn field value if set, zero value otherwise.
func (o *HostMeta) GetSocketFqdn() string {
	if o == nil || o.SocketFqdn == nil {
		var ret string
		return ret
	}
	return *o.SocketFqdn
}

// GetSocketFqdnOk returns a tuple with the SocketFqdn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetSocketFqdnOk() (*string, bool) {
	if o == nil || o.SocketFqdn == nil {
		return nil, false
	}
	return o.SocketFqdn, true
}

// HasSocketFqdn returns a boolean if a field has been set.
func (o *HostMeta) HasSocketFqdn() bool {
	if o != nil && o.SocketFqdn != nil {
		return true
	}

	return false
}

// SetSocketFqdn gets a reference to the given string and assigns it to the SocketFqdn field.
func (o *HostMeta) SetSocketFqdn(v string) {
	o.SocketFqdn = &v
}

// GetSocketHostname returns the SocketHostname field value if set, zero value otherwise.
func (o *HostMeta) GetSocketHostname() string {
	if o == nil || o.SocketHostname == nil {
		var ret string
		return ret
	}
	return *o.SocketHostname
}

// GetSocketHostnameOk returns a tuple with the SocketHostname field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetSocketHostnameOk() (*string, bool) {
	if o == nil || o.SocketHostname == nil {
		return nil, false
	}
	return o.SocketHostname, true
}

// HasSocketHostname returns a boolean if a field has been set.
func (o *HostMeta) HasSocketHostname() bool {
	if o != nil && o.SocketHostname != nil {
		return true
	}

	return false
}

// SetSocketHostname gets a reference to the given string and assigns it to the SocketHostname field.
func (o *HostMeta) SetSocketHostname(v string) {
	o.SocketHostname = &v
}

// GetWinV returns the WinV field value if set, zero value otherwise.
func (o *HostMeta) GetWinV() []string {
	if o == nil || o.WinV == nil {
		var ret []string
		return ret
	}
	return o.WinV
}

// GetWinVOk returns a tuple with the WinV field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMeta) GetWinVOk() (*[]string, bool) {
	if o == nil || o.WinV == nil {
		return nil, false
	}
	return &o.WinV, true
}

// HasWinV returns a boolean if a field has been set.
func (o *HostMeta) HasWinV() bool {
	if o != nil && o.WinV != nil {
		return true
	}

	return false
}

// SetWinV gets a reference to the given []string and assigns it to the WinV field.
func (o *HostMeta) SetWinV(v []string) {
	o.WinV = v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AgentChecks != nil {
		toSerialize["agent_checks"] = o.AgentChecks
	}
	if o.AgentVersion != nil {
		toSerialize["agent_version"] = o.AgentVersion
	}
	if o.CpuCores != nil {
		toSerialize["cpuCores"] = o.CpuCores
	}
	if o.FbsdV != nil {
		toSerialize["fbsdV"] = o.FbsdV
	}
	if o.Gohai != nil {
		toSerialize["gohai"] = o.Gohai
	}
	if o.InstallMethod != nil {
		toSerialize["install_method"] = o.InstallMethod
	}
	if o.MacV != nil {
		toSerialize["macV"] = o.MacV
	}
	if o.Machine != nil {
		toSerialize["machine"] = o.Machine
	}
	if o.NixV != nil {
		toSerialize["nixV"] = o.NixV
	}
	if o.Platform != nil {
		toSerialize["platform"] = o.Platform
	}
	if o.Processor != nil {
		toSerialize["processor"] = o.Processor
	}
	if o.PythonV != nil {
		toSerialize["pythonV"] = o.PythonV
	}
	if o.SocketFqdn != nil {
		toSerialize["socket-fqdn"] = o.SocketFqdn
	}
	if o.SocketHostname != nil {
		toSerialize["socket-hostname"] = o.SocketHostname
	}
	if o.WinV != nil {
		toSerialize["winV"] = o.WinV
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AgentChecks    [][]interface{}        `json:"agent_checks,omitempty"`
		AgentVersion   *string                `json:"agent_version,omitempty"`
		CpuCores       *int64                 `json:"cpuCores,omitempty"`
		FbsdV          []string               `json:"fbsdV,omitempty"`
		Gohai          *string                `json:"gohai,omitempty"`
		InstallMethod  *HostMetaInstallMethod `json:"install_method,omitempty"`
		MacV           []string               `json:"macV,omitempty"`
		Machine        *string                `json:"machine,omitempty"`
		NixV           []string               `json:"nixV,omitempty"`
		Platform       *string                `json:"platform,omitempty"`
		Processor      *string                `json:"processor,omitempty"`
		PythonV        *string                `json:"pythonV,omitempty"`
		SocketFqdn     *string                `json:"socket-fqdn,omitempty"`
		SocketHostname *string                `json:"socket-hostname,omitempty"`
		WinV           []string               `json:"winV,omitempty"`
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
	o.AgentChecks = all.AgentChecks
	o.AgentVersion = all.AgentVersion
	o.CpuCores = all.CpuCores
	o.FbsdV = all.FbsdV
	o.Gohai = all.Gohai
	if all.InstallMethod != nil && all.InstallMethod.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InstallMethod = all.InstallMethod
	o.MacV = all.MacV
	o.Machine = all.Machine
	o.NixV = all.NixV
	o.Platform = all.Platform
	o.Processor = all.Processor
	o.PythonV = all.PythonV
	o.SocketFqdn = all.SocketFqdn
	o.SocketHostname = all.SocketHostname
	o.WinV = all.WinV
	return nil
}

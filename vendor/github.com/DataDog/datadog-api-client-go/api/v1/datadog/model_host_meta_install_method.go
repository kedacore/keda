// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// HostMetaInstallMethod Agent install method.
type HostMetaInstallMethod struct {
	// The installer version.
	InstallerVersion *string `json:"installer_version,omitempty"`
	// Tool used to install the agent.
	Tool *string `json:"tool,omitempty"`
	// The tool version.
	ToolVersion *string `json:"tool_version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHostMetaInstallMethod instantiates a new HostMetaInstallMethod object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHostMetaInstallMethod() *HostMetaInstallMethod {
	this := HostMetaInstallMethod{}
	return &this
}

// NewHostMetaInstallMethodWithDefaults instantiates a new HostMetaInstallMethod object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHostMetaInstallMethodWithDefaults() *HostMetaInstallMethod {
	this := HostMetaInstallMethod{}
	return &this
}

// GetInstallerVersion returns the InstallerVersion field value if set, zero value otherwise.
func (o *HostMetaInstallMethod) GetInstallerVersion() string {
	if o == nil || o.InstallerVersion == nil {
		var ret string
		return ret
	}
	return *o.InstallerVersion
}

// GetInstallerVersionOk returns a tuple with the InstallerVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetaInstallMethod) GetInstallerVersionOk() (*string, bool) {
	if o == nil || o.InstallerVersion == nil {
		return nil, false
	}
	return o.InstallerVersion, true
}

// HasInstallerVersion returns a boolean if a field has been set.
func (o *HostMetaInstallMethod) HasInstallerVersion() bool {
	if o != nil && o.InstallerVersion != nil {
		return true
	}

	return false
}

// SetInstallerVersion gets a reference to the given string and assigns it to the InstallerVersion field.
func (o *HostMetaInstallMethod) SetInstallerVersion(v string) {
	o.InstallerVersion = &v
}

// GetTool returns the Tool field value if set, zero value otherwise.
func (o *HostMetaInstallMethod) GetTool() string {
	if o == nil || o.Tool == nil {
		var ret string
		return ret
	}
	return *o.Tool
}

// GetToolOk returns a tuple with the Tool field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetaInstallMethod) GetToolOk() (*string, bool) {
	if o == nil || o.Tool == nil {
		return nil, false
	}
	return o.Tool, true
}

// HasTool returns a boolean if a field has been set.
func (o *HostMetaInstallMethod) HasTool() bool {
	if o != nil && o.Tool != nil {
		return true
	}

	return false
}

// SetTool gets a reference to the given string and assigns it to the Tool field.
func (o *HostMetaInstallMethod) SetTool(v string) {
	o.Tool = &v
}

// GetToolVersion returns the ToolVersion field value if set, zero value otherwise.
func (o *HostMetaInstallMethod) GetToolVersion() string {
	if o == nil || o.ToolVersion == nil {
		var ret string
		return ret
	}
	return *o.ToolVersion
}

// GetToolVersionOk returns a tuple with the ToolVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HostMetaInstallMethod) GetToolVersionOk() (*string, bool) {
	if o == nil || o.ToolVersion == nil {
		return nil, false
	}
	return o.ToolVersion, true
}

// HasToolVersion returns a boolean if a field has been set.
func (o *HostMetaInstallMethod) HasToolVersion() bool {
	if o != nil && o.ToolVersion != nil {
		return true
	}

	return false
}

// SetToolVersion gets a reference to the given string and assigns it to the ToolVersion field.
func (o *HostMetaInstallMethod) SetToolVersion(v string) {
	o.ToolVersion = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HostMetaInstallMethod) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.InstallerVersion != nil {
		toSerialize["installer_version"] = o.InstallerVersion
	}
	if o.Tool != nil {
		toSerialize["tool"] = o.Tool
	}
	if o.ToolVersion != nil {
		toSerialize["tool_version"] = o.ToolVersion
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HostMetaInstallMethod) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		InstallerVersion *string `json:"installer_version,omitempty"`
		Tool             *string `json:"tool,omitempty"`
		ToolVersion      *string `json:"tool_version,omitempty"`
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
	o.InstallerVersion = all.InstallerVersion
	o.Tool = all.Tool
	o.ToolVersion = all.ToolVersion
	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserTest Object containing details about a Synthetic browser test.
type SyntheticsBrowserTest struct {
	// Configuration object for a Synthetic browser test.
	Config SyntheticsBrowserTestConfig `json:"config"`
	// Array of locations used to run the test.
	Locations []string `json:"locations"`
	// Notification message associated with the test. Message can either be text or an empty string.
	Message string `json:"message"`
	// The associated monitor ID.
	MonitorId *int64 `json:"monitor_id,omitempty"`
	// Name of the test.
	Name string `json:"name"`
	// Object describing the extra options for a Synthetic test.
	Options SyntheticsTestOptions `json:"options"`
	// The public ID of the test.
	PublicId *string `json:"public_id,omitempty"`
	// Define whether you want to start (`live`) or pause (`paused`) a
	// Synthetic test.
	Status *SyntheticsTestPauseStatus `json:"status,omitempty"`
	// The steps of the test.
	Steps []SyntheticsStep `json:"steps,omitempty"`
	// Array of tags attached to the test.
	Tags []string `json:"tags,omitempty"`
	// Type of the Synthetic test, `browser`.
	Type SyntheticsBrowserTestType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserTest instantiates a new SyntheticsBrowserTest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserTest(config SyntheticsBrowserTestConfig, locations []string, message string, name string, options SyntheticsTestOptions, typeVar SyntheticsBrowserTestType) *SyntheticsBrowserTest {
	this := SyntheticsBrowserTest{}
	this.Config = config
	this.Locations = locations
	this.Message = message
	this.Name = name
	this.Options = options
	this.Type = typeVar
	return &this
}

// NewSyntheticsBrowserTestWithDefaults instantiates a new SyntheticsBrowserTest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserTestWithDefaults() *SyntheticsBrowserTest {
	this := SyntheticsBrowserTest{}
	var typeVar SyntheticsBrowserTestType = SYNTHETICSBROWSERTESTTYPE_BROWSER
	this.Type = typeVar
	return &this
}

// GetConfig returns the Config field value.
func (o *SyntheticsBrowserTest) GetConfig() SyntheticsBrowserTestConfig {
	if o == nil {
		var ret SyntheticsBrowserTestConfig
		return ret
	}
	return o.Config
}

// GetConfigOk returns a tuple with the Config field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetConfigOk() (*SyntheticsBrowserTestConfig, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Config, true
}

// SetConfig sets field value.
func (o *SyntheticsBrowserTest) SetConfig(v SyntheticsBrowserTestConfig) {
	o.Config = v
}

// GetLocations returns the Locations field value.
func (o *SyntheticsBrowserTest) GetLocations() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Locations
}

// GetLocationsOk returns a tuple with the Locations field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetLocationsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Locations, true
}

// SetLocations sets field value.
func (o *SyntheticsBrowserTest) SetLocations(v []string) {
	o.Locations = v
}

// GetMessage returns the Message field value.
func (o *SyntheticsBrowserTest) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *SyntheticsBrowserTest) SetMessage(v string) {
	o.Message = v
}

// GetMonitorId returns the MonitorId field value if set, zero value otherwise.
func (o *SyntheticsBrowserTest) GetMonitorId() int64 {
	if o == nil || o.MonitorId == nil {
		var ret int64
		return ret
	}
	return *o.MonitorId
}

// GetMonitorIdOk returns a tuple with the MonitorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetMonitorIdOk() (*int64, bool) {
	if o == nil || o.MonitorId == nil {
		return nil, false
	}
	return o.MonitorId, true
}

// HasMonitorId returns a boolean if a field has been set.
func (o *SyntheticsBrowserTest) HasMonitorId() bool {
	if o != nil && o.MonitorId != nil {
		return true
	}

	return false
}

// SetMonitorId gets a reference to the given int64 and assigns it to the MonitorId field.
func (o *SyntheticsBrowserTest) SetMonitorId(v int64) {
	o.MonitorId = &v
}

// GetName returns the Name field value.
func (o *SyntheticsBrowserTest) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsBrowserTest) SetName(v string) {
	o.Name = v
}

// GetOptions returns the Options field value.
func (o *SyntheticsBrowserTest) GetOptions() SyntheticsTestOptions {
	if o == nil {
		var ret SyntheticsTestOptions
		return ret
	}
	return o.Options
}

// GetOptionsOk returns a tuple with the Options field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetOptionsOk() (*SyntheticsTestOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Options, true
}

// SetOptions sets field value.
func (o *SyntheticsBrowserTest) SetOptions(v SyntheticsTestOptions) {
	o.Options = v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *SyntheticsBrowserTest) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *SyntheticsBrowserTest) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *SyntheticsBrowserTest) SetPublicId(v string) {
	o.PublicId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsBrowserTest) GetStatus() SyntheticsTestPauseStatus {
	if o == nil || o.Status == nil {
		var ret SyntheticsTestPauseStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetStatusOk() (*SyntheticsTestPauseStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsBrowserTest) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given SyntheticsTestPauseStatus and assigns it to the Status field.
func (o *SyntheticsBrowserTest) SetStatus(v SyntheticsTestPauseStatus) {
	o.Status = &v
}

// GetSteps returns the Steps field value if set, zero value otherwise.
func (o *SyntheticsBrowserTest) GetSteps() []SyntheticsStep {
	if o == nil || o.Steps == nil {
		var ret []SyntheticsStep
		return ret
	}
	return o.Steps
}

// GetStepsOk returns a tuple with the Steps field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetStepsOk() (*[]SyntheticsStep, bool) {
	if o == nil || o.Steps == nil {
		return nil, false
	}
	return &o.Steps, true
}

// HasSteps returns a boolean if a field has been set.
func (o *SyntheticsBrowserTest) HasSteps() bool {
	if o != nil && o.Steps != nil {
		return true
	}

	return false
}

// SetSteps gets a reference to the given []SyntheticsStep and assigns it to the Steps field.
func (o *SyntheticsBrowserTest) SetSteps(v []SyntheticsStep) {
	o.Steps = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SyntheticsBrowserTest) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SyntheticsBrowserTest) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SyntheticsBrowserTest) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value.
func (o *SyntheticsBrowserTest) GetType() SyntheticsBrowserTestType {
	if o == nil {
		var ret SyntheticsBrowserTestType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserTest) GetTypeOk() (*SyntheticsBrowserTestType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsBrowserTest) SetType(v SyntheticsBrowserTestType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserTest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["config"] = o.Config
	toSerialize["locations"] = o.Locations
	toSerialize["message"] = o.Message
	if o.MonitorId != nil {
		toSerialize["monitor_id"] = o.MonitorId
	}
	toSerialize["name"] = o.Name
	toSerialize["options"] = o.Options
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Steps != nil {
		toSerialize["steps"] = o.Steps
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserTest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Config    *SyntheticsBrowserTestConfig `json:"config"`
		Locations *[]string                    `json:"locations"`
		Message   *string                      `json:"message"`
		Name      *string                      `json:"name"`
		Options   *SyntheticsTestOptions       `json:"options"`
		Type      *SyntheticsBrowserTestType   `json:"type"`
	}{}
	all := struct {
		Config    SyntheticsBrowserTestConfig `json:"config"`
		Locations []string                    `json:"locations"`
		Message   string                      `json:"message"`
		MonitorId *int64                      `json:"monitor_id,omitempty"`
		Name      string                      `json:"name"`
		Options   SyntheticsTestOptions       `json:"options"`
		PublicId  *string                     `json:"public_id,omitempty"`
		Status    *SyntheticsTestPauseStatus  `json:"status,omitempty"`
		Steps     []SyntheticsStep            `json:"steps,omitempty"`
		Tags      []string                    `json:"tags,omitempty"`
		Type      SyntheticsBrowserTestType   `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Config == nil {
		return fmt.Errorf("Required field config missing")
	}
	if required.Locations == nil {
		return fmt.Errorf("Required field locations missing")
	}
	if required.Message == nil {
		return fmt.Errorf("Required field message missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Options == nil {
		return fmt.Errorf("Required field options missing")
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
	if v := all.Status; v != nil && !v.IsValid() {
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
	if all.Config.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Config = all.Config
	o.Locations = all.Locations
	o.Message = all.Message
	o.MonitorId = all.MonitorId
	o.Name = all.Name
	if all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	o.PublicId = all.PublicId
	o.Status = all.Status
	o.Steps = all.Steps
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}

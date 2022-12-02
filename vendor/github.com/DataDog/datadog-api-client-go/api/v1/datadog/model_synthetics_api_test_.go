// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAPITest Object containing details about a Synthetic API test.
type SyntheticsAPITest struct {
	// Configuration object for a Synthetic API test.
	Config SyntheticsAPITestConfig `json:"config"`
	// Array of locations used to run the test.
	Locations []string `json:"locations"`
	// Notification message associated with the test.
	Message string `json:"message"`
	// The associated monitor ID.
	MonitorId *int64 `json:"monitor_id,omitempty"`
	// Name of the test.
	Name string `json:"name"`
	// Object describing the extra options for a Synthetic test.
	Options SyntheticsTestOptions `json:"options"`
	// The public ID for the test.
	PublicId *string `json:"public_id,omitempty"`
	// Define whether you want to start (`live`) or pause (`paused`) a
	// Synthetic test.
	Status *SyntheticsTestPauseStatus `json:"status,omitempty"`
	// The subtype of the Synthetic API test, `http`, `ssl`, `tcp`,
	// `dns`, `icmp`, `udp`, `websocket`, `grpc` or `multi`.
	Subtype *SyntheticsTestDetailsSubType `json:"subtype,omitempty"`
	// Array of tags attached to the test.
	Tags []string `json:"tags,omitempty"`
	// Type of the Synthetic test, `api`.
	Type SyntheticsAPITestType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAPITest instantiates a new SyntheticsAPITest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAPITest(config SyntheticsAPITestConfig, locations []string, message string, name string, options SyntheticsTestOptions, typeVar SyntheticsAPITestType) *SyntheticsAPITest {
	this := SyntheticsAPITest{}
	this.Config = config
	this.Locations = locations
	this.Message = message
	this.Name = name
	this.Options = options
	this.Type = typeVar
	return &this
}

// NewSyntheticsAPITestWithDefaults instantiates a new SyntheticsAPITest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAPITestWithDefaults() *SyntheticsAPITest {
	this := SyntheticsAPITest{}
	var typeVar SyntheticsAPITestType = SYNTHETICSAPITESTTYPE_API
	this.Type = typeVar
	return &this
}

// GetConfig returns the Config field value.
func (o *SyntheticsAPITest) GetConfig() SyntheticsAPITestConfig {
	if o == nil {
		var ret SyntheticsAPITestConfig
		return ret
	}
	return o.Config
}

// GetConfigOk returns a tuple with the Config field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetConfigOk() (*SyntheticsAPITestConfig, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Config, true
}

// SetConfig sets field value.
func (o *SyntheticsAPITest) SetConfig(v SyntheticsAPITestConfig) {
	o.Config = v
}

// GetLocations returns the Locations field value.
func (o *SyntheticsAPITest) GetLocations() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Locations
}

// GetLocationsOk returns a tuple with the Locations field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetLocationsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Locations, true
}

// SetLocations sets field value.
func (o *SyntheticsAPITest) SetLocations(v []string) {
	o.Locations = v
}

// GetMessage returns the Message field value.
func (o *SyntheticsAPITest) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *SyntheticsAPITest) SetMessage(v string) {
	o.Message = v
}

// GetMonitorId returns the MonitorId field value if set, zero value otherwise.
func (o *SyntheticsAPITest) GetMonitorId() int64 {
	if o == nil || o.MonitorId == nil {
		var ret int64
		return ret
	}
	return *o.MonitorId
}

// GetMonitorIdOk returns a tuple with the MonitorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetMonitorIdOk() (*int64, bool) {
	if o == nil || o.MonitorId == nil {
		return nil, false
	}
	return o.MonitorId, true
}

// HasMonitorId returns a boolean if a field has been set.
func (o *SyntheticsAPITest) HasMonitorId() bool {
	if o != nil && o.MonitorId != nil {
		return true
	}

	return false
}

// SetMonitorId gets a reference to the given int64 and assigns it to the MonitorId field.
func (o *SyntheticsAPITest) SetMonitorId(v int64) {
	o.MonitorId = &v
}

// GetName returns the Name field value.
func (o *SyntheticsAPITest) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsAPITest) SetName(v string) {
	o.Name = v
}

// GetOptions returns the Options field value.
func (o *SyntheticsAPITest) GetOptions() SyntheticsTestOptions {
	if o == nil {
		var ret SyntheticsTestOptions
		return ret
	}
	return o.Options
}

// GetOptionsOk returns a tuple with the Options field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetOptionsOk() (*SyntheticsTestOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Options, true
}

// SetOptions sets field value.
func (o *SyntheticsAPITest) SetOptions(v SyntheticsTestOptions) {
	o.Options = v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *SyntheticsAPITest) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *SyntheticsAPITest) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *SyntheticsAPITest) SetPublicId(v string) {
	o.PublicId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsAPITest) GetStatus() SyntheticsTestPauseStatus {
	if o == nil || o.Status == nil {
		var ret SyntheticsTestPauseStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetStatusOk() (*SyntheticsTestPauseStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsAPITest) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given SyntheticsTestPauseStatus and assigns it to the Status field.
func (o *SyntheticsAPITest) SetStatus(v SyntheticsTestPauseStatus) {
	o.Status = &v
}

// GetSubtype returns the Subtype field value if set, zero value otherwise.
func (o *SyntheticsAPITest) GetSubtype() SyntheticsTestDetailsSubType {
	if o == nil || o.Subtype == nil {
		var ret SyntheticsTestDetailsSubType
		return ret
	}
	return *o.Subtype
}

// GetSubtypeOk returns a tuple with the Subtype field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetSubtypeOk() (*SyntheticsTestDetailsSubType, bool) {
	if o == nil || o.Subtype == nil {
		return nil, false
	}
	return o.Subtype, true
}

// HasSubtype returns a boolean if a field has been set.
func (o *SyntheticsAPITest) HasSubtype() bool {
	if o != nil && o.Subtype != nil {
		return true
	}

	return false
}

// SetSubtype gets a reference to the given SyntheticsTestDetailsSubType and assigns it to the Subtype field.
func (o *SyntheticsAPITest) SetSubtype(v SyntheticsTestDetailsSubType) {
	o.Subtype = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SyntheticsAPITest) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SyntheticsAPITest) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SyntheticsAPITest) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value.
func (o *SyntheticsAPITest) GetType() SyntheticsAPITestType {
	if o == nil {
		var ret SyntheticsAPITestType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAPITest) GetTypeOk() (*SyntheticsAPITestType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsAPITest) SetType(v SyntheticsAPITestType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAPITest) MarshalJSON() ([]byte, error) {
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
	if o.Subtype != nil {
		toSerialize["subtype"] = o.Subtype
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
func (o *SyntheticsAPITest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Config    *SyntheticsAPITestConfig `json:"config"`
		Locations *[]string                `json:"locations"`
		Message   *string                  `json:"message"`
		Name      *string                  `json:"name"`
		Options   *SyntheticsTestOptions   `json:"options"`
		Type      *SyntheticsAPITestType   `json:"type"`
	}{}
	all := struct {
		Config    SyntheticsAPITestConfig       `json:"config"`
		Locations []string                      `json:"locations"`
		Message   string                        `json:"message"`
		MonitorId *int64                        `json:"monitor_id,omitempty"`
		Name      string                        `json:"name"`
		Options   SyntheticsTestOptions         `json:"options"`
		PublicId  *string                       `json:"public_id,omitempty"`
		Status    *SyntheticsTestPauseStatus    `json:"status,omitempty"`
		Subtype   *SyntheticsTestDetailsSubType `json:"subtype,omitempty"`
		Tags      []string                      `json:"tags,omitempty"`
		Type      SyntheticsAPITestType         `json:"type"`
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
	if v := all.Subtype; v != nil && !v.IsValid() {
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
	o.Subtype = all.Subtype
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}

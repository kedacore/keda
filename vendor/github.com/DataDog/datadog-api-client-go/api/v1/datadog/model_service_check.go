// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ServiceCheck An object containing service check and status.
type ServiceCheck struct {
	// The check.
	Check string `json:"check"`
	// The host name correlated with the check.
	HostName string `json:"host_name"`
	// Message containing check status.
	Message *string `json:"message,omitempty"`
	// The status of a service check.
	Status ServiceCheckStatus `json:"status"`
	// Tags related to a check.
	Tags []string `json:"tags"`
	// Time of check.
	Timestamp *int64 `json:"timestamp,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewServiceCheck instantiates a new ServiceCheck object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceCheck(check string, hostName string, status ServiceCheckStatus, tags []string) *ServiceCheck {
	this := ServiceCheck{}
	this.Check = check
	this.HostName = hostName
	this.Status = status
	this.Tags = tags
	return &this
}

// NewServiceCheckWithDefaults instantiates a new ServiceCheck object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceCheckWithDefaults() *ServiceCheck {
	this := ServiceCheck{}
	return &this
}

// GetCheck returns the Check field value.
func (o *ServiceCheck) GetCheck() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Check
}

// GetCheckOk returns a tuple with the Check field value
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetCheckOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Check, true
}

// SetCheck sets field value.
func (o *ServiceCheck) SetCheck(v string) {
	o.Check = v
}

// GetHostName returns the HostName field value.
func (o *ServiceCheck) GetHostName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.HostName
}

// GetHostNameOk returns a tuple with the HostName field value
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetHostNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.HostName, true
}

// SetHostName sets field value.
func (o *ServiceCheck) SetHostName(v string) {
	o.HostName = v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *ServiceCheck) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *ServiceCheck) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *ServiceCheck) SetMessage(v string) {
	o.Message = &v
}

// GetStatus returns the Status field value.
func (o *ServiceCheck) GetStatus() ServiceCheckStatus {
	if o == nil {
		var ret ServiceCheckStatus
		return ret
	}
	return o.Status
}

// GetStatusOk returns a tuple with the Status field value
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetStatusOk() (*ServiceCheckStatus, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Status, true
}

// SetStatus sets field value.
func (o *ServiceCheck) SetStatus(v ServiceCheckStatus) {
	o.Status = v
}

// GetTags returns the Tags field value.
func (o *ServiceCheck) GetTags() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetTagsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Tags, true
}

// SetTags sets field value.
func (o *ServiceCheck) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *ServiceCheck) GetTimestamp() int64 {
	if o == nil || o.Timestamp == nil {
		var ret int64
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceCheck) GetTimestampOk() (*int64, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *ServiceCheck) HasTimestamp() bool {
	if o != nil && o.Timestamp != nil {
		return true
	}

	return false
}

// SetTimestamp gets a reference to the given int64 and assigns it to the Timestamp field.
func (o *ServiceCheck) SetTimestamp(v int64) {
	o.Timestamp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceCheck) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["check"] = o.Check
	toSerialize["host_name"] = o.HostName
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	toSerialize["status"] = o.Status
	toSerialize["tags"] = o.Tags
	if o.Timestamp != nil {
		toSerialize["timestamp"] = o.Timestamp
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceCheck) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Check    *string             `json:"check"`
		HostName *string             `json:"host_name"`
		Status   *ServiceCheckStatus `json:"status"`
		Tags     *[]string           `json:"tags"`
	}{}
	all := struct {
		Check     string             `json:"check"`
		HostName  string             `json:"host_name"`
		Message   *string            `json:"message,omitempty"`
		Status    ServiceCheckStatus `json:"status"`
		Tags      []string           `json:"tags"`
		Timestamp *int64             `json:"timestamp,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Check == nil {
		return fmt.Errorf("Required field check missing")
	}
	if required.HostName == nil {
		return fmt.Errorf("Required field host_name missing")
	}
	if required.Status == nil {
		return fmt.Errorf("Required field status missing")
	}
	if required.Tags == nil {
		return fmt.Errorf("Required field tags missing")
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
	if v := all.Status; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Check = all.Check
	o.HostName = all.HostName
	o.Message = all.Message
	o.Status = all.Status
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	return nil
}

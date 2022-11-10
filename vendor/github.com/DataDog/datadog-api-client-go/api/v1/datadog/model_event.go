// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// Event Object representing an event.
type Event struct {
	// If an alert event is enabled, set its type.
	// For example, `error`, `warning`, `info`, `success`, `user_update`,
	// `recommendation`, and `snapshot`.
	AlertType *EventAlertType `json:"alert_type,omitempty"`
	// POSIX timestamp of the event. Must be sent as an integer (that is no quotes).
	// Limited to events no older than 18 hours.
	DateHappened *int64 `json:"date_happened,omitempty"`
	// A device name.
	DeviceName *string `json:"device_name,omitempty"`
	// Host name to associate with the event.
	// Any tags associated with the host are also applied to this event.
	Host *string `json:"host,omitempty"`
	// Integer ID of the event.
	Id *int64 `json:"id,omitempty"`
	// Handling IDs as large 64-bit numbers can cause loss of accuracy issues with some programming languages.
	// Instead, use the string representation of the Event ID to avoid losing accuracy.
	IdStr *string `json:"id_str,omitempty"`
	// Payload of the event.
	Payload *string `json:"payload,omitempty"`
	// The priority of the event. For example, `normal` or `low`.
	Priority NullableEventPriority `json:"priority,omitempty"`
	// The type of event being posted. Option examples include nagios, hudson, jenkins, my_apps, chef, puppet, git, bitbucket, etc.
	// The list of standard source attribute values [available here](https://docs.datadoghq.com/integrations/faq/list-of-api-source-attribute-value).
	SourceTypeName *string `json:"source_type_name,omitempty"`
	// A list of tags to apply to the event.
	Tags []string `json:"tags,omitempty"`
	// The body of the event. Limited to 4000 characters. The text supports markdown.
	// To use markdown in the event text, start the text block with `%%% \n` and end the text block with `\n %%%`.
	// Use `msg_text` with the Datadog Ruby library.
	Text *string `json:"text,omitempty"`
	// The event title.
	Title *string `json:"title,omitempty"`
	// URL of the event.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewEvent instantiates a new Event object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEvent() *Event {
	this := Event{}
	return &this
}

// NewEventWithDefaults instantiates a new Event object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventWithDefaults() *Event {
	this := Event{}
	return &this
}

// GetAlertType returns the AlertType field value if set, zero value otherwise.
func (o *Event) GetAlertType() EventAlertType {
	if o == nil || o.AlertType == nil {
		var ret EventAlertType
		return ret
	}
	return *o.AlertType
}

// GetAlertTypeOk returns a tuple with the AlertType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetAlertTypeOk() (*EventAlertType, bool) {
	if o == nil || o.AlertType == nil {
		return nil, false
	}
	return o.AlertType, true
}

// HasAlertType returns a boolean if a field has been set.
func (o *Event) HasAlertType() bool {
	if o != nil && o.AlertType != nil {
		return true
	}

	return false
}

// SetAlertType gets a reference to the given EventAlertType and assigns it to the AlertType field.
func (o *Event) SetAlertType(v EventAlertType) {
	o.AlertType = &v
}

// GetDateHappened returns the DateHappened field value if set, zero value otherwise.
func (o *Event) GetDateHappened() int64 {
	if o == nil || o.DateHappened == nil {
		var ret int64
		return ret
	}
	return *o.DateHappened
}

// GetDateHappenedOk returns a tuple with the DateHappened field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetDateHappenedOk() (*int64, bool) {
	if o == nil || o.DateHappened == nil {
		return nil, false
	}
	return o.DateHappened, true
}

// HasDateHappened returns a boolean if a field has been set.
func (o *Event) HasDateHappened() bool {
	if o != nil && o.DateHappened != nil {
		return true
	}

	return false
}

// SetDateHappened gets a reference to the given int64 and assigns it to the DateHappened field.
func (o *Event) SetDateHappened(v int64) {
	o.DateHappened = &v
}

// GetDeviceName returns the DeviceName field value if set, zero value otherwise.
func (o *Event) GetDeviceName() string {
	if o == nil || o.DeviceName == nil {
		var ret string
		return ret
	}
	return *o.DeviceName
}

// GetDeviceNameOk returns a tuple with the DeviceName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetDeviceNameOk() (*string, bool) {
	if o == nil || o.DeviceName == nil {
		return nil, false
	}
	return o.DeviceName, true
}

// HasDeviceName returns a boolean if a field has been set.
func (o *Event) HasDeviceName() bool {
	if o != nil && o.DeviceName != nil {
		return true
	}

	return false
}

// SetDeviceName gets a reference to the given string and assigns it to the DeviceName field.
func (o *Event) SetDeviceName(v string) {
	o.DeviceName = &v
}

// GetHost returns the Host field value if set, zero value otherwise.
func (o *Event) GetHost() string {
	if o == nil || o.Host == nil {
		var ret string
		return ret
	}
	return *o.Host
}

// GetHostOk returns a tuple with the Host field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetHostOk() (*string, bool) {
	if o == nil || o.Host == nil {
		return nil, false
	}
	return o.Host, true
}

// HasHost returns a boolean if a field has been set.
func (o *Event) HasHost() bool {
	if o != nil && o.Host != nil {
		return true
	}

	return false
}

// SetHost gets a reference to the given string and assigns it to the Host field.
func (o *Event) SetHost(v string) {
	o.Host = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *Event) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *Event) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *Event) SetId(v int64) {
	o.Id = &v
}

// GetIdStr returns the IdStr field value if set, zero value otherwise.
func (o *Event) GetIdStr() string {
	if o == nil || o.IdStr == nil {
		var ret string
		return ret
	}
	return *o.IdStr
}

// GetIdStrOk returns a tuple with the IdStr field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetIdStrOk() (*string, bool) {
	if o == nil || o.IdStr == nil {
		return nil, false
	}
	return o.IdStr, true
}

// HasIdStr returns a boolean if a field has been set.
func (o *Event) HasIdStr() bool {
	if o != nil && o.IdStr != nil {
		return true
	}

	return false
}

// SetIdStr gets a reference to the given string and assigns it to the IdStr field.
func (o *Event) SetIdStr(v string) {
	o.IdStr = &v
}

// GetPayload returns the Payload field value if set, zero value otherwise.
func (o *Event) GetPayload() string {
	if o == nil || o.Payload == nil {
		var ret string
		return ret
	}
	return *o.Payload
}

// GetPayloadOk returns a tuple with the Payload field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetPayloadOk() (*string, bool) {
	if o == nil || o.Payload == nil {
		return nil, false
	}
	return o.Payload, true
}

// HasPayload returns a boolean if a field has been set.
func (o *Event) HasPayload() bool {
	if o != nil && o.Payload != nil {
		return true
	}

	return false
}

// SetPayload gets a reference to the given string and assigns it to the Payload field.
func (o *Event) SetPayload(v string) {
	o.Payload = &v
}

// GetPriority returns the Priority field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Event) GetPriority() EventPriority {
	if o == nil || o.Priority.Get() == nil {
		var ret EventPriority
		return ret
	}
	return *o.Priority.Get()
}

// GetPriorityOk returns a tuple with the Priority field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *Event) GetPriorityOk() (*EventPriority, bool) {
	if o == nil {
		return nil, false
	}
	return o.Priority.Get(), o.Priority.IsSet()
}

// HasPriority returns a boolean if a field has been set.
func (o *Event) HasPriority() bool {
	if o != nil && o.Priority.IsSet() {
		return true
	}

	return false
}

// SetPriority gets a reference to the given NullableEventPriority and assigns it to the Priority field.
func (o *Event) SetPriority(v EventPriority) {
	o.Priority.Set(&v)
}

// SetPriorityNil sets the value for Priority to be an explicit nil.
func (o *Event) SetPriorityNil() {
	o.Priority.Set(nil)
}

// UnsetPriority ensures that no value is present for Priority, not even an explicit nil.
func (o *Event) UnsetPriority() {
	o.Priority.Unset()
}

// GetSourceTypeName returns the SourceTypeName field value if set, zero value otherwise.
func (o *Event) GetSourceTypeName() string {
	if o == nil || o.SourceTypeName == nil {
		var ret string
		return ret
	}
	return *o.SourceTypeName
}

// GetSourceTypeNameOk returns a tuple with the SourceTypeName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetSourceTypeNameOk() (*string, bool) {
	if o == nil || o.SourceTypeName == nil {
		return nil, false
	}
	return o.SourceTypeName, true
}

// HasSourceTypeName returns a boolean if a field has been set.
func (o *Event) HasSourceTypeName() bool {
	if o != nil && o.SourceTypeName != nil {
		return true
	}

	return false
}

// SetSourceTypeName gets a reference to the given string and assigns it to the SourceTypeName field.
func (o *Event) SetSourceTypeName(v string) {
	o.SourceTypeName = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *Event) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *Event) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *Event) SetTags(v []string) {
	o.Tags = v
}

// GetText returns the Text field value if set, zero value otherwise.
func (o *Event) GetText() string {
	if o == nil || o.Text == nil {
		var ret string
		return ret
	}
	return *o.Text
}

// GetTextOk returns a tuple with the Text field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetTextOk() (*string, bool) {
	if o == nil || o.Text == nil {
		return nil, false
	}
	return o.Text, true
}

// HasText returns a boolean if a field has been set.
func (o *Event) HasText() bool {
	if o != nil && o.Text != nil {
		return true
	}

	return false
}

// SetText gets a reference to the given string and assigns it to the Text field.
func (o *Event) SetText(v string) {
	o.Text = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *Event) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *Event) HasTitle() bool {
	if o != nil && o.Title != nil {
		return true
	}

	return false
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *Event) SetTitle(v string) {
	o.Title = &v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *Event) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Event) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *Event) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *Event) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o Event) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AlertType != nil {
		toSerialize["alert_type"] = o.AlertType
	}
	if o.DateHappened != nil {
		toSerialize["date_happened"] = o.DateHappened
	}
	if o.DeviceName != nil {
		toSerialize["device_name"] = o.DeviceName
	}
	if o.Host != nil {
		toSerialize["host"] = o.Host
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IdStr != nil {
		toSerialize["id_str"] = o.IdStr
	}
	if o.Payload != nil {
		toSerialize["payload"] = o.Payload
	}
	if o.Priority.IsSet() {
		toSerialize["priority"] = o.Priority.Get()
	}
	if o.SourceTypeName != nil {
		toSerialize["source_type_name"] = o.SourceTypeName
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Text != nil {
		toSerialize["text"] = o.Text
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *Event) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AlertType      *EventAlertType       `json:"alert_type,omitempty"`
		DateHappened   *int64                `json:"date_happened,omitempty"`
		DeviceName     *string               `json:"device_name,omitempty"`
		Host           *string               `json:"host,omitempty"`
		Id             *int64                `json:"id,omitempty"`
		IdStr          *string               `json:"id_str,omitempty"`
		Payload        *string               `json:"payload,omitempty"`
		Priority       NullableEventPriority `json:"priority,omitempty"`
		SourceTypeName *string               `json:"source_type_name,omitempty"`
		Tags           []string              `json:"tags,omitempty"`
		Text           *string               `json:"text,omitempty"`
		Title          *string               `json:"title,omitempty"`
		Url            *string               `json:"url,omitempty"`
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
	if v := all.AlertType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Priority; v.Get() != nil && !v.Get().IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AlertType = all.AlertType
	o.DateHappened = all.DateHappened
	o.DeviceName = all.DeviceName
	o.Host = all.Host
	o.Id = all.Id
	o.IdStr = all.IdStr
	o.Payload = all.Payload
	o.Priority = all.Priority
	o.SourceTypeName = all.SourceTypeName
	o.Tags = all.Tags
	o.Text = all.Text
	o.Title = all.Title
	o.Url = all.Url
	return nil
}

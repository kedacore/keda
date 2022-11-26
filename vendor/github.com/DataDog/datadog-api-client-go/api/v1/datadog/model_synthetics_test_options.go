// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestOptions Object describing the extra options for a Synthetic test.
type SyntheticsTestOptions struct {
	// For SSL test, whether or not the test should allow self signed
	// certificates.
	AcceptSelfSigned *bool `json:"accept_self_signed,omitempty"`
	// Allows loading insecure content for an HTTP request.
	AllowInsecure *bool `json:"allow_insecure,omitempty"`
	// For SSL test, whether or not the test should fail on revoked certificate in stapled OCSP.
	CheckCertificateRevocation *bool `json:"checkCertificateRevocation,omitempty"`
	// CI/CD options for a Synthetic test.
	Ci *SyntheticsTestCiOptions `json:"ci,omitempty"`
	// For browser test, array with the different device IDs used to run the test.
	DeviceIds []SyntheticsDeviceID `json:"device_ids,omitempty"`
	// Whether or not to disable CORS mechanism.
	DisableCors *bool `json:"disableCors,omitempty"`
	// For API HTTP test, whether or not the test should follow redirects.
	FollowRedirects *bool `json:"follow_redirects,omitempty"`
	// Minimum amount of time in failure required to trigger an alert.
	MinFailureDuration *int64 `json:"min_failure_duration,omitempty"`
	// Minimum number of locations in failure required to trigger
	// an alert.
	MinLocationFailed *int64 `json:"min_location_failed,omitempty"`
	// The monitor name is used for the alert title as well as for all monitor dashboard widgets and SLOs.
	MonitorName *string `json:"monitor_name,omitempty"`
	// Object containing the options for a Synthetic test as a monitor
	// (for example, renotification).
	MonitorOptions *SyntheticsTestOptionsMonitorOptions `json:"monitor_options,omitempty"`
	// Integer from 1 (high) to 5 (low) indicating alert severity.
	MonitorPriority *int32 `json:"monitor_priority,omitempty"`
	// Prevents saving screenshots of the steps.
	NoScreenshot *bool `json:"noScreenshot,omitempty"`
	// A list of role identifiers that can be pulled from the Roles API, for restricting read and write access.
	RestrictedRoles []string `json:"restricted_roles,omitempty"`
	// Object describing the retry strategy to apply to a Synthetic test.
	Retry *SyntheticsTestOptionsRetry `json:"retry,omitempty"`
	// The RUM data collection settings for the Synthetic browser test.
	// **Note:** There are 3 ways to format RUM settings:
	//
	// `{ isEnabled: false }`
	// RUM data is not collected.
	//
	// `{ isEnabled: true }`
	// RUM data is collected from the Synthetic test's default application.
	//
	// `{ isEnabled: true, applicationId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", clientTokenId: 12345 }`
	// RUM data is collected using the specified application.
	RumSettings *SyntheticsBrowserTestRumSettings `json:"rumSettings,omitempty"`
	// The frequency at which to run the Synthetic test (in seconds).
	TickEvery *int64 `json:"tick_every,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestOptions instantiates a new SyntheticsTestOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestOptions() *SyntheticsTestOptions {
	this := SyntheticsTestOptions{}
	return &this
}

// NewSyntheticsTestOptionsWithDefaults instantiates a new SyntheticsTestOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestOptionsWithDefaults() *SyntheticsTestOptions {
	this := SyntheticsTestOptions{}
	return &this
}

// GetAcceptSelfSigned returns the AcceptSelfSigned field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetAcceptSelfSigned() bool {
	if o == nil || o.AcceptSelfSigned == nil {
		var ret bool
		return ret
	}
	return *o.AcceptSelfSigned
}

// GetAcceptSelfSignedOk returns a tuple with the AcceptSelfSigned field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetAcceptSelfSignedOk() (*bool, bool) {
	if o == nil || o.AcceptSelfSigned == nil {
		return nil, false
	}
	return o.AcceptSelfSigned, true
}

// HasAcceptSelfSigned returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasAcceptSelfSigned() bool {
	if o != nil && o.AcceptSelfSigned != nil {
		return true
	}

	return false
}

// SetAcceptSelfSigned gets a reference to the given bool and assigns it to the AcceptSelfSigned field.
func (o *SyntheticsTestOptions) SetAcceptSelfSigned(v bool) {
	o.AcceptSelfSigned = &v
}

// GetAllowInsecure returns the AllowInsecure field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetAllowInsecure() bool {
	if o == nil || o.AllowInsecure == nil {
		var ret bool
		return ret
	}
	return *o.AllowInsecure
}

// GetAllowInsecureOk returns a tuple with the AllowInsecure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetAllowInsecureOk() (*bool, bool) {
	if o == nil || o.AllowInsecure == nil {
		return nil, false
	}
	return o.AllowInsecure, true
}

// HasAllowInsecure returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasAllowInsecure() bool {
	if o != nil && o.AllowInsecure != nil {
		return true
	}

	return false
}

// SetAllowInsecure gets a reference to the given bool and assigns it to the AllowInsecure field.
func (o *SyntheticsTestOptions) SetAllowInsecure(v bool) {
	o.AllowInsecure = &v
}

// GetCheckCertificateRevocation returns the CheckCertificateRevocation field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetCheckCertificateRevocation() bool {
	if o == nil || o.CheckCertificateRevocation == nil {
		var ret bool
		return ret
	}
	return *o.CheckCertificateRevocation
}

// GetCheckCertificateRevocationOk returns a tuple with the CheckCertificateRevocation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetCheckCertificateRevocationOk() (*bool, bool) {
	if o == nil || o.CheckCertificateRevocation == nil {
		return nil, false
	}
	return o.CheckCertificateRevocation, true
}

// HasCheckCertificateRevocation returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasCheckCertificateRevocation() bool {
	if o != nil && o.CheckCertificateRevocation != nil {
		return true
	}

	return false
}

// SetCheckCertificateRevocation gets a reference to the given bool and assigns it to the CheckCertificateRevocation field.
func (o *SyntheticsTestOptions) SetCheckCertificateRevocation(v bool) {
	o.CheckCertificateRevocation = &v
}

// GetCi returns the Ci field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetCi() SyntheticsTestCiOptions {
	if o == nil || o.Ci == nil {
		var ret SyntheticsTestCiOptions
		return ret
	}
	return *o.Ci
}

// GetCiOk returns a tuple with the Ci field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetCiOk() (*SyntheticsTestCiOptions, bool) {
	if o == nil || o.Ci == nil {
		return nil, false
	}
	return o.Ci, true
}

// HasCi returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasCi() bool {
	if o != nil && o.Ci != nil {
		return true
	}

	return false
}

// SetCi gets a reference to the given SyntheticsTestCiOptions and assigns it to the Ci field.
func (o *SyntheticsTestOptions) SetCi(v SyntheticsTestCiOptions) {
	o.Ci = &v
}

// GetDeviceIds returns the DeviceIds field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetDeviceIds() []SyntheticsDeviceID {
	if o == nil || o.DeviceIds == nil {
		var ret []SyntheticsDeviceID
		return ret
	}
	return o.DeviceIds
}

// GetDeviceIdsOk returns a tuple with the DeviceIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetDeviceIdsOk() (*[]SyntheticsDeviceID, bool) {
	if o == nil || o.DeviceIds == nil {
		return nil, false
	}
	return &o.DeviceIds, true
}

// HasDeviceIds returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasDeviceIds() bool {
	if o != nil && o.DeviceIds != nil {
		return true
	}

	return false
}

// SetDeviceIds gets a reference to the given []SyntheticsDeviceID and assigns it to the DeviceIds field.
func (o *SyntheticsTestOptions) SetDeviceIds(v []SyntheticsDeviceID) {
	o.DeviceIds = v
}

// GetDisableCors returns the DisableCors field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetDisableCors() bool {
	if o == nil || o.DisableCors == nil {
		var ret bool
		return ret
	}
	return *o.DisableCors
}

// GetDisableCorsOk returns a tuple with the DisableCors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetDisableCorsOk() (*bool, bool) {
	if o == nil || o.DisableCors == nil {
		return nil, false
	}
	return o.DisableCors, true
}

// HasDisableCors returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasDisableCors() bool {
	if o != nil && o.DisableCors != nil {
		return true
	}

	return false
}

// SetDisableCors gets a reference to the given bool and assigns it to the DisableCors field.
func (o *SyntheticsTestOptions) SetDisableCors(v bool) {
	o.DisableCors = &v
}

// GetFollowRedirects returns the FollowRedirects field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetFollowRedirects() bool {
	if o == nil || o.FollowRedirects == nil {
		var ret bool
		return ret
	}
	return *o.FollowRedirects
}

// GetFollowRedirectsOk returns a tuple with the FollowRedirects field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetFollowRedirectsOk() (*bool, bool) {
	if o == nil || o.FollowRedirects == nil {
		return nil, false
	}
	return o.FollowRedirects, true
}

// HasFollowRedirects returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasFollowRedirects() bool {
	if o != nil && o.FollowRedirects != nil {
		return true
	}

	return false
}

// SetFollowRedirects gets a reference to the given bool and assigns it to the FollowRedirects field.
func (o *SyntheticsTestOptions) SetFollowRedirects(v bool) {
	o.FollowRedirects = &v
}

// GetMinFailureDuration returns the MinFailureDuration field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetMinFailureDuration() int64 {
	if o == nil || o.MinFailureDuration == nil {
		var ret int64
		return ret
	}
	return *o.MinFailureDuration
}

// GetMinFailureDurationOk returns a tuple with the MinFailureDuration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetMinFailureDurationOk() (*int64, bool) {
	if o == nil || o.MinFailureDuration == nil {
		return nil, false
	}
	return o.MinFailureDuration, true
}

// HasMinFailureDuration returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasMinFailureDuration() bool {
	if o != nil && o.MinFailureDuration != nil {
		return true
	}

	return false
}

// SetMinFailureDuration gets a reference to the given int64 and assigns it to the MinFailureDuration field.
func (o *SyntheticsTestOptions) SetMinFailureDuration(v int64) {
	o.MinFailureDuration = &v
}

// GetMinLocationFailed returns the MinLocationFailed field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetMinLocationFailed() int64 {
	if o == nil || o.MinLocationFailed == nil {
		var ret int64
		return ret
	}
	return *o.MinLocationFailed
}

// GetMinLocationFailedOk returns a tuple with the MinLocationFailed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetMinLocationFailedOk() (*int64, bool) {
	if o == nil || o.MinLocationFailed == nil {
		return nil, false
	}
	return o.MinLocationFailed, true
}

// HasMinLocationFailed returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasMinLocationFailed() bool {
	if o != nil && o.MinLocationFailed != nil {
		return true
	}

	return false
}

// SetMinLocationFailed gets a reference to the given int64 and assigns it to the MinLocationFailed field.
func (o *SyntheticsTestOptions) SetMinLocationFailed(v int64) {
	o.MinLocationFailed = &v
}

// GetMonitorName returns the MonitorName field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetMonitorName() string {
	if o == nil || o.MonitorName == nil {
		var ret string
		return ret
	}
	return *o.MonitorName
}

// GetMonitorNameOk returns a tuple with the MonitorName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetMonitorNameOk() (*string, bool) {
	if o == nil || o.MonitorName == nil {
		return nil, false
	}
	return o.MonitorName, true
}

// HasMonitorName returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasMonitorName() bool {
	if o != nil && o.MonitorName != nil {
		return true
	}

	return false
}

// SetMonitorName gets a reference to the given string and assigns it to the MonitorName field.
func (o *SyntheticsTestOptions) SetMonitorName(v string) {
	o.MonitorName = &v
}

// GetMonitorOptions returns the MonitorOptions field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetMonitorOptions() SyntheticsTestOptionsMonitorOptions {
	if o == nil || o.MonitorOptions == nil {
		var ret SyntheticsTestOptionsMonitorOptions
		return ret
	}
	return *o.MonitorOptions
}

// GetMonitorOptionsOk returns a tuple with the MonitorOptions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetMonitorOptionsOk() (*SyntheticsTestOptionsMonitorOptions, bool) {
	if o == nil || o.MonitorOptions == nil {
		return nil, false
	}
	return o.MonitorOptions, true
}

// HasMonitorOptions returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasMonitorOptions() bool {
	if o != nil && o.MonitorOptions != nil {
		return true
	}

	return false
}

// SetMonitorOptions gets a reference to the given SyntheticsTestOptionsMonitorOptions and assigns it to the MonitorOptions field.
func (o *SyntheticsTestOptions) SetMonitorOptions(v SyntheticsTestOptionsMonitorOptions) {
	o.MonitorOptions = &v
}

// GetMonitorPriority returns the MonitorPriority field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetMonitorPriority() int32 {
	if o == nil || o.MonitorPriority == nil {
		var ret int32
		return ret
	}
	return *o.MonitorPriority
}

// GetMonitorPriorityOk returns a tuple with the MonitorPriority field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetMonitorPriorityOk() (*int32, bool) {
	if o == nil || o.MonitorPriority == nil {
		return nil, false
	}
	return o.MonitorPriority, true
}

// HasMonitorPriority returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasMonitorPriority() bool {
	if o != nil && o.MonitorPriority != nil {
		return true
	}

	return false
}

// SetMonitorPriority gets a reference to the given int32 and assigns it to the MonitorPriority field.
func (o *SyntheticsTestOptions) SetMonitorPriority(v int32) {
	o.MonitorPriority = &v
}

// GetNoScreenshot returns the NoScreenshot field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetNoScreenshot() bool {
	if o == nil || o.NoScreenshot == nil {
		var ret bool
		return ret
	}
	return *o.NoScreenshot
}

// GetNoScreenshotOk returns a tuple with the NoScreenshot field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetNoScreenshotOk() (*bool, bool) {
	if o == nil || o.NoScreenshot == nil {
		return nil, false
	}
	return o.NoScreenshot, true
}

// HasNoScreenshot returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasNoScreenshot() bool {
	if o != nil && o.NoScreenshot != nil {
		return true
	}

	return false
}

// SetNoScreenshot gets a reference to the given bool and assigns it to the NoScreenshot field.
func (o *SyntheticsTestOptions) SetNoScreenshot(v bool) {
	o.NoScreenshot = &v
}

// GetRestrictedRoles returns the RestrictedRoles field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetRestrictedRoles() []string {
	if o == nil || o.RestrictedRoles == nil {
		var ret []string
		return ret
	}
	return o.RestrictedRoles
}

// GetRestrictedRolesOk returns a tuple with the RestrictedRoles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetRestrictedRolesOk() (*[]string, bool) {
	if o == nil || o.RestrictedRoles == nil {
		return nil, false
	}
	return &o.RestrictedRoles, true
}

// HasRestrictedRoles returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasRestrictedRoles() bool {
	if o != nil && o.RestrictedRoles != nil {
		return true
	}

	return false
}

// SetRestrictedRoles gets a reference to the given []string and assigns it to the RestrictedRoles field.
func (o *SyntheticsTestOptions) SetRestrictedRoles(v []string) {
	o.RestrictedRoles = v
}

// GetRetry returns the Retry field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetRetry() SyntheticsTestOptionsRetry {
	if o == nil || o.Retry == nil {
		var ret SyntheticsTestOptionsRetry
		return ret
	}
	return *o.Retry
}

// GetRetryOk returns a tuple with the Retry field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetRetryOk() (*SyntheticsTestOptionsRetry, bool) {
	if o == nil || o.Retry == nil {
		return nil, false
	}
	return o.Retry, true
}

// HasRetry returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasRetry() bool {
	if o != nil && o.Retry != nil {
		return true
	}

	return false
}

// SetRetry gets a reference to the given SyntheticsTestOptionsRetry and assigns it to the Retry field.
func (o *SyntheticsTestOptions) SetRetry(v SyntheticsTestOptionsRetry) {
	o.Retry = &v
}

// GetRumSettings returns the RumSettings field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetRumSettings() SyntheticsBrowserTestRumSettings {
	if o == nil || o.RumSettings == nil {
		var ret SyntheticsBrowserTestRumSettings
		return ret
	}
	return *o.RumSettings
}

// GetRumSettingsOk returns a tuple with the RumSettings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetRumSettingsOk() (*SyntheticsBrowserTestRumSettings, bool) {
	if o == nil || o.RumSettings == nil {
		return nil, false
	}
	return o.RumSettings, true
}

// HasRumSettings returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasRumSettings() bool {
	if o != nil && o.RumSettings != nil {
		return true
	}

	return false
}

// SetRumSettings gets a reference to the given SyntheticsBrowserTestRumSettings and assigns it to the RumSettings field.
func (o *SyntheticsTestOptions) SetRumSettings(v SyntheticsBrowserTestRumSettings) {
	o.RumSettings = &v
}

// GetTickEvery returns the TickEvery field value if set, zero value otherwise.
func (o *SyntheticsTestOptions) GetTickEvery() int64 {
	if o == nil || o.TickEvery == nil {
		var ret int64
		return ret
	}
	return *o.TickEvery
}

// GetTickEveryOk returns a tuple with the TickEvery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptions) GetTickEveryOk() (*int64, bool) {
	if o == nil || o.TickEvery == nil {
		return nil, false
	}
	return o.TickEvery, true
}

// HasTickEvery returns a boolean if a field has been set.
func (o *SyntheticsTestOptions) HasTickEvery() bool {
	if o != nil && o.TickEvery != nil {
		return true
	}

	return false
}

// SetTickEvery gets a reference to the given int64 and assigns it to the TickEvery field.
func (o *SyntheticsTestOptions) SetTickEvery(v int64) {
	o.TickEvery = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AcceptSelfSigned != nil {
		toSerialize["accept_self_signed"] = o.AcceptSelfSigned
	}
	if o.AllowInsecure != nil {
		toSerialize["allow_insecure"] = o.AllowInsecure
	}
	if o.CheckCertificateRevocation != nil {
		toSerialize["checkCertificateRevocation"] = o.CheckCertificateRevocation
	}
	if o.Ci != nil {
		toSerialize["ci"] = o.Ci
	}
	if o.DeviceIds != nil {
		toSerialize["device_ids"] = o.DeviceIds
	}
	if o.DisableCors != nil {
		toSerialize["disableCors"] = o.DisableCors
	}
	if o.FollowRedirects != nil {
		toSerialize["follow_redirects"] = o.FollowRedirects
	}
	if o.MinFailureDuration != nil {
		toSerialize["min_failure_duration"] = o.MinFailureDuration
	}
	if o.MinLocationFailed != nil {
		toSerialize["min_location_failed"] = o.MinLocationFailed
	}
	if o.MonitorName != nil {
		toSerialize["monitor_name"] = o.MonitorName
	}
	if o.MonitorOptions != nil {
		toSerialize["monitor_options"] = o.MonitorOptions
	}
	if o.MonitorPriority != nil {
		toSerialize["monitor_priority"] = o.MonitorPriority
	}
	if o.NoScreenshot != nil {
		toSerialize["noScreenshot"] = o.NoScreenshot
	}
	if o.RestrictedRoles != nil {
		toSerialize["restricted_roles"] = o.RestrictedRoles
	}
	if o.Retry != nil {
		toSerialize["retry"] = o.Retry
	}
	if o.RumSettings != nil {
		toSerialize["rumSettings"] = o.RumSettings
	}
	if o.TickEvery != nil {
		toSerialize["tick_every"] = o.TickEvery
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AcceptSelfSigned           *bool                                `json:"accept_self_signed,omitempty"`
		AllowInsecure              *bool                                `json:"allow_insecure,omitempty"`
		CheckCertificateRevocation *bool                                `json:"checkCertificateRevocation,omitempty"`
		Ci                         *SyntheticsTestCiOptions             `json:"ci,omitempty"`
		DeviceIds                  []SyntheticsDeviceID                 `json:"device_ids,omitempty"`
		DisableCors                *bool                                `json:"disableCors,omitempty"`
		FollowRedirects            *bool                                `json:"follow_redirects,omitempty"`
		MinFailureDuration         *int64                               `json:"min_failure_duration,omitempty"`
		MinLocationFailed          *int64                               `json:"min_location_failed,omitempty"`
		MonitorName                *string                              `json:"monitor_name,omitempty"`
		MonitorOptions             *SyntheticsTestOptionsMonitorOptions `json:"monitor_options,omitempty"`
		MonitorPriority            *int32                               `json:"monitor_priority,omitempty"`
		NoScreenshot               *bool                                `json:"noScreenshot,omitempty"`
		RestrictedRoles            []string                             `json:"restricted_roles,omitempty"`
		Retry                      *SyntheticsTestOptionsRetry          `json:"retry,omitempty"`
		RumSettings                *SyntheticsBrowserTestRumSettings    `json:"rumSettings,omitempty"`
		TickEvery                  *int64                               `json:"tick_every,omitempty"`
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
	o.AcceptSelfSigned = all.AcceptSelfSigned
	o.AllowInsecure = all.AllowInsecure
	o.CheckCertificateRevocation = all.CheckCertificateRevocation
	if all.Ci != nil && all.Ci.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Ci = all.Ci
	o.DeviceIds = all.DeviceIds
	o.DisableCors = all.DisableCors
	o.FollowRedirects = all.FollowRedirects
	o.MinFailureDuration = all.MinFailureDuration
	o.MinLocationFailed = all.MinLocationFailed
	o.MonitorName = all.MonitorName
	if all.MonitorOptions != nil && all.MonitorOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.MonitorOptions = all.MonitorOptions
	o.MonitorPriority = all.MonitorPriority
	o.NoScreenshot = all.NoScreenshot
	o.RestrictedRoles = all.RestrictedRoles
	if all.Retry != nil && all.Retry.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Retry = all.Retry
	if all.RumSettings != nil && all.RumSettings.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumSettings = all.RumSettings
	o.TickEvery = all.TickEvery
	return nil
}

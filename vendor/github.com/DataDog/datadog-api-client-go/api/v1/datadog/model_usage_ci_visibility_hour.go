// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageCIVisibilityHour CI visibility usage in a given hour.
type UsageCIVisibilityHour struct {
	// The number of spans for pipelines in the queried hour.
	CiPipelineIndexedSpans *int32 `json:"ci_pipeline_indexed_spans,omitempty"`
	// The number of spans for tests in the queried hour.
	CiTestIndexedSpans *int32 `json:"ci_test_indexed_spans,omitempty"`
	// Shows the total count of all active Git committers for Pipelines in the current month. A committer is active if they commit at least 3 times in a given month.
	CiVisibilityPipelineCommitters *int32 `json:"ci_visibility_pipeline_committers,omitempty"`
	// The total count of all active Git committers for tests in the current month. A committer is active if they commit at least 3 times in a given month.
	CiVisibilityTestCommitters *int32 `json:"ci_visibility_test_committers,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageCIVisibilityHour instantiates a new UsageCIVisibilityHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageCIVisibilityHour() *UsageCIVisibilityHour {
	this := UsageCIVisibilityHour{}
	return &this
}

// NewUsageCIVisibilityHourWithDefaults instantiates a new UsageCIVisibilityHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageCIVisibilityHourWithDefaults() *UsageCIVisibilityHour {
	this := UsageCIVisibilityHour{}
	return &this
}

// GetCiPipelineIndexedSpans returns the CiPipelineIndexedSpans field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetCiPipelineIndexedSpans() int32 {
	if o == nil || o.CiPipelineIndexedSpans == nil {
		var ret int32
		return ret
	}
	return *o.CiPipelineIndexedSpans
}

// GetCiPipelineIndexedSpansOk returns a tuple with the CiPipelineIndexedSpans field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetCiPipelineIndexedSpansOk() (*int32, bool) {
	if o == nil || o.CiPipelineIndexedSpans == nil {
		return nil, false
	}
	return o.CiPipelineIndexedSpans, true
}

// HasCiPipelineIndexedSpans returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasCiPipelineIndexedSpans() bool {
	if o != nil && o.CiPipelineIndexedSpans != nil {
		return true
	}

	return false
}

// SetCiPipelineIndexedSpans gets a reference to the given int32 and assigns it to the CiPipelineIndexedSpans field.
func (o *UsageCIVisibilityHour) SetCiPipelineIndexedSpans(v int32) {
	o.CiPipelineIndexedSpans = &v
}

// GetCiTestIndexedSpans returns the CiTestIndexedSpans field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetCiTestIndexedSpans() int32 {
	if o == nil || o.CiTestIndexedSpans == nil {
		var ret int32
		return ret
	}
	return *o.CiTestIndexedSpans
}

// GetCiTestIndexedSpansOk returns a tuple with the CiTestIndexedSpans field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetCiTestIndexedSpansOk() (*int32, bool) {
	if o == nil || o.CiTestIndexedSpans == nil {
		return nil, false
	}
	return o.CiTestIndexedSpans, true
}

// HasCiTestIndexedSpans returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasCiTestIndexedSpans() bool {
	if o != nil && o.CiTestIndexedSpans != nil {
		return true
	}

	return false
}

// SetCiTestIndexedSpans gets a reference to the given int32 and assigns it to the CiTestIndexedSpans field.
func (o *UsageCIVisibilityHour) SetCiTestIndexedSpans(v int32) {
	o.CiTestIndexedSpans = &v
}

// GetCiVisibilityPipelineCommitters returns the CiVisibilityPipelineCommitters field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetCiVisibilityPipelineCommitters() int32 {
	if o == nil || o.CiVisibilityPipelineCommitters == nil {
		var ret int32
		return ret
	}
	return *o.CiVisibilityPipelineCommitters
}

// GetCiVisibilityPipelineCommittersOk returns a tuple with the CiVisibilityPipelineCommitters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetCiVisibilityPipelineCommittersOk() (*int32, bool) {
	if o == nil || o.CiVisibilityPipelineCommitters == nil {
		return nil, false
	}
	return o.CiVisibilityPipelineCommitters, true
}

// HasCiVisibilityPipelineCommitters returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasCiVisibilityPipelineCommitters() bool {
	if o != nil && o.CiVisibilityPipelineCommitters != nil {
		return true
	}

	return false
}

// SetCiVisibilityPipelineCommitters gets a reference to the given int32 and assigns it to the CiVisibilityPipelineCommitters field.
func (o *UsageCIVisibilityHour) SetCiVisibilityPipelineCommitters(v int32) {
	o.CiVisibilityPipelineCommitters = &v
}

// GetCiVisibilityTestCommitters returns the CiVisibilityTestCommitters field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetCiVisibilityTestCommitters() int32 {
	if o == nil || o.CiVisibilityTestCommitters == nil {
		var ret int32
		return ret
	}
	return *o.CiVisibilityTestCommitters
}

// GetCiVisibilityTestCommittersOk returns a tuple with the CiVisibilityTestCommitters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetCiVisibilityTestCommittersOk() (*int32, bool) {
	if o == nil || o.CiVisibilityTestCommitters == nil {
		return nil, false
	}
	return o.CiVisibilityTestCommitters, true
}

// HasCiVisibilityTestCommitters returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasCiVisibilityTestCommitters() bool {
	if o != nil && o.CiVisibilityTestCommitters != nil {
		return true
	}

	return false
}

// SetCiVisibilityTestCommitters gets a reference to the given int32 and assigns it to the CiVisibilityTestCommitters field.
func (o *UsageCIVisibilityHour) SetCiVisibilityTestCommitters(v int32) {
	o.CiVisibilityTestCommitters = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageCIVisibilityHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageCIVisibilityHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCIVisibilityHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageCIVisibilityHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageCIVisibilityHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageCIVisibilityHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CiPipelineIndexedSpans != nil {
		toSerialize["ci_pipeline_indexed_spans"] = o.CiPipelineIndexedSpans
	}
	if o.CiTestIndexedSpans != nil {
		toSerialize["ci_test_indexed_spans"] = o.CiTestIndexedSpans
	}
	if o.CiVisibilityPipelineCommitters != nil {
		toSerialize["ci_visibility_pipeline_committers"] = o.CiVisibilityPipelineCommitters
	}
	if o.CiVisibilityTestCommitters != nil {
		toSerialize["ci_visibility_test_committers"] = o.CiVisibilityTestCommitters
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageCIVisibilityHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CiPipelineIndexedSpans         *int32  `json:"ci_pipeline_indexed_spans,omitempty"`
		CiTestIndexedSpans             *int32  `json:"ci_test_indexed_spans,omitempty"`
		CiVisibilityPipelineCommitters *int32  `json:"ci_visibility_pipeline_committers,omitempty"`
		CiVisibilityTestCommitters     *int32  `json:"ci_visibility_test_committers,omitempty"`
		OrgName                        *string `json:"org_name,omitempty"`
		PublicId                       *string `json:"public_id,omitempty"`
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
	o.CiPipelineIndexedSpans = all.CiPipelineIndexedSpans
	o.CiTestIndexedSpans = all.CiTestIndexedSpans
	o.CiVisibilityPipelineCommitters = all.CiVisibilityPipelineCommitters
	o.CiVisibilityTestCommitters = all.CiVisibilityTestCommitters
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}

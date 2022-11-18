// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsStepType Step type used in your Synthetic test.
type SyntheticsStepType string

// List of SyntheticsStepType.
const (
	SYNTHETICSSTEPTYPE_ASSERT_CURRENT_URL        SyntheticsStepType = "assertCurrentUrl"
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_ATTRIBUTE  SyntheticsStepType = "assertElementAttribute"
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_CONTENT    SyntheticsStepType = "assertElementContent"
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_PRESENT    SyntheticsStepType = "assertElementPresent"
	SYNTHETICSSTEPTYPE_ASSERT_EMAIL              SyntheticsStepType = "assertEmail"
	SYNTHETICSSTEPTYPE_ASSERT_FILE_DOWNLOAD      SyntheticsStepType = "assertFileDownload"
	SYNTHETICSSTEPTYPE_ASSERT_FROM_JAVASCRIPT    SyntheticsStepType = "assertFromJavascript"
	SYNTHETICSSTEPTYPE_ASSERT_PAGE_CONTAINS      SyntheticsStepType = "assertPageContains"
	SYNTHETICSSTEPTYPE_ASSERT_PAGE_LACKS         SyntheticsStepType = "assertPageLacks"
	SYNTHETICSSTEPTYPE_CLICK                     SyntheticsStepType = "click"
	SYNTHETICSSTEPTYPE_EXTRACT_FROM_JAVASCRIPT   SyntheticsStepType = "extractFromJavascript"
	SYNTHETICSSTEPTYPE_EXTRACT_VARIABLE          SyntheticsStepType = "extractVariable"
	SYNTHETICSSTEPTYPE_GO_TO_EMAIL_LINK          SyntheticsStepType = "goToEmailLink"
	SYNTHETICSSTEPTYPE_GO_TO_URL                 SyntheticsStepType = "goToUrl"
	SYNTHETICSSTEPTYPE_GO_TO_URL_AND_MEASURE_TTI SyntheticsStepType = "goToUrlAndMeasureTti"
	SYNTHETICSSTEPTYPE_HOVER                     SyntheticsStepType = "hover"
	SYNTHETICSSTEPTYPE_PLAY_SUB_TEST             SyntheticsStepType = "playSubTest"
	SYNTHETICSSTEPTYPE_PRESS_KEY                 SyntheticsStepType = "pressKey"
	SYNTHETICSSTEPTYPE_REFRESH                   SyntheticsStepType = "refresh"
	SYNTHETICSSTEPTYPE_RUN_API_TEST              SyntheticsStepType = "runApiTest"
	SYNTHETICSSTEPTYPE_SCROLL                    SyntheticsStepType = "scroll"
	SYNTHETICSSTEPTYPE_SELECT_OPTION             SyntheticsStepType = "selectOption"
	SYNTHETICSSTEPTYPE_TYPE_TEXT                 SyntheticsStepType = "typeText"
	SYNTHETICSSTEPTYPE_UPLOAD_FILES              SyntheticsStepType = "uploadFiles"
	SYNTHETICSSTEPTYPE_WAIT                      SyntheticsStepType = "wait"
)

var allowedSyntheticsStepTypeEnumValues = []SyntheticsStepType{
	SYNTHETICSSTEPTYPE_ASSERT_CURRENT_URL,
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_ATTRIBUTE,
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_CONTENT,
	SYNTHETICSSTEPTYPE_ASSERT_ELEMENT_PRESENT,
	SYNTHETICSSTEPTYPE_ASSERT_EMAIL,
	SYNTHETICSSTEPTYPE_ASSERT_FILE_DOWNLOAD,
	SYNTHETICSSTEPTYPE_ASSERT_FROM_JAVASCRIPT,
	SYNTHETICSSTEPTYPE_ASSERT_PAGE_CONTAINS,
	SYNTHETICSSTEPTYPE_ASSERT_PAGE_LACKS,
	SYNTHETICSSTEPTYPE_CLICK,
	SYNTHETICSSTEPTYPE_EXTRACT_FROM_JAVASCRIPT,
	SYNTHETICSSTEPTYPE_EXTRACT_VARIABLE,
	SYNTHETICSSTEPTYPE_GO_TO_EMAIL_LINK,
	SYNTHETICSSTEPTYPE_GO_TO_URL,
	SYNTHETICSSTEPTYPE_GO_TO_URL_AND_MEASURE_TTI,
	SYNTHETICSSTEPTYPE_HOVER,
	SYNTHETICSSTEPTYPE_PLAY_SUB_TEST,
	SYNTHETICSSTEPTYPE_PRESS_KEY,
	SYNTHETICSSTEPTYPE_REFRESH,
	SYNTHETICSSTEPTYPE_RUN_API_TEST,
	SYNTHETICSSTEPTYPE_SCROLL,
	SYNTHETICSSTEPTYPE_SELECT_OPTION,
	SYNTHETICSSTEPTYPE_TYPE_TEXT,
	SYNTHETICSSTEPTYPE_UPLOAD_FILES,
	SYNTHETICSSTEPTYPE_WAIT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsStepType) GetAllowedValues() []SyntheticsStepType {
	return allowedSyntheticsStepTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsStepType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsStepType(value)
	return nil
}

// NewSyntheticsStepTypeFromValue returns a pointer to a valid SyntheticsStepType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsStepTypeFromValue(v string) (*SyntheticsStepType, error) {
	ev := SyntheticsStepType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsStepType: valid values are %v", v, allowedSyntheticsStepTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsStepType) IsValid() bool {
	for _, existing := range allowedSyntheticsStepTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsStepType value.
func (v SyntheticsStepType) Ptr() *SyntheticsStepType {
	return &v
}

// NullableSyntheticsStepType handles when a null is used for SyntheticsStepType.
type NullableSyntheticsStepType struct {
	value *SyntheticsStepType
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsStepType) Get() *SyntheticsStepType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsStepType) Set(val *SyntheticsStepType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsStepType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsStepType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsStepType initializes the struct as if Set has been called.
func NewNullableSyntheticsStepType(val *SyntheticsStepType) *NullableSyntheticsStepType {
	return &NullableSyntheticsStepType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsStepType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsStepType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

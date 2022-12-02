// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserTestFailureCode Error code that can be returned by a Synthetic test.
type SyntheticsBrowserTestFailureCode string

// List of SyntheticsBrowserTestFailureCode.
const (
	SYNTHETICSBROWSERTESTFAILURECODE_API_REQUEST_FAILURE          SyntheticsBrowserTestFailureCode = "API_REQUEST_FAILURE"
	SYNTHETICSBROWSERTESTFAILURECODE_ASSERTION_FAILURE            SyntheticsBrowserTestFailureCode = "ASSERTION_FAILURE"
	SYNTHETICSBROWSERTESTFAILURECODE_DOWNLOAD_FILE_TOO_LARGE      SyntheticsBrowserTestFailureCode = "DOWNLOAD_FILE_TOO_LARGE"
	SYNTHETICSBROWSERTESTFAILURECODE_ELEMENT_NOT_INTERACTABLE     SyntheticsBrowserTestFailureCode = "ELEMENT_NOT_INTERACTABLE"
	SYNTHETICSBROWSERTESTFAILURECODE_EMAIL_VARIABLE_NOT_DEFINED   SyntheticsBrowserTestFailureCode = "EMAIL_VARIABLE_NOT_DEFINED"
	SYNTHETICSBROWSERTESTFAILURECODE_EVALUATE_JAVASCRIPT          SyntheticsBrowserTestFailureCode = "EVALUATE_JAVASCRIPT"
	SYNTHETICSBROWSERTESTFAILURECODE_EVALUATE_JAVASCRIPT_CONTEXT  SyntheticsBrowserTestFailureCode = "EVALUATE_JAVASCRIPT_CONTEXT"
	SYNTHETICSBROWSERTESTFAILURECODE_EXTRACT_VARIABLE             SyntheticsBrowserTestFailureCode = "EXTRACT_VARIABLE"
	SYNTHETICSBROWSERTESTFAILURECODE_FORBIDDEN_URL                SyntheticsBrowserTestFailureCode = "FORBIDDEN_URL"
	SYNTHETICSBROWSERTESTFAILURECODE_FRAME_DETACHED               SyntheticsBrowserTestFailureCode = "FRAME_DETACHED"
	SYNTHETICSBROWSERTESTFAILURECODE_INCONSISTENCIES              SyntheticsBrowserTestFailureCode = "INCONSISTENCIES"
	SYNTHETICSBROWSERTESTFAILURECODE_INTERNAL_ERROR               SyntheticsBrowserTestFailureCode = "INTERNAL_ERROR"
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_TYPE_TEXT_DELAY      SyntheticsBrowserTestFailureCode = "INVALID_TYPE_TEXT_DELAY"
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_URL                  SyntheticsBrowserTestFailureCode = "INVALID_URL"
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_VARIABLE_PATTERN     SyntheticsBrowserTestFailureCode = "INVALID_VARIABLE_PATTERN"
	SYNTHETICSBROWSERTESTFAILURECODE_INVISIBLE_ELEMENT            SyntheticsBrowserTestFailureCode = "INVISIBLE_ELEMENT"
	SYNTHETICSBROWSERTESTFAILURECODE_LOCATE_ELEMENT               SyntheticsBrowserTestFailureCode = "LOCATE_ELEMENT"
	SYNTHETICSBROWSERTESTFAILURECODE_NAVIGATE_TO_LINK             SyntheticsBrowserTestFailureCode = "NAVIGATE_TO_LINK"
	SYNTHETICSBROWSERTESTFAILURECODE_OPEN_URL                     SyntheticsBrowserTestFailureCode = "OPEN_URL"
	SYNTHETICSBROWSERTESTFAILURECODE_PRESS_KEY                    SyntheticsBrowserTestFailureCode = "PRESS_KEY"
	SYNTHETICSBROWSERTESTFAILURECODE_SERVER_CERTIFICATE           SyntheticsBrowserTestFailureCode = "SERVER_CERTIFICATE"
	SYNTHETICSBROWSERTESTFAILURECODE_SELECT_OPTION                SyntheticsBrowserTestFailureCode = "SELECT_OPTION"
	SYNTHETICSBROWSERTESTFAILURECODE_STEP_TIMEOUT                 SyntheticsBrowserTestFailureCode = "STEP_TIMEOUT"
	SYNTHETICSBROWSERTESTFAILURECODE_SUB_TEST_NOT_PASSED          SyntheticsBrowserTestFailureCode = "SUB_TEST_NOT_PASSED"
	SYNTHETICSBROWSERTESTFAILURECODE_TEST_TIMEOUT                 SyntheticsBrowserTestFailureCode = "TEST_TIMEOUT"
	SYNTHETICSBROWSERTESTFAILURECODE_TOO_MANY_HTTP_REQUESTS       SyntheticsBrowserTestFailureCode = "TOO_MANY_HTTP_REQUESTS"
	SYNTHETICSBROWSERTESTFAILURECODE_UNAVAILABLE_BROWSER          SyntheticsBrowserTestFailureCode = "UNAVAILABLE_BROWSER"
	SYNTHETICSBROWSERTESTFAILURECODE_UNKNOWN                      SyntheticsBrowserTestFailureCode = "UNKNOWN"
	SYNTHETICSBROWSERTESTFAILURECODE_UNSUPPORTED_AUTH_SCHEMA      SyntheticsBrowserTestFailureCode = "UNSUPPORTED_AUTH_SCHEMA"
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_ELEMENT_TYPE    SyntheticsBrowserTestFailureCode = "UPLOAD_FILES_ELEMENT_TYPE"
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_DIALOG          SyntheticsBrowserTestFailureCode = "UPLOAD_FILES_DIALOG"
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_DYNAMIC_ELEMENT SyntheticsBrowserTestFailureCode = "UPLOAD_FILES_DYNAMIC_ELEMENT"
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_NAME            SyntheticsBrowserTestFailureCode = "UPLOAD_FILES_NAME"
)

var allowedSyntheticsBrowserTestFailureCodeEnumValues = []SyntheticsBrowserTestFailureCode{
	SYNTHETICSBROWSERTESTFAILURECODE_API_REQUEST_FAILURE,
	SYNTHETICSBROWSERTESTFAILURECODE_ASSERTION_FAILURE,
	SYNTHETICSBROWSERTESTFAILURECODE_DOWNLOAD_FILE_TOO_LARGE,
	SYNTHETICSBROWSERTESTFAILURECODE_ELEMENT_NOT_INTERACTABLE,
	SYNTHETICSBROWSERTESTFAILURECODE_EMAIL_VARIABLE_NOT_DEFINED,
	SYNTHETICSBROWSERTESTFAILURECODE_EVALUATE_JAVASCRIPT,
	SYNTHETICSBROWSERTESTFAILURECODE_EVALUATE_JAVASCRIPT_CONTEXT,
	SYNTHETICSBROWSERTESTFAILURECODE_EXTRACT_VARIABLE,
	SYNTHETICSBROWSERTESTFAILURECODE_FORBIDDEN_URL,
	SYNTHETICSBROWSERTESTFAILURECODE_FRAME_DETACHED,
	SYNTHETICSBROWSERTESTFAILURECODE_INCONSISTENCIES,
	SYNTHETICSBROWSERTESTFAILURECODE_INTERNAL_ERROR,
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_TYPE_TEXT_DELAY,
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_URL,
	SYNTHETICSBROWSERTESTFAILURECODE_INVALID_VARIABLE_PATTERN,
	SYNTHETICSBROWSERTESTFAILURECODE_INVISIBLE_ELEMENT,
	SYNTHETICSBROWSERTESTFAILURECODE_LOCATE_ELEMENT,
	SYNTHETICSBROWSERTESTFAILURECODE_NAVIGATE_TO_LINK,
	SYNTHETICSBROWSERTESTFAILURECODE_OPEN_URL,
	SYNTHETICSBROWSERTESTFAILURECODE_PRESS_KEY,
	SYNTHETICSBROWSERTESTFAILURECODE_SERVER_CERTIFICATE,
	SYNTHETICSBROWSERTESTFAILURECODE_SELECT_OPTION,
	SYNTHETICSBROWSERTESTFAILURECODE_STEP_TIMEOUT,
	SYNTHETICSBROWSERTESTFAILURECODE_SUB_TEST_NOT_PASSED,
	SYNTHETICSBROWSERTESTFAILURECODE_TEST_TIMEOUT,
	SYNTHETICSBROWSERTESTFAILURECODE_TOO_MANY_HTTP_REQUESTS,
	SYNTHETICSBROWSERTESTFAILURECODE_UNAVAILABLE_BROWSER,
	SYNTHETICSBROWSERTESTFAILURECODE_UNKNOWN,
	SYNTHETICSBROWSERTESTFAILURECODE_UNSUPPORTED_AUTH_SCHEMA,
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_ELEMENT_TYPE,
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_DIALOG,
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_DYNAMIC_ELEMENT,
	SYNTHETICSBROWSERTESTFAILURECODE_UPLOAD_FILES_NAME,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsBrowserTestFailureCode) GetAllowedValues() []SyntheticsBrowserTestFailureCode {
	return allowedSyntheticsBrowserTestFailureCodeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsBrowserTestFailureCode) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsBrowserTestFailureCode(value)
	return nil
}

// NewSyntheticsBrowserTestFailureCodeFromValue returns a pointer to a valid SyntheticsBrowserTestFailureCode
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsBrowserTestFailureCodeFromValue(v string) (*SyntheticsBrowserTestFailureCode, error) {
	ev := SyntheticsBrowserTestFailureCode(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsBrowserTestFailureCode: valid values are %v", v, allowedSyntheticsBrowserTestFailureCodeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsBrowserTestFailureCode) IsValid() bool {
	for _, existing := range allowedSyntheticsBrowserTestFailureCodeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsBrowserTestFailureCode value.
func (v SyntheticsBrowserTestFailureCode) Ptr() *SyntheticsBrowserTestFailureCode {
	return &v
}

// NullableSyntheticsBrowserTestFailureCode handles when a null is used for SyntheticsBrowserTestFailureCode.
type NullableSyntheticsBrowserTestFailureCode struct {
	value *SyntheticsBrowserTestFailureCode
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBrowserTestFailureCode) Get() *SyntheticsBrowserTestFailureCode {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBrowserTestFailureCode) Set(val *SyntheticsBrowserTestFailureCode) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBrowserTestFailureCode) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsBrowserTestFailureCode) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBrowserTestFailureCode initializes the struct as if Set has been called.
func NewNullableSyntheticsBrowserTestFailureCode(val *SyntheticsBrowserTestFailureCode) *NullableSyntheticsBrowserTestFailureCode {
	return &NullableSyntheticsBrowserTestFailureCode{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBrowserTestFailureCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBrowserTestFailureCode) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

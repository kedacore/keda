// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
// --------------------------------------------------------------------------------------------
// Generated file, DO NOT EDIT
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// --------------------------------------------------------------------------------------------

package forminput

import (
	"math/big"
)

// Enumerates data types that are supported as subscription input values.
type InputDataType string

type inputDataTypeValuesType struct {
	None    InputDataType
	String  InputDataType
	Number  InputDataType
	Boolean InputDataType
	Guid    InputDataType
	Uri     InputDataType
}

var InputDataTypeValues = inputDataTypeValuesType{
	// No data type is specified.
	None: "none",
	// Represents a textual value.
	String: "string",
	// Represents a numeric value.
	Number: "number",
	// Represents a value of true or false.
	Boolean: "boolean",
	// Represents a Guid.
	Guid: "guid",
	// Represents a URI.
	Uri: "uri",
}

// Describes an input for subscriptions.
type InputDescriptor struct {
	// The ids of all inputs that the value of this input is dependent on.
	DependencyInputIds *[]string `json:"dependencyInputIds,omitempty"`
	// Description of what this input is used for
	Description *string `json:"description,omitempty"`
	// The group localized name to which this input belongs and can be shown as a header for the container that will include all the inputs in the group.
	GroupName *string `json:"groupName,omitempty"`
	// If true, the value information for this input is dynamic and should be fetched when the value of dependency inputs change.
	HasDynamicValueInformation *bool `json:"hasDynamicValueInformation,omitempty"`
	// Identifier for the subscription input
	Id *string `json:"id,omitempty"`
	// Mode in which the value of this input should be entered
	InputMode *InputMode `json:"inputMode,omitempty"`
	// Gets whether this input is confidential, such as for a password or application key
	IsConfidential *bool `json:"isConfidential,omitempty"`
	// Localized name which can be shown as a label for the subscription input
	Name *string `json:"name,omitempty"`
	// Custom properties for the input which can be used by the service provider
	Properties *map[string]interface{} `json:"properties,omitempty"`
	// Underlying data type for the input value. When this value is specified, InputMode, Validation and Values are optional.
	Type *string `json:"type,omitempty"`
	// Gets whether this input is included in the default generated action description.
	UseInDefaultDescription *bool `json:"useInDefaultDescription,omitempty"`
	// Information to use to validate this input's value
	Validation *InputValidation `json:"validation,omitempty"`
	// A hint for input value. It can be used in the UI as the input placeholder.
	ValueHint *string `json:"valueHint,omitempty"`
	// Information about possible values for this input
	Values *InputValues `json:"values,omitempty"`
}

// Defines a filter for subscription inputs. The filter matches a set of inputs if any (one or more) of the groups evaluates to true.
type InputFilter struct {
	// Groups of input filter expressions. This filter matches a set of inputs if any (one or more) of the groups evaluates to true.
	Conditions *[]InputFilterCondition `json:"conditions,omitempty"`
}

// An expression which can be applied to filter a list of subscription inputs
type InputFilterCondition struct {
	// Whether or not to do a case sensitive match
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
	// The Id of the input to filter on
	InputId *string `json:"inputId,omitempty"`
	// The "expected" input value to compare with the actual input value
	InputValue *string `json:"inputValue,omitempty"`
	// The operator applied between the expected and actual input value
	Operator *InputFilterOperator `json:"operator,omitempty"`
}

type InputFilterOperator string

type inputFilterOperatorValuesType struct {
	Equals    InputFilterOperator
	NotEquals InputFilterOperator
}

var InputFilterOperatorValues = inputFilterOperatorValuesType{
	Equals:    "equals",
	NotEquals: "notEquals",
}

// Mode in which a subscription input should be entered (in a UI)
type InputMode string

type inputModeValuesType struct {
	None         InputMode
	TextBox      InputMode
	PasswordBox  InputMode
	Combo        InputMode
	RadioButtons InputMode
	CheckBox     InputMode
	TextArea     InputMode
}

var InputModeValues = inputModeValuesType{
	// This input should not be shown in the UI
	None: "none",
	// An input text box should be shown
	TextBox: "textBox",
	// An password input box should be shown
	PasswordBox: "passwordBox",
	// A select/combo control should be shown
	Combo: "combo",
	// Radio buttons should be shown
	RadioButtons: "radioButtons",
	// Checkbox should be shown(for true/false values)
	CheckBox: "checkBox",
	// A multi-line text area should be shown
	TextArea: "textArea",
}

// Describes what values are valid for a subscription input
type InputValidation struct {
	// Gets or sets the data data type to validate.
	DataType *InputDataType `json:"dataType,omitempty"`
	// Gets or sets if this is a required field.
	IsRequired *bool `json:"isRequired,omitempty"`
	// Gets or sets the maximum length of this descriptor.
	MaxLength *int `json:"maxLength,omitempty"`
	// Gets or sets the minimum value for this descriptor.
	MaxValue *big.Float `json:"maxValue,omitempty"`
	// Gets or sets the minimum length of this descriptor.
	MinLength *int `json:"minLength,omitempty"`
	// Gets or sets the minimum value for this descriptor.
	MinValue *big.Float `json:"minValue,omitempty"`
	// Gets or sets the pattern to validate.
	Pattern *string `json:"pattern,omitempty"`
	// Gets or sets the error on pattern mismatch.
	PatternMismatchErrorMessage *string `json:"patternMismatchErrorMessage,omitempty"`
}

// Information about a single value for an input
type InputValue struct {
	// Any other data about this input
	Data *map[string]interface{} `json:"data,omitempty"`
	// The text to show for the display of this value
	DisplayValue *string `json:"displayValue,omitempty"`
	// The value to store for this input
	Value *string `json:"value,omitempty"`
}

// Information about the possible/allowed values for a given subscription input
type InputValues struct {
	// The default value to use for this input
	DefaultValue *string `json:"defaultValue,omitempty"`
	// Errors encountered while computing dynamic values.
	Error *InputValuesError `json:"error,omitempty"`
	// The id of the input
	InputId *string `json:"inputId,omitempty"`
	// Should this input be disabled
	IsDisabled *bool `json:"isDisabled,omitempty"`
	// Should the value be restricted to one of the values in the PossibleValues (True) or are the values in PossibleValues just a suggestion (False)
	IsLimitedToPossibleValues *bool `json:"isLimitedToPossibleValues,omitempty"`
	// Should this input be made read-only
	IsReadOnly *bool `json:"isReadOnly,omitempty"`
	// Possible values that this input can take
	PossibleValues *[]InputValue `json:"possibleValues,omitempty"`
}

// Error information related to a subscription input value.
type InputValuesError struct {
	// The error message.
	Message *string `json:"message,omitempty"`
}

type InputValuesQuery struct {
	CurrentValues *map[string]string `json:"currentValues,omitempty"`
	// The input values to return on input, and the result from the consumer on output.
	InputValues *[]InputValues `json:"inputValues,omitempty"`
	// Subscription containing information about the publisher/consumer and the current input values
	Resource interface{} `json:"resource,omitempty"`
}

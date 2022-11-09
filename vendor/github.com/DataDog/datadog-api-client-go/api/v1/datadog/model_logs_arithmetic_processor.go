// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// LogsArithmeticProcessor Use the Arithmetic Processor to add a new attribute (without spaces or special characters
// in the new attribute name) to a log with the result of the provided formula.
// This enables you to remap different time attributes with different units into a single attribute,
// or to compute operations on attributes within the same log.
//
// The formula can use parentheses and the basic arithmetic operators `-`, `+`, `*`, `/`.
//
// By default, the calculation is skipped if an attribute is missing.
// Select “Replace missing attribute by 0” to automatically populate
// missing attribute values with 0 to ensure that the calculation is done.
// An attribute is missing if it is not found in the log attributes,
// or if it cannot be converted to a number.
//
// *Notes*:
//
// - The operator `-` needs to be space split in the formula as it can also be contained in attribute names.
// - If the target attribute already exists, it is overwritten by the result of the formula.
// - Results are rounded up to the 9th decimal. For example, if the result of the formula is `0.1234567891`,
//   the actual value stored for the attribute is `0.123456789`.
// - If you need to scale a unit of measure,
//   see [Scale Filter](https://docs.datadoghq.com/logs/log_configuration/parsing/?tab=filter#matcher-and-filter).
type LogsArithmeticProcessor struct {
	// Arithmetic operation between one or more log attributes.
	Expression string `json:"expression"`
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// If `true`, it replaces all missing attributes of expression by `0`, `false`
	// skip the operation if an attribute is missing.
	IsReplaceMissing *bool `json:"is_replace_missing,omitempty"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Name of the attribute that contains the result of the arithmetic operation.
	Target string `json:"target"`
	// Type of logs arithmetic processor.
	Type LogsArithmeticProcessorType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsArithmeticProcessor instantiates a new LogsArithmeticProcessor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArithmeticProcessor(expression string, target string, typeVar LogsArithmeticProcessorType) *LogsArithmeticProcessor {
	this := LogsArithmeticProcessor{}
	this.Expression = expression
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var isReplaceMissing bool = false
	this.IsReplaceMissing = &isReplaceMissing
	this.Target = target
	this.Type = typeVar
	return &this
}

// NewLogsArithmeticProcessorWithDefaults instantiates a new LogsArithmeticProcessor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArithmeticProcessorWithDefaults() *LogsArithmeticProcessor {
	this := LogsArithmeticProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var isReplaceMissing bool = false
	this.IsReplaceMissing = &isReplaceMissing
	var typeVar LogsArithmeticProcessorType = LOGSARITHMETICPROCESSORTYPE_ARITHMETIC_PROCESSOR
	this.Type = typeVar
	return &this
}

// GetExpression returns the Expression field value.
func (o *LogsArithmeticProcessor) GetExpression() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetExpressionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Expression, true
}

// SetExpression sets field value.
func (o *LogsArithmeticProcessor) SetExpression(v string) {
	o.Expression = v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *LogsArithmeticProcessor) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *LogsArithmeticProcessor) HasIsEnabled() bool {
	if o != nil && o.IsEnabled != nil {
		return true
	}

	return false
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *LogsArithmeticProcessor) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetIsReplaceMissing returns the IsReplaceMissing field value if set, zero value otherwise.
func (o *LogsArithmeticProcessor) GetIsReplaceMissing() bool {
	if o == nil || o.IsReplaceMissing == nil {
		var ret bool
		return ret
	}
	return *o.IsReplaceMissing
}

// GetIsReplaceMissingOk returns a tuple with the IsReplaceMissing field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetIsReplaceMissingOk() (*bool, bool) {
	if o == nil || o.IsReplaceMissing == nil {
		return nil, false
	}
	return o.IsReplaceMissing, true
}

// HasIsReplaceMissing returns a boolean if a field has been set.
func (o *LogsArithmeticProcessor) HasIsReplaceMissing() bool {
	if o != nil && o.IsReplaceMissing != nil {
		return true
	}

	return false
}

// SetIsReplaceMissing gets a reference to the given bool and assigns it to the IsReplaceMissing field.
func (o *LogsArithmeticProcessor) SetIsReplaceMissing(v bool) {
	o.IsReplaceMissing = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *LogsArithmeticProcessor) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *LogsArithmeticProcessor) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *LogsArithmeticProcessor) SetName(v string) {
	o.Name = &v
}

// GetTarget returns the Target field value.
func (o *LogsArithmeticProcessor) GetTarget() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetTargetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *LogsArithmeticProcessor) SetTarget(v string) {
	o.Target = v
}

// GetType returns the Type field value.
func (o *LogsArithmeticProcessor) GetType() LogsArithmeticProcessorType {
	if o == nil {
		var ret LogsArithmeticProcessorType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsArithmeticProcessor) GetTypeOk() (*LogsArithmeticProcessorType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsArithmeticProcessor) SetType(v LogsArithmeticProcessorType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArithmeticProcessor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["expression"] = o.Expression
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.IsReplaceMissing != nil {
		toSerialize["is_replace_missing"] = o.IsReplaceMissing
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["target"] = o.Target
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArithmeticProcessor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Expression *string                      `json:"expression"`
		Target     *string                      `json:"target"`
		Type       *LogsArithmeticProcessorType `json:"type"`
	}{}
	all := struct {
		Expression       string                      `json:"expression"`
		IsEnabled        *bool                       `json:"is_enabled,omitempty"`
		IsReplaceMissing *bool                       `json:"is_replace_missing,omitempty"`
		Name             *string                     `json:"name,omitempty"`
		Target           string                      `json:"target"`
		Type             LogsArithmeticProcessorType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Expression == nil {
		return fmt.Errorf("Required field expression missing")
	}
	if required.Target == nil {
		return fmt.Errorf("Required field target missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Expression = all.Expression
	o.IsEnabled = all.IsEnabled
	o.IsReplaceMissing = all.IsReplaceMissing
	o.Name = all.Name
	o.Target = all.Target
	o.Type = all.Type
	return nil
}

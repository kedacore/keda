/*
Copyright 2024 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalersconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kedacore/keda/v2/pkg/eventreason"
)

// CustomValidator is an interface that can be implemented to validate the configuration of the typed config
type CustomValidator interface {
	Validate() error
}

// ParsingOrder is a type that represents the order in which the parameters are parsed
type ParsingOrder string

// Constants that represent the order in which the parameters are parsed
const (
	TriggerMetadata ParsingOrder = "triggerMetadata"
	ResolvedEnv     ParsingOrder = "resolvedEnv"
	AuthParams      ParsingOrder = "authParams"
)

// allowedParsingOrderMap is a map with set of valid parsing orders
var allowedParsingOrderMap = map[ParsingOrder]bool{
	TriggerMetadata: true,
	ResolvedEnv:     true,
	AuthParams:      true,
}

// separators for field tag structure
// e.g. name=stringVal,order=triggerMetadata;resolvedEnv;authParams,optional
const (
	tagSeparator      = ","
	tagKeySeparator   = "="
	tagValueSeparator = ";"
)

// separators for map and slice elements
const (
	elemKeyValSeparator = "="
)

// field tag parameters
const (
	optionalTag           = "optional"
	deprecatedTag         = "deprecated"
	deprecatedAnnounceTag = "deprecatedAnnounce"
	defaultTag            = "default"
	orderTag              = "order"
	nameTag               = "name"
	enumTag               = "enum"
	exclusiveSetTag       = "exclusiveSet"
	rangeTag              = "range"
	separatorTag          = "separator"
)

// Params is a struct that represents the parameter list that can be used in the keda tag
type Params struct {
	// FieldName is the name of the field in the struct
	FieldName string

	// Names is the 'name' tag parameter defining the key in triggerMetadata, resolvedEnv or authParams
	Names []string

	// Optional is the 'optional' tag parameter defining if the parameter is optional
	Optional bool

	// Order is the 'order' tag parameter defining the parsing order in which the parameter is looked up
	// in the triggerMetadata, resolvedEnv or authParams maps
	Order []ParsingOrder

	// Default is the 'default' tag parameter defining the default value of the parameter if it's not found
	// in any of the maps from ParsingOrder
	Default string

	// Deprecated is the 'deprecated' tag parameter, if the map contain this parameter, it is considered
	// as an error and the DeprecatedMessage should be returned to the user
	Deprecated string

	// DeprecatedAnnounce is the 'deprecatedAnnounce' tag parameter, if set this will trigger
	// an info event with the deprecation message
	DeprecatedAnnounce string

	// Enum is the 'enum' tag parameter defining the list of possible values for the parameter
	Enum []string

	// ExclusiveSet is the 'exclusiveSet' tag parameter defining the list of values that are mutually exclusive
	ExclusiveSet []string

	// RangeSeparator is the 'range' tag parameter defining the separator for range values
	RangeSeparator string

	// Separator is the tag parameter to define which separator will be used
	Separator string
}

// Name returns the name of the parameter (or comma separated list of names if it has multiple)
func (p Params) Name() string {
	return strings.Join(p.Names, ",")
}

// IsNested is a function that returns true if the parameter is nested
func (p Params) IsNested() bool {
	return len(p.Names) == 0
}

// IsDeprecated is a function that returns true if the parameter is deprecated
func (p Params) IsDeprecated() bool {
	return p.Deprecated != ""
}

// TypedConfig is a function that is used to unmarshal the TriggerMetadata, ResolvedEnv and AuthParams
// populating the provided typedConfig where structure fields along with complementary field tags define
// declaratively the parsing rules
func (sc *ScalerConfig) TypedConfig(typedConfig any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// this shouldn't happen, but calling certain reflection functions may result in panic
			// if it does, it's better to return a error with stacktrace and reject parsing config
			// rather than crashing KEDA
			err = fmt.Errorf("failed to parse typed config %T resulted in panic\n%v", r, string(debug.Stack()))
		}
	}()
	err = sc.parseTypedConfig(typedConfig, false)
	return
}

// parseTypedConfig is a function that is used to unmarshal the TriggerMetadata, ResolvedEnv and AuthParams
// this can be called recursively to parse nested structures
func (sc *ScalerConfig) parseTypedConfig(typedConfig any, parentOptional bool) error {
	t := reflect.TypeOf(typedConfig)
	if t.Kind() != reflect.Pointer {
		return fmt.Errorf("typedConfig must be a pointer")
	}
	t = t.Elem()
	v := reflect.ValueOf(typedConfig).Elem()

	errs := []error{}
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)
		tag, exists := fieldType.Tag.Lookup("keda")
		if !exists {
			continue
		}
		tagParams, err := paramsFromTag(tag, fieldType)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tagParams.Optional = tagParams.Optional || parentOptional
		if err := sc.setValue(fieldValue, tagParams); err != nil {
			errs = append(errs, err)
		}
	}
	if validator, ok := typedConfig.(CustomValidator); ok {
		if err := validator.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// setValue is a function that sets the value of the field based on the provided params
func (sc *ScalerConfig) setValue(field reflect.Value, params Params) error {
	valFromConfig, exists := sc.configParamValue(params)
	if exists && params.IsDeprecated() {
		return fmt.Errorf("scaler %s info: %s", sc.TriggerType, params.Deprecated)
	}
	if exists && params.DeprecatedAnnounce != "" {
		if sc.Recorder != nil {
			message := fmt.Sprintf("scaler %s info: %s", sc.TriggerType, params.DeprecatedAnnounce)
			fmt.Print(message)
			sc.Recorder.Event(sc.ScaledObject, corev1.EventTypeNormal, eventreason.KEDAScalersInfo, message)
		}
	}
	if !exists && params.Default != "" {
		exists = true
		valFromConfig = params.Default
	}
	if !exists && (params.Optional || params.IsDeprecated()) {
		return nil
	}
	if !exists && !(params.Optional || params.IsDeprecated()) {
		if len(params.Order) == 0 {
			apo := slices.Sorted(maps.Keys(allowedParsingOrderMap))
			return fmt.Errorf("missing required parameter %q, no 'order' tag, provide any from %v", params.Name(), apo)
		}
		return fmt.Errorf("missing required parameter %q in %v", params.Name(), params.Order)
	}
	if params.Enum != nil {
		enumMap := make(map[string]bool)
		for _, e := range params.Enum {
			enumMap[e] = true
		}
		missingMap := make(map[string]bool)
		split := splitWithSeparator(valFromConfig, params.Separator)
		for _, s := range split {
			s := strings.TrimSpace(s)
			if !enumMap[s] {
				missingMap[s] = true
			}
		}
		if len(missingMap) > 0 {
			return fmt.Errorf("parameter %q value %q must be one of %v", params.Name(), valFromConfig, params.Enum)
		}
	}
	if params.ExclusiveSet != nil {
		exclusiveMap := make(map[string]bool)
		for _, e := range params.ExclusiveSet {
			exclusiveMap[e] = true
		}
		split := splitWithSeparator(valFromConfig, params.Separator)
		exclusiveCount := 0
		for _, s := range split {
			s := strings.TrimSpace(s)
			if exclusiveMap[s] {
				exclusiveCount++
			}
		}
		if exclusiveCount > 1 {
			return fmt.Errorf("parameter %q value %q must contain only one of %v", params.Name(), valFromConfig, params.ExclusiveSet)
		}
	}
	if params.IsNested() {
		for field.Kind() == reflect.Ptr {
			field.Set(reflect.New(field.Type().Elem()))
			field = field.Elem()
		}
		if field.Kind() != reflect.Struct {
			return fmt.Errorf("nested parameter %q must be a struct, has kind %q", params.FieldName, field.Kind())
		}
		return sc.parseTypedConfig(field.Addr().Interface(), params.Optional)
	}
	if err := setConfigValueHelper(params, valFromConfig, field); err != nil {
		return fmt.Errorf("unable to set param %q value %q: %w", params.Name(), valFromConfig, err)
	}
	return nil
}

// setConfigValueURLParams is a function that sets the value of the url.Values field
func setConfigValueURLParams(params Params, valFromConfig string, field reflect.Value) error {
	field.Set(reflect.MakeMap(reflect.MapOf(field.Type().Key(), field.Type().Elem())))
	vals, err := url.ParseQuery(valFromConfig)
	if err != nil {
		return fmt.Errorf("expected url.Values, unable to parse query %q: %w", valFromConfig, err)
	}
	for k, vs := range vals {
		ifcMapKeyElem := reflect.New(field.Type().Key()).Elem()
		ifcMapValueElem := reflect.New(field.Type().Elem()).Elem()
		if err := setConfigValueHelper(params, k, ifcMapKeyElem); err != nil {
			return fmt.Errorf("map key %q: %w", k, err)
		}
		for _, v := range vs {
			ifcMapValueElem.Set(reflect.Append(ifcMapValueElem, reflect.ValueOf(v)))
		}
		field.SetMapIndex(ifcMapKeyElem, ifcMapValueElem)
	}
	return nil
}

// setConfigValueMap is a function that sets the value of the map field
func setConfigValueMap(params Params, valFromConfig string, field reflect.Value) error {
	field.Set(reflect.MakeMap(reflect.MapOf(field.Type().Key(), field.Type().Elem())))
	split := splitWithSeparator(valFromConfig, params.Separator)
	for _, s := range split {
		s := strings.TrimSpace(s)
		kv := strings.Split(s, elemKeyValSeparator)
		if len(kv) != 2 {
			return fmt.Errorf("expected format key%vvalue, got %q", elemKeyValSeparator, s)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		ifcKeyElem := reflect.New(field.Type().Key()).Elem()
		if err := setConfigValueHelper(params, key, ifcKeyElem); err != nil {
			return fmt.Errorf("map key %q: %w", key, err)
		}
		ifcValueElem := reflect.New(field.Type().Elem()).Elem()
		if err := setConfigValueHelper(params, val, ifcValueElem); err != nil {
			return fmt.Errorf("map key %q, value %q: %w", key, val, err)
		}
		field.SetMapIndex(ifcKeyElem, ifcValueElem)
	}
	return nil
}

// canRange is a function that checks if the value can be ranged
func canRange(valFromConfig, elemRangeSeparator string, field reflect.Value) bool {
	if elemRangeSeparator == "" {
		return false
	}
	if field.Kind() != reflect.Slice {
		return false
	}
	elemIfc := reflect.New(field.Type().Elem()).Interface()
	elemVal := reflect.ValueOf(elemIfc).Elem()
	if !elemVal.CanInt() {
		return false
	}
	return strings.Contains(valFromConfig, elemRangeSeparator)
}

// splitWithSeparator is a function that splits on default or custom separator
func splitWithSeparator(valFromConfig, customSeparator string) []string {
	separator := ","
	if customSeparator != "" {
		separator = customSeparator
	}
	return strings.Split(valFromConfig, separator)
}

// setConfigValueRange is a function that sets the value of the range field
func setConfigValueRange(params Params, valFromConfig string, field reflect.Value) error {
	rangeSplit := strings.Split(valFromConfig, params.RangeSeparator)
	if len(rangeSplit) != 2 {
		return fmt.Errorf("expected format start%vend, got %q", params.RangeSeparator, valFromConfig)
	}
	start := reflect.New(field.Type().Elem()).Interface()
	end := reflect.New(field.Type().Elem()).Interface()
	if err := json.Unmarshal([]byte(rangeSplit[0]), &start); err != nil {
		return fmt.Errorf("unable to parse start value %q: %w", rangeSplit[0], err)
	}
	if err := json.Unmarshal([]byte(rangeSplit[1]), &end); err != nil {
		return fmt.Errorf("unable to parse end value %q: %w", rangeSplit[1], err)
	}

	startVal := reflect.ValueOf(start).Elem()
	endVal := reflect.ValueOf(end).Elem()
	for i := startVal.Int(); i <= endVal.Int(); i++ {
		elemVal := reflect.New(field.Type().Elem()).Elem()
		elemVal.SetInt(i)
		field.Set(reflect.Append(field, elemVal))
	}
	return nil
}

// setConfigValueSlice is a function that sets the value of the slice field
func setConfigValueSlice(params Params, valFromConfig string, field reflect.Value) error {
	elemIfc := reflect.New(field.Type().Elem()).Interface()
	split := splitWithSeparator(valFromConfig, params.Separator)
	for i, s := range split {
		s := strings.TrimSpace(s)
		if canRange(s, params.RangeSeparator, field) {
			if err := setConfigValueRange(params, s, field); err != nil {
				return fmt.Errorf("slice element %d: %w", i, err)
			}
		} else {
			if err := setConfigValueHelper(params, s, reflect.ValueOf(elemIfc).Elem()); err != nil {
				return fmt.Errorf("slice element %d: %w", i, err)
			}
			field.Set(reflect.Append(field, reflect.ValueOf(elemIfc).Elem()))
		}
	}
	return nil
}

// setParamValueHelper is a function that sets the value of the parameter
func setConfigValueHelper(params Params, valFromConfig string, field reflect.Value) error {
	paramValue := reflect.ValueOf(valFromConfig)
	if paramValue.Type().AssignableTo(field.Type()) {
		field.SetString(valFromConfig)
		return nil
	}
	if paramValue.Type().ConvertibleTo(field.Type()) {
		field.Set(paramValue.Convert(field.Type()))
		return nil
	}
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		// Try to parse as duration string first
		duration, err := time.ParseDuration(valFromConfig)
		if err == nil {
			if duration < 0 {
				return fmt.Errorf("duration cannot be negative: %q", valFromConfig)
			}
			field.Set(reflect.ValueOf(duration))
			return nil
		}
		// If that fails, interpret as number of seconds
		seconds, err := strconv.ParseInt(valFromConfig, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse duration value %q: must be either a duration string (e.g. '30s', '5m') or a number of seconds", valFromConfig)
		}
		if seconds < 0 {
			return fmt.Errorf("duration cannot be negative: %d seconds", seconds)
		}
		field.Set(reflect.ValueOf(time.Duration(seconds) * time.Second))
		return nil
	}
	if field.Type() == reflect.TypeOf(url.Values{}) {
		return setConfigValueURLParams(params, valFromConfig, field)
	}
	if field.Kind() == reflect.Map {
		return setConfigValueMap(params, valFromConfig, field)
	}
	if field.Kind() == reflect.Slice {
		return setConfigValueSlice(params, valFromConfig, field)
	}
	if field.Kind() == reflect.Bool {
		boolVal, err := strconv.ParseBool(valFromConfig)
		if err != nil {
			return fmt.Errorf("unable to parse boolean value %q: %w", valFromConfig, err)
		}
		field.SetBool(boolVal)
		return nil
	}
	if field.CanInterface() {
		ifc := reflect.New(field.Type()).Interface()
		if err := json.Unmarshal([]byte(valFromConfig), &ifc); err != nil {
			return fmt.Errorf("unable to unmarshal to field type %v: %w", field.Type(), err)
		}
		field.Set(reflect.ValueOf(ifc).Elem())
		return nil
	}
	return fmt.Errorf("unable to find matching parser for field type %v", field.Type())
}

// configParamValue is a function that returns the value of the parameter based on the parsing order
func (sc *ScalerConfig) configParamValue(params Params) (string, bool) {
	for _, po := range params.Order {
		var m map[string]string
		for _, key := range params.Names {
			switch po {
			case TriggerMetadata:
				m = sc.TriggerMetadata
			case AuthParams:
				m = sc.AuthParams
			case ResolvedEnv:
				m = sc.ResolvedEnv
				key = sc.TriggerMetadata[fmt.Sprintf("%sFromEnv", key)]
			default:
				// this is checked when parsing the tags but adding as default case to avoid any potential future problems
				return "", false
			}
			param, ok := m[key]
			param = strings.TrimSpace(param)
			if ok && param != "" {
				return param, true
			}
		}
	}
	return "", params.IsNested()
}

// paramsFromTag is a function that returns the Params struct based on the field tag
func paramsFromTag(tag string, field reflect.StructField) (Params, error) {
	params := Params{FieldName: field.Name}
	tagSplit := strings.Split(tag, tagSeparator)
	for _, ts := range tagSplit {
		tsplit := strings.Split(ts, tagKeySeparator)
		tsplit[0] = strings.TrimSpace(tsplit[0])
		switch tsplit[0] {
		case optionalTag:
			if len(tsplit) == 1 {
				params.Optional = true
			}
			if len(tsplit) > 1 {
				params.Optional, _ = strconv.ParseBool(strings.TrimSpace(tsplit[1]))
			}
		case orderTag:
			if len(tsplit) > 1 {
				order := strings.Split(tsplit[1], tagValueSeparator)
				for _, po := range order {
					poTyped := ParsingOrder(strings.TrimSpace(po))
					if !allowedParsingOrderMap[poTyped] {
						apo := slices.Sorted(maps.Keys(allowedParsingOrderMap))
						return params, fmt.Errorf("unknown parsing order value %s, has to be one of %s", po, apo)
					}
					params.Order = append(params.Order, poTyped)
				}
			}
		case nameTag:
			if len(tsplit) > 1 {
				params.Names = strings.Split(strings.TrimSpace(tsplit[1]), tagValueSeparator)
			}
		case deprecatedTag:
			if len(tsplit) == 1 {
				params.Deprecated = deprecatedTag
			} else {
				params.Deprecated = strings.TrimSpace(tsplit[1])
			}
		case deprecatedAnnounceTag:
			if len(tsplit) == 1 {
				params.DeprecatedAnnounce = deprecatedAnnounceTag
			} else {
				params.DeprecatedAnnounce = strings.TrimSpace(tsplit[1])
			}
		case defaultTag:
			if len(tsplit) > 1 {
				params.Default = strings.TrimSpace(tsplit[1])
			}
		case enumTag:
			if len(tsplit) > 1 {
				params.Enum = strings.Split(tsplit[1], tagValueSeparator)
			}
		case exclusiveSetTag:
			if len(tsplit) > 1 {
				params.ExclusiveSet = strings.Split(tsplit[1], tagValueSeparator)
			}
		case rangeTag:
			if len(tsplit) == 1 {
				params.RangeSeparator = "-"
			}
			if len(tsplit) == 2 {
				params.RangeSeparator = strings.TrimSpace(tsplit[1])
			}
		case separatorTag:
			if len(tsplit) > 1 {
				params.Separator = strings.TrimSpace(tsplit[1])
			}
		case "":
			continue
		default:
			return params, fmt.Errorf("unknown tag param %s: %s", tsplit[0], tag)
		}
	}
	return params, nil
}

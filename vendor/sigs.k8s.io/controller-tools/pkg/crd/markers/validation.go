/*
Copyright 2019 The Kubernetes Authors.

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

package markers

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

const (
	validationPrefix = "kubebuilder:validation:"

	SchemalessName        = "kubebuilder:validation:Schemaless"
	ValidationItemsPrefix = validationPrefix + "items:"

	ValidationExactlyOneOfPrefix = validationPrefix + "ExactlyOneOf"
	ValidationAtMostOneOfPrefix  = validationPrefix + "AtMostOneOf"
	ValidationAtLeastOneOfPrefix = validationPrefix + "AtLeastOneOf"
)

// ValidationMarkers lists all available markers that affect CRD schema generation,
// except for the few that don't make sense as type-level markers (see FieldOnlyMarkers).
// All markers start with `+kubebuilder:validation:`, and continue with their type name.
// A copy is produced of all markers that describes types as well, for making types
// reusable and writing complex validations on slice items.
// At last a copy of all markers with the prefix `+kubebuilder:validation:items:` is
// produced for marking slice fields and types.
var ValidationMarkers = mustMakeAllWithPrefix(validationPrefix, markers.DescribesField,

	// numeric markers

	Maximum(0),
	Minimum(0),
	ExclusiveMaximum(false),
	ExclusiveMinimum(false),
	MultipleOf(0),

	// object markers

	MinProperties(0),
	MaxProperties(0),

	// string markers

	MaxLength(0),
	MinLength(0),
	Pattern(""),

	// array markers

	MaxItems(0),
	MinItems(0),
	UniqueItems(false),

	// general markers

	Enum(nil),
	Format(""),
	Type(""),
	XPreserveUnknownFields{},
	XEmbeddedResource{},
	XIntOrString{},
	XValidation{},
)

// TypeOnlyMarkers list type-specific validation markers (i.e. those markers that don't make sense on a field, and thus aren't in ValidationMarkers or FieldOnlyMarkers).
var TypeOnlyMarkers = []*definitionWithHelp{
	must(markers.MakeDefinition(ValidationAtMostOneOfPrefix, markers.DescribesType, AtMostOneOf(nil))).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies a list of field names that must conform to the AtMostOneOf constraint.")),
	must(markers.MakeDefinition(ValidationExactlyOneOfPrefix, markers.DescribesType, ExactlyOneOf(nil))).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies a list of field names that must conform to the ExactlyOneOf constraint.")),
	must(markers.MakeDefinition(ValidationAtLeastOneOfPrefix, markers.DescribesType, AtLeastOneOf(nil))).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies a list of field names that must conform to the AtLeastOneOf constraint.")),
}

// FieldOnlyMarkers list field-specific validation markers (i.e. those markers that don't make
// sense on a type, and thus aren't in ValidationMarkers).
var FieldOnlyMarkers = []*definitionWithHelp{
	must(markers.MakeDefinition("kubebuilder:validation:Required", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is required.")),
	must(markers.MakeDefinition("kubebuilder:validation:Optional", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is optional.")),
	must(markers.MakeDefinition("required", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is required.")),
	must(markers.MakeDefinition("optional", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is optional.")),
	must(markers.MakeDefinition("k8s:required", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is required.")),
	must(markers.MakeDefinition("k8s:optional", markers.DescribesField, struct{}{})).
		WithHelp(markers.SimpleHelp("CRD validation", "specifies that this field is optional.")),
	must(markers.MakeDefinition("nullable", markers.DescribesField, Nullable{})).
		WithHelp(Nullable{}.Help()),

	must(markers.MakeAnyTypeDefinition("kubebuilder:default", markers.DescribesField, Default{})).
		WithHelp(Default{}.Help()),
	must(markers.MakeDefinition("default", markers.DescribesField, KubernetesDefault{})).
		WithHelp(KubernetesDefault{}.Help()),

	must(markers.MakeAnyTypeDefinition("kubebuilder:example", markers.DescribesField, Example{})).
		WithHelp(Example{}.Help()),

	must(markers.MakeDefinition("kubebuilder:validation:EmbeddedResource", markers.DescribesField, XEmbeddedResource{})).
		WithHelp(XEmbeddedResource{}.Help()),

	must(markers.MakeDefinition(SchemalessName, markers.DescribesField, Schemaless{})).
		WithHelp(Schemaless{}.Help()),
}

// ValidationIshMarkers are field-and-type markers that don't fall under the
// :validation: prefix, and/or don't have a name that directly matches their
// type.
var ValidationIshMarkers = []*definitionWithHelp{
	must(markers.MakeDefinition("kubebuilder:pruning:PreserveUnknownFields", markers.DescribesField, XPreserveUnknownFields{})).
		WithHelp(XPreserveUnknownFields{}.Help()),
	must(markers.MakeDefinition("kubebuilder:pruning:PreserveUnknownFields", markers.DescribesType, XPreserveUnknownFields{})).
		WithHelp(XPreserveUnknownFields{}.Help()),

	must(markers.MakeAnyTypeDefinition("kubebuilder:title", markers.DescribesField, Title{})).
		WithHelp(Title{}.Help()),
	must(markers.MakeAnyTypeDefinition("kubebuilder:title", markers.DescribesType, Title{})).
		WithHelp(Title{}.Help()),
}

func init() {
	AllDefinitions = append(AllDefinitions, ValidationMarkers...)

	for _, def := range ValidationMarkers {
		typDef := def.clone()
		typDef.Target = markers.DescribesType
		AllDefinitions = append(AllDefinitions, typDef)

		itemsName := ValidationItemsPrefix + strings.TrimPrefix(def.Name, validationPrefix)

		itemsFieldDef := def.clone()
		itemsFieldDef.Name = itemsName
		itemsFieldDef.Help.Summary = "for array items " + itemsFieldDef.Help.Summary
		AllDefinitions = append(AllDefinitions, itemsFieldDef)

		itemsTypDef := def.clone()
		itemsTypDef.Name = itemsName
		itemsTypDef.Help.Summary = "for array items " + itemsTypDef.Help.Summary
		itemsTypDef.Target = markers.DescribesType
		AllDefinitions = append(AllDefinitions, itemsTypDef)
	}

	AllDefinitions = append(AllDefinitions, FieldOnlyMarkers...)
	AllDefinitions = append(AllDefinitions, TypeOnlyMarkers...)
	AllDefinitions = append(AllDefinitions, ValidationIshMarkers...)
}

// Maximum specifies the maximum numeric value that this field can have.
// +controllertools:marker:generateHelp:category="CRD validation"
type Maximum float64

func (m Maximum) Value() float64 {
	return float64(m)
}

// Minimum specifies the minimum numeric value that this field can have. Negative numbers are supported.
// +controllertools:marker:generateHelp:category="CRD validation"
type Minimum float64

func (m Minimum) Value() float64 {
	return float64(m)
}

// ExclusiveMinimum indicates that the minimum is "up to" but not including that value.
// +controllertools:marker:generateHelp:category="CRD validation"
type ExclusiveMinimum bool

// ExclusiveMaximum indicates that the maximum is "up to" but not including that value.
// +controllertools:marker:generateHelp:category="CRD validation"
type ExclusiveMaximum bool

// MultipleOf specifies that this field must have a numeric value that's a multiple of this one.
// +controllertools:marker:generateHelp:category="CRD validation"
type MultipleOf float64

func (m MultipleOf) Value() float64 {
	return float64(m)
}

// MaxLength specifies the maximum length for this string.
// +controllertools:marker:generateHelp:category="CRD validation"
type MaxLength int

// MinLength specifies the minimum length for this string.
// +controllertools:marker:generateHelp:category="CRD validation"
type MinLength int

// Pattern specifies that this string must match the given regular expression.
// +controllertools:marker:generateHelp:category="CRD validation"
type Pattern string

// MaxItems specifies the maximum length for this list.
// +controllertools:marker:generateHelp:category="CRD validation"
type MaxItems int

// MinItems specifies the minimum length for this list.
// +controllertools:marker:generateHelp:category="CRD validation"
type MinItems int

// UniqueItems specifies that all items in this list must be unique.
// +controllertools:marker:generateHelp:category="CRD validation"
type UniqueItems bool

// MaxProperties restricts the number of keys in an object
// +controllertools:marker:generateHelp:category="CRD validation"
type MaxProperties int

// MinProperties restricts the number of keys in an object
// +controllertools:marker:generateHelp:category="CRD validation"
type MinProperties int

// Enum specifies that this (scalar) field is restricted to the *exact* values specified here.
// +controllertools:marker:generateHelp:category="CRD validation"
type Enum []any

// Format specifies additional "complex" formatting for this field.
//
// For example, a date-time field would be marked as "type: string" and
// "format: date-time".
// +controllertools:marker:generateHelp:category="CRD validation"
type Format string

// Type overrides the type for this field (which defaults to the equivalent of the Go type).
//
// This generally must be paired with custom serialization.  For example, the
// metav1.Time field would be marked as "type: string" and "format: date-time".
// +controllertools:marker:generateHelp:category="CRD validation"
type Type string

// Nullable marks this field as allowing the "null" value.
//
// This is often not necessary, but may be helpful with custom serialization.
// +controllertools:marker:generateHelp:category="CRD validation"
type Nullable struct{}

// Default sets the default value for this field.
//
// A default value will be accepted as any value valid for the
// field. Formatting for common types include: boolean: `true`, string:
// `Cluster`, numerical: `1.24`, array: `{1,2}`, object: `{policy:
// "delete"}`). Defaults should be defined in pruned form, and only best-effort
// validation will be performed. Full validation of a default requires
// submission of the containing CRD to an apiserver.
// +controllertools:marker:generateHelp:category="CRD validation"
type Default struct {
	Value any
}

// Title sets the title for this field.
//
// The title is metadata that makes the OpenAPI documentation more user-friendly,
// making the schema more understandable when viewed in documentation tools.
// It's a metadata field that doesn't affect validation but provides
// important context about what the schema represents.
// +controllertools:marker:generateHelp:category="CRD validation"
type Title struct {
	Value any
}

// KubernetesDefault sets the default value for this field.
//
// A default value will be accepted as any value valid for the field.
// Only JSON-formatted values are accepted. `ref(...)` values are ignored.
// Formatting for common types include: boolean: `true`, string:
// `"Cluster"`, numerical: `1.24`, array: `[1,2]`, object: `{"policy":
// "delete"}`). Defaults should be defined in pruned form, and only best-effort
// validation will be performed. Full validation of a default requires
// submission of the containing CRD to an apiserver.
// +controllertools:marker:generateHelp:category="CRD validation"
type KubernetesDefault struct {
	Value any
}

// Example sets the example value for this field.
//
// An example value will be accepted as any value valid for the
// field. Formatting for common types include: boolean: `true`, string:
// `Cluster`, numerical: `1.24`, array: `{1,2}`, object: `{policy:
// "delete"}`). Examples should be defined in pruned form, and only best-effort
// validation will be performed. Full validation of an example requires
// submission of the containing CRD to an apiserver.
// +controllertools:marker:generateHelp:category="CRD validation"
type Example struct {
	Value any
}

// XPreserveUnknownFields stops the apiserver from pruning fields which are not specified.
//
// By default the apiserver drops unknown fields from the request payload
// during the decoding step. This marker stops the API server from doing so.
// It affects fields recursively, but switches back to normal pruning behaviour
// if nested  properties or additionalProperties are specified in the schema.
// This can either be true or undefined. False
// is forbidden.
//
// NB: The kubebuilder:validation:XPreserveUnknownFields variant is deprecated
// in favor of the kubebuilder:pruning:PreserveUnknownFields variant.  They function
// identically.
// +controllertools:marker:generateHelp:category="CRD processing"
type XPreserveUnknownFields struct{}

// XEmbeddedResource marks a fields as an embedded resource with apiVersion, kind and metadata fields.
//
// An embedded resource is a value that has apiVersion, kind and metadata fields.
// They are validated implicitly according to the semantics of the currently
// running apiserver. It is not necessary to add any additional schema for these
// field, yet it is possible. This can be combined with PreserveUnknownFields.
// +controllertools:marker:generateHelp:category="CRD validation"
type XEmbeddedResource struct{}

// XIntOrString marks a fields as an IntOrString.
//
// This is required when applying patterns or other validations to an IntOrString
// field. Known information about the type is applied during the collapse phase
// and as such is not normally available during marker application.
// +controllertools:marker:generateHelp:category="CRD validation"
type XIntOrString struct{}

// Schemaless marks a field as being a schemaless object.
//
// Schemaless objects are not introspected, so you must provide
// any type and validation information yourself. One use for this
// tag is for embedding fields that hold JSONSchema typed objects.
// Because this field disables all type checking, it is recommended
// to be used only as a last resort.
// +controllertools:marker:generateHelp:category="CRD validation"
type Schemaless struct{}

func hasNumericType(schema *apiextensionsv1.JSONSchemaProps) bool {
	return schema.Type == string(Integer) || schema.Type == string(Number)
}

func hasTextualType(schema *apiextensionsv1.JSONSchemaProps) bool {
	return schema.Type == "string" || schema.XIntOrString
}

func isIntegral(value float64) bool {
	return value == math.Trunc(value) && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// XValidation marks a field as requiring a value for which a given
// expression evaluates to true.
//
// This marker may be repeated to specify multiple expressions, all of
// which must evaluate to true.
// +controllertools:marker:generateHelp:category="CRD validation"
type XValidation struct {
	Rule              string
	Message           string `marker:",optional"`
	MessageExpression string `marker:"messageExpression,optional"`
	Reason            string `marker:"reason,optional"`
	FieldPath         string `marker:"fieldPath,optional"`
	OptionalOldSelf   *bool  `marker:"optionalOldSelf,optional"`
}

// AtMostOneOf adds a validation constraint that allows at most one of the specified fields.
//
// This marker may be repeated to specify multiple AtMostOneOf constraints that are mutually exclusive.
// +controllertools:marker:generateHelp:category="CRD validation"
type AtMostOneOf []string

// ExactlyOneOf adds a validation constraint that allows at exactly one of the specified fields.
//
// This marker may be repeated to specify multiple ExactlyOneOf constraints that are mutually exclusive.
// +controllertools:marker:generateHelp:category="CRD validation"
type ExactlyOneOf []string

// AtLeastOneOf adds a validation constraint that allows at least one of the specified fields.
//
// This marker may be repeated to specify multiple AtLeastOneOf constraints that are mutually exclusive.
// +controllertools:marker:generateHelp:category="CRD validation"
type AtLeastOneOf []string

func (m Maximum) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasNumericType(schema) {
		return fmt.Errorf("must apply maximum to a numeric value, found %s", schema.Type)
	}

	if schema.Type == string(Integer) && !isIntegral(m.Value()) {
		return fmt.Errorf("cannot apply non-integral maximum validation (%v) to integer value", m.Value())
	}

	val := m.Value()
	schema.Maximum = &val
	return nil
}

func (m Minimum) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasNumericType(schema) {
		return fmt.Errorf("must apply minimum to a numeric value, found %s", schema.Type)
	}

	if schema.Type == "integer" && !isIntegral(m.Value()) {
		return fmt.Errorf("cannot apply non-integral minimum validation (%v) to integer value", m.Value())
	}

	val := m.Value()
	schema.Minimum = &val
	return nil
}

func (m ExclusiveMaximum) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasNumericType(schema) {
		return fmt.Errorf("must apply exclusivemaximum to a numeric value, found %s", schema.Type)
	}
	schema.ExclusiveMaximum = bool(m)
	return nil
}

func (m ExclusiveMinimum) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasNumericType(schema) {
		return fmt.Errorf("must apply exclusiveminimum to a numeric value, found %s", schema.Type)
	}

	schema.ExclusiveMinimum = bool(m)
	return nil
}

func (m MultipleOf) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasNumericType(schema) {
		return fmt.Errorf("must apply multipleof to a numeric value, found %s", schema.Type)
	}

	if schema.Type == "integer" && !isIntegral(m.Value()) {
		return fmt.Errorf("cannot apply non-integral multipleof validation (%v) to integer value", m.Value())
	}

	val := m.Value()
	schema.MultipleOf = &val
	return nil
}

func (m MaxLength) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasTextualType(schema) {
		return fmt.Errorf("must apply maxlength to a textual value, found type %q", schema.Type)
	}
	val := int64(m)
	schema.MaxLength = &val
	return nil
}

func (m MinLength) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasTextualType(schema) {
		return fmt.Errorf("must apply minlength to a textual value, found type %q", schema.Type)
	}
	val := int64(m)
	schema.MinLength = &val
	return nil
}

func (m Pattern) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if !hasTextualType(schema) {
		return fmt.Errorf("must apply pattern to a textual value, found type %q", schema.Type)
	}
	schema.Pattern = string(m)
	return nil
}

func (m MaxItems) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if schema.Type != string(Array) {
		return fmt.Errorf("must apply maxitem to an array")
	}
	val := int64(m)
	schema.MaxItems = &val
	return nil
}

func (m MinItems) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if schema.Type != string(Array) {
		return fmt.Errorf("must apply minitems to an array")
	}
	val := int64(m)
	schema.MinItems = &val
	return nil
}

func (m UniqueItems) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if schema.Type != "array" {
		return fmt.Errorf("must apply uniqueitems to an array")
	}
	schema.UniqueItems = bool(m)
	return nil
}

func (m MinProperties) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if schema.Type != "object" {
		return fmt.Errorf("must apply minproperties to an object")
	}
	val := int64(m)
	schema.MinProperties = &val
	return nil
}

func (m MaxProperties) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if schema.Type != "object" {
		return fmt.Errorf("must apply maxproperties to an object")
	}
	val := int64(m)
	schema.MaxProperties = &val
	return nil
}

func (m Enum) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	// TODO(directxman12): this is a bit hacky -- we should
	// probably support AnyType better + using the schema structure
	vals := make([]apiextensionsv1.JSON, len(m))
	for i, val := range m {
		// TODO(directxman12): check actual type with schema type?
		// if we're expecting a string, marshal the string properly...
		// NB(directxman12): we use json.Marshal to ensure we handle JSON escaping properly
		valMarshalled, err := json.Marshal(val)
		if err != nil {
			return err
		}
		vals[i] = apiextensionsv1.JSON{Raw: valMarshalled}
	}
	schema.Enum = vals
	return nil
}

func (m Format) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	schema.Format = string(m)
	return nil
}

// NB(directxman12): we "typecheck" on target schema properties here,
// which means the "Type" marker *must* be applied first.
// TODO(directxman12): find a less hacky way to do this
// (we could preserve ordering of markers, but that feels bad in its own right).

func (m Type) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	schema.Type = string(m)
	return nil
}

func (m Type) ApplyPriority() ApplyPriority {
	return ApplyPriorityDefault - 1
}

func (m Nullable) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	schema.Nullable = true
	return nil
}

// ApplyToSchema defaults are only valid CRDs created with the v1 API
func (m Default) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	marshalledDefault, err := json.Marshal(m.Value)
	if err != nil {
		return err
	}
	if schema.Type == "array" && string(marshalledDefault) == "{}" {
		marshalledDefault = []byte("[]")
	}
	schema.Default = &apiextensionsv1.JSON{Raw: marshalledDefault}
	return nil
}

func (m Default) ApplyPriority() ApplyPriority {
	// explicitly go after +default markers, so kubebuilder-specific defaults get applied last and stomp
	return 10
}

func (m Title) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if m.Value == nil {
		// only apply to the schema if we have a non-nil title
		return nil
	}
	title, isStr := m.Value.(string)
	if !isStr {
		return fmt.Errorf("expected string, got %T", m.Value)
	}
	schema.Title = title
	return nil
}

func (m *KubernetesDefault) ParseMarker(_ string, _ string, restFields string) error {
	if strings.HasPrefix(strings.TrimSpace(restFields), "ref(") {
		// Skip +default=ref(...) values for now, since we don't have a good way to evaluate go constant values via AST.
		// See https://github.com/kubernetes-sigs/controller-tools/pull/938#issuecomment-2096790018
		return nil
	}
	return json.Unmarshal([]byte(restFields), &m.Value)
}

// ApplyToSchema defaults are only valid CRDs created with the v1 API
func (m KubernetesDefault) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if m.Value == nil {
		// only apply to the schema if we have a non-nil default value
		return nil
	}
	marshalledDefault, err := json.Marshal(m.Value)
	if err != nil {
		return err
	}
	schema.Default = &apiextensionsv1.JSON{Raw: marshalledDefault}
	return nil
}

func (m KubernetesDefault) ApplyPriority() ApplyPriority {
	// explicitly go before +kubebuilder:default markers, so kubebuilder-specific defaults get applied last and stomp
	return 9
}

func (m Example) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	marshalledExample, err := json.Marshal(m.Value)
	if err != nil {
		return err
	}
	schema.Example = &apiextensionsv1.JSON{Raw: marshalledExample}
	return nil
}

func (m XPreserveUnknownFields) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	defTrue := true
	schema.XPreserveUnknownFields = &defTrue
	return nil
}

func (m XEmbeddedResource) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	schema.XEmbeddedResource = true
	return nil
}

// NB(JoelSpeed): we use this property in other markers here,
// which means the "XIntOrString" marker *must* be applied first.

func (m XIntOrString) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	schema.XIntOrString = true
	return nil
}

func (m XIntOrString) ApplyPriority() ApplyPriority {
	return ApplyPriorityDefault - 1
}

func (m XValidation) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	var reason *apiextensionsv1.FieldValueErrorReason
	if m.Reason != "" {
		switch m.Reason {
		case string(apiextensionsv1.FieldValueRequired), string(apiextensionsv1.FieldValueInvalid), string(apiextensionsv1.FieldValueForbidden), string(apiextensionsv1.FieldValueDuplicate):
			reason = (*apiextensionsv1.FieldValueErrorReason)(&m.Reason)
		default:
			return fmt.Errorf("invalid reason %s, valid values are %s, %s, %s and %s", m.Reason, apiextensionsv1.FieldValueRequired, apiextensionsv1.FieldValueInvalid, apiextensionsv1.FieldValueForbidden, apiextensionsv1.FieldValueDuplicate)
		}
	}

	schema.XValidations = append(schema.XValidations, apiextensionsv1.ValidationRule{
		Rule:              m.Rule,
		Message:           m.Message,
		MessageExpression: m.MessageExpression,
		Reason:            reason,
		FieldPath:         m.FieldPath,
		OptionalOldSelf:   m.OptionalOldSelf,
	})
	return nil
}

func (XValidation) ApplyPriority() ApplyPriority {
	return ApplyPriorityDefault
}

func (fields AtMostOneOf) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if len(fields) == 0 {
		return nil
	}
	rule := fieldsToOneOfCelRuleStr(fields)
	xvalidation := XValidation{
		Rule:    fmt.Sprintf("%s <= 1", rule),
		Message: fmt.Sprintf("at most one of the fields in %v may be set", fields),
	}
	return xvalidation.ApplyToSchema(schema)
}

func (AtMostOneOf) ApplyPriority() ApplyPriority {
	// explicitly go after XValidation markers so that the ordering is deterministic
	return XValidation{}.ApplyPriority() + 1
}

func (fields ExactlyOneOf) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if len(fields) == 0 {
		return nil
	}
	rule := fieldsToOneOfCelRuleStr(fields)
	xvalidation := XValidation{
		Rule:    fmt.Sprintf("%s == 1", rule),
		Message: fmt.Sprintf("exactly one of the fields in %v must be set", fields),
	}
	return xvalidation.ApplyToSchema(schema)
}

func (ExactlyOneOf) ApplyPriority() ApplyPriority {
	// explicitly go after AtMostOneOf markers so that the ordering is deterministic
	return AtMostOneOf{}.ApplyPriority() + 1
}

func (fields AtLeastOneOf) ApplyToSchema(schema *apiextensionsv1.JSONSchemaProps) error {
	if len(fields) == 0 {
		return nil
	}
	rule := fieldsToOneOfCelRuleStr(fields)
	xvalidation := XValidation{
		Rule:    fmt.Sprintf("%s >= 1", rule),
		Message: fmt.Sprintf("at least one of the fields in %v must be set", fields),
	}
	return xvalidation.ApplyToSchema(schema)
}

func (AtLeastOneOf) ApplyPriority() ApplyPriority {
	// explicitly go after ExactlyOneOf markers so that the ordering is deterministic
	return ExactlyOneOf{}.ApplyPriority() + 1
}

// fieldsToOneOfCelRuleStr converts a slice of field names to a string representation
// [has(self.field1),has(self.field1),...].filter(x, x == true).size()
func fieldsToOneOfCelRuleStr(fields []string) string {
	var list strings.Builder
	list.WriteString("[")
	for i, f := range fields {
		if i > 0 {
			list.WriteString(",")
		}
		list.WriteString("has(self.")
		list.WriteString(f)
		list.WriteString(")")
	}
	list.WriteString("].filter(x,x==true).size()")
	return list.String()
}

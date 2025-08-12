package internal

import (
	"fmt"
	"reflect"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/log"
)

type (
	// SearchAttributes represents a collection of typed search attributes
	SearchAttributes struct {
		untypedValue map[SearchAttributeKey]interface{}
	}

	// SearchAttributeUpdate represents a change to SearchAttributes
	SearchAttributeUpdate func(*SearchAttributes)

	// SearchAttributeKey represents a typed search attribute key.
	SearchAttributeKey interface {
		// GetName of the search attribute.
		GetName() string
		// GetValueType of the search attribute.
		GetValueType() enumspb.IndexedValueType
		// GetReflectType of the search attribute.
		GetReflectType() reflect.Type
	}

	baseSearchAttributeKey struct {
		name        string
		valueType   enumspb.IndexedValueType
		reflectType reflect.Type
	}

	// SearchAttributeKeyString represents a search attribute key for a text attribute type.
	SearchAttributeKeyString struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyKeyword represents a search attribute key for a keyword attribute type.
	SearchAttributeKeyKeyword struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyBool represents a search attribute key for a boolean attribute type.
	SearchAttributeKeyBool struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyInt64 represents a search attribute key for a integer attribute type.
	SearchAttributeKeyInt64 struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyFloat64 represents a search attribute key for a float attribute type.
	SearchAttributeKeyFloat64 struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyTime represents a search attribute key for a date time attribute type.
	SearchAttributeKeyTime struct {
		baseSearchAttributeKey
	}

	// SearchAttributeKeyKeywordList represents a search attribute key for a list of keyword attribute type.
	SearchAttributeKeyKeywordList struct {
		baseSearchAttributeKey
	}
)

// GetName of the search attribute.
func (bk baseSearchAttributeKey) GetName() string {
	return bk.name
}

// GetValueType of the search attribute.
func (bk baseSearchAttributeKey) GetValueType() enumspb.IndexedValueType {
	return bk.valueType
}

// GetReflectType of the search attribute.
func (bk baseSearchAttributeKey) GetReflectType() reflect.Type {
	return bk.reflectType
}

func NewSearchAttributeKeyString(name string) SearchAttributeKeyString {
	return SearchAttributeKeyString{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_TEXT,
			reflectType: reflect.TypeOf(""),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyString) ValueSet(value string) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyString) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyKeyword(name string) SearchAttributeKeyKeyword {
	return SearchAttributeKeyKeyword{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_KEYWORD,
			reflectType: reflect.TypeOf(""),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyKeyword) ValueSet(value string) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyKeyword) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyBool(name string) SearchAttributeKeyBool {
	return SearchAttributeKeyBool{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_BOOL,
			reflectType: reflect.TypeOf(false),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyBool) ValueSet(value bool) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyBool) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyInt64(name string) SearchAttributeKeyInt64 {
	return SearchAttributeKeyInt64{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_INT,
			reflectType: reflect.TypeOf(int64(0)),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyInt64) ValueSet(value int64) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyInt64) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyFloat64(name string) SearchAttributeKeyFloat64 {
	return SearchAttributeKeyFloat64{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_DOUBLE,
			reflectType: reflect.TypeOf(float64(0)),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyFloat64) ValueSet(value float64) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyFloat64) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyTime(name string) SearchAttributeKeyTime {
	return SearchAttributeKeyTime{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_DATETIME,
			reflectType: reflect.TypeOf(time.Time{}),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyTime) ValueSet(value time.Time) SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = value
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyTime) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributeKeyKeywordList(name string) SearchAttributeKeyKeywordList {
	return SearchAttributeKeyKeywordList{
		baseSearchAttributeKey: baseSearchAttributeKey{
			name:        name,
			valueType:   enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST,
			reflectType: reflect.TypeOf([]string{}),
		},
	}
}

// ValueSet creates an update to set the value of the attribute.
func (k SearchAttributeKeyKeywordList) ValueSet(values []string) SearchAttributeUpdate {
	listCopy := append([]string(nil), values...)
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = listCopy
	}
}

// ValueUnset creates an update to remove the attribute.
func (k SearchAttributeKeyKeywordList) ValueUnset() SearchAttributeUpdate {
	return func(sa *SearchAttributes) {
		sa.untypedValue[k] = nil
	}
}

func NewSearchAttributes(attributes ...SearchAttributeUpdate) SearchAttributes {
	sa := SearchAttributes{
		untypedValue: make(map[SearchAttributeKey]interface{}),
	}
	for _, attr := range attributes {
		attr(&sa)
	}
	return sa
}

// GetString gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetString(key SearchAttributeKeyString) (string, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return "", false
	}
	return value.(string), true
}

// GetKeyword gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetKeyword(key SearchAttributeKeyKeyword) (string, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return "", false
	}
	return value.(string), true
}

// GetBool gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetBool(key SearchAttributeKeyBool) (bool, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return false, false
	}
	return value.(bool), true
}

// GetInt64 gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetInt64(key SearchAttributeKeyInt64) (int64, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return 0, false
	}
	return value.(int64), true
}

// GetFloat64 gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetFloat64(key SearchAttributeKeyFloat64) (float64, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return 0.0, false
	}
	return value.(float64), true
}

// GetTime gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetTime(key SearchAttributeKeyTime) (time.Time, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return time.Time{}, false
	}
	return value.(time.Time), true
}

// GetKeywordList gets a value for the given key and whether it was present.
func (sa SearchAttributes) GetKeywordList(key SearchAttributeKeyKeywordList) ([]string, bool) {
	value, ok := sa.untypedValue[key]
	if !ok || value == nil {
		return nil, false
	}
	result := value.([]string)
	// Return a copy to prevent caller from mutating the underlying value
	return append([]string(nil), result...), true
}

// ContainsKey gets whether a key is present.
func (sa SearchAttributes) ContainsKey(key SearchAttributeKey) bool {
	val, ok := sa.untypedValue[key]
	return ok && val != nil
}

// Size gets the size of the attribute collection.
func (sa SearchAttributes) Size() int {
	return len(sa.GetUntypedValues())
}

// GetUntypedValues gets a copy of the collection with raw types.
func (sa SearchAttributes) GetUntypedValues() map[SearchAttributeKey]interface{} {
	untypedValueCopy := make(map[SearchAttributeKey]interface{}, len(sa.untypedValue))
	for key, value := range sa.untypedValue {
		// Filter out nil values
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case []string:
			untypedValueCopy[key] = append([]string(nil), v...)
		default:
			untypedValueCopy[key] = v
		}
	}
	return untypedValueCopy
}

// Copy creates an update that copies existing values.
//
//workflowcheck:ignore
func (sa SearchAttributes) Copy() SearchAttributeUpdate {
	return func(s *SearchAttributes) {
		// GetUntypedValues returns a copy of the map without nil values
		// so the copy won't delete any existing values
		untypedValues := sa.GetUntypedValues()
		for key, value := range untypedValues {
			s.untypedValue[key] = value
		}
	}
}

func serializeUntypedSearchAttributes(input map[string]interface{}) (*commonpb.SearchAttributes, error) {
	if input == nil {
		return nil, nil
	}

	attr := make(map[string]*commonpb.Payload)
	for k, v := range input {
		// If search attribute value is already of Payload type, then use it directly.
		// This allows to copy search attributes from workflow info to child workflow options.
		if vp, ok := v.(*commonpb.Payload); ok {
			attr[k] = vp
			continue
		}
		var err error
		attr[k], err = converter.GetDefaultDataConverter().ToPayload(v)
		if err != nil {
			return nil, fmt.Errorf("encode search attribute [%s] error: %v", k, err)
		}
	}
	return &commonpb.SearchAttributes{IndexedFields: attr}, nil
}

func serializeTypedSearchAttributes(searchAttributes map[SearchAttributeKey]interface{}) (*commonpb.SearchAttributes, error) {
	if searchAttributes == nil {
		return nil, nil
	}

	serializedAttr := make(map[string]*commonpb.Payload)
	for k, v := range searchAttributes {
		payload, err := converter.GetDefaultDataConverter().ToPayload(v)
		if err != nil {
			return nil, fmt.Errorf("encode search attribute [%s] error: %v", k, err)
		}
		// Server does not remove search attributes if they set a type
		if payload.GetData() != nil {
			payload.Metadata["type"] = []byte(k.GetValueType().String())
		}
		serializedAttr[k.GetName()] = payload
	}
	return &commonpb.SearchAttributes{IndexedFields: serializedAttr}, nil
}

func serializeSearchAttributes(
	untypedAttributes map[string]interface{},
	typedAttributes SearchAttributes,
) (*commonpb.SearchAttributes, error) {
	var searchAttr *commonpb.SearchAttributes
	var err error
	if untypedAttributes != nil && typedAttributes.Size() != 0 {
		return nil, fmt.Errorf("cannot specify both SearchAttributes and TypedSearchAttributes")
	} else if untypedAttributes != nil {
		searchAttr, err = serializeUntypedSearchAttributes(untypedAttributes)
		if err != nil {
			return nil, err
		}
	} else if typedAttributes.Size() != 0 {
		searchAttr, err = serializeTypedSearchAttributes(typedAttributes.GetUntypedValues())
		if err != nil {
			return nil, err
		}
	}
	return searchAttr, nil
}

func convertToTypedSearchAttributes(logger log.Logger, attributes map[string]*commonpb.Payload) SearchAttributes {
	updates := make([]SearchAttributeUpdate, 0, len(attributes))
	for key, payload := range attributes {
		if payload.Data == nil {
			continue
		}
		valueType := enumspb.IndexedValueType(
			enumspb.IndexedValueType_shorthandValue[string(payload.GetMetadata()["type"])])
		// For TemporalChangeVersion, we imply the value type
		if valueType == 0 && key == TemporalChangeVersion {
			valueType = enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST
		}
		switch valueType {
		case enumspb.INDEXED_VALUE_TYPE_BOOL:
			attr := NewSearchAttributeKeyBool(key)
			var value bool
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_KEYWORD:
			attr := NewSearchAttributeKeyKeyword(key)
			var value string
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_TEXT:
			attr := NewSearchAttributeKeyString(key)
			var value string
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_INT:
			attr := NewSearchAttributeKeyInt64(key)
			var value int64
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_DOUBLE:
			attr := NewSearchAttributeKeyFloat64(key)
			var value float64
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_DATETIME:
			attr := NewSearchAttributeKeyTime(key)
			var value time.Time
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		case enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST:
			attr := NewSearchAttributeKeyKeywordList(key)
			var value []string
			err := converter.GetDefaultDataConverter().FromPayload(payload, &value)
			if err != nil {
				panic(err)
			}
			updates = append(updates, attr.ValueSet(value))
		default:
			logger.Warn("Unrecognized indexed value type on search attribute key", "key", key, "type", valueType)
		}
	}
	return NewSearchAttributes(updates...)
}

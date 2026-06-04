/*
Copyright 2026 Google LLC

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

package spanner

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/google/uuid"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

var ssInfinity = []byte{0xFF}

type keyRecipePartKind int

const (
	keyRecipePartTag keyRecipePartKind = iota
	keyRecipePartValue
	keyRecipePartInvalid
)

type keyType int

const (
	keyTypeFull keyType = iota
	keyTypePrefix
	keyTypePrefixSuccessor
	keyTypeIndex
)

type encodeState int

const (
	encodeStateOK encodeState = iota
	encodeStateFailed
	encodeStateEndOfKeys
)

type valueLookupStatus int

const (
	valueLookupFound valueLookupStatus = iota
	valueLookupMissing
	valueLookupFailed
)

type valueFinder func(valueIndex int, identifier string) (*structpb.Value, valueLookupStatus)

type keyRecipePart struct {
	kind              keyRecipePartKind
	tag               int
	typeCode          sppb.TypeCode
	ascending         bool
	nullOrder         sppb.KeyRecipe_Part_NullOrder
	identifier        string
	structIdentifiers []int32
	constantValue     *structpb.Value
	random            bool
}

func (p *keyRecipePart) hasConstantValue() bool {
	return p.constantValue != nil
}

func (p *keyRecipePart) shouldConsumeValueIndex() bool {
	return !p.hasConstantValue() && !p.random
}

type keyRecipe struct {
	parts   []keyRecipePart
	isIndex bool
}

func newKeyRecipe(in *sppb.KeyRecipe) (*keyRecipe, error) {
	if in == nil || len(in.GetPart()) == 0 {
		return nil, fmt.Errorf("KeyRecipe must have at least one part")
	}
	parts := make([]keyRecipePart, 0, len(in.GetPart()))
	for _, partProto := range in.GetPart() {
		parts = append(parts, keyRecipePartFromProto(partProto))
	}
	if parts[0].kind != keyRecipePartTag {
		return nil, fmt.Errorf("KeyRecipe must start with a tag")
	}
	return &keyRecipe{parts: parts, isIndex: in.GetIndexName() != ""}, nil
}

func keyRecipePartFromProto(partProto *sppb.KeyRecipe_Part) keyRecipePart {
	if partProto == nil {
		return keyRecipePart{kind: keyRecipePartInvalid}
	}
	if partProto.GetTag() != 0 {
		return keyRecipePart{kind: keyRecipePartTag, tag: int(partProto.GetTag())}
	}
	if partProto.GetType() == nil {
		return keyRecipePart{kind: keyRecipePartInvalid}
	}
	if partProto.GetOrder() != sppb.KeyRecipe_Part_ASCENDING && partProto.GetOrder() != sppb.KeyRecipe_Part_DESCENDING {
		return keyRecipePart{kind: keyRecipePartInvalid}
	}
	if partProto.GetNullOrder() != sppb.KeyRecipe_Part_NULLS_FIRST &&
		partProto.GetNullOrder() != sppb.KeyRecipe_Part_NULLS_LAST &&
		partProto.GetNullOrder() != sppb.KeyRecipe_Part_NOT_NULL {
		return keyRecipePart{kind: keyRecipePartInvalid}
	}
	random := false
	identifier := ""
	var constantValue *structpb.Value
	switch v := partProto.GetValueType().(type) {
	case *sppb.KeyRecipe_Part_Random:
		random = v.Random
	case *sppb.KeyRecipe_Part_Identifier:
		identifier = v.Identifier
	case *sppb.KeyRecipe_Part_Value:
		constantValue = v.Value
	}
	if random && partProto.GetType().GetCode() != sppb.TypeCode_INT64 {
		return keyRecipePart{kind: keyRecipePartInvalid}
	}
	return keyRecipePart{
		kind:              keyRecipePartValue,
		typeCode:          partProto.GetType().GetCode(),
		ascending:         partProto.GetOrder() == sppb.KeyRecipe_Part_ASCENDING,
		nullOrder:         partProto.GetNullOrder(),
		identifier:        identifier,
		structIdentifiers: append([]int32(nil), partProto.GetStructIdentifiers()...),
		constantValue:     constantValue,
		random:            random,
	}
}

func encodeNull(part keyRecipePart, out *[]byte) error {
	switch part.nullOrder {
	case sppb.KeyRecipe_Part_NULLS_FIRST:
		*out = appendNullOrderedFirst(*out)
	case sppb.KeyRecipe_Part_NULLS_LAST:
		*out = appendNullOrderedLast(*out)
	case sppb.KeyRecipe_Part_NOT_NULL:
		return fmt.Errorf("Key part cannot be NULL")
	default:
		return fmt.Errorf("unknown null order: %v", part.nullOrder)
	}
	return nil
}

func encodeNotNull(part keyRecipePart, out *[]byte) error {
	switch part.nullOrder {
	case sppb.KeyRecipe_Part_NULLS_FIRST:
		*out = appendNotNullMarkerNullOrderedFirst(*out)
	case sppb.KeyRecipe_Part_NULLS_LAST:
		*out = appendNotNullMarkerNullOrderedLast(*out)
	case sppb.KeyRecipe_Part_NOT_NULL:
		// no marker
	default:
		return fmt.Errorf("unknown null order: %v", part.nullOrder)
	}
	return nil
}

func encodeSingleValuePart(part keyRecipePart, value *structpb.Value, out *[]byte) error {
	if value == nil {
		return fmt.Errorf("nil value")
	}
	if _, ok := value.GetKind().(*structpb.Value_NullValue); ok {
		return encodeNull(part, out)
	}
	initialLen := len(*out)
	if err := validatePartValue(part, value); err != nil {
		return err
	}
	if err := encodeNotNull(part, out); err != nil {
		return err
	}
	var err error
	switch part.typeCode {
	case sppb.TypeCode_BOOL:
		if part.ascending {
			*out = appendUint64Increasing(*out, boolToUint64(value.GetBoolValue()))
		} else {
			*out = appendUint64Decreasing(*out, boolToUint64(value.GetBoolValue()))
		}
	case sppb.TypeCode_INT64, sppb.TypeCode_ENUM:
		var i int64
		i, err = strconv.ParseInt(value.GetStringValue(), 10, 64)
		if err == nil {
			if part.ascending {
				*out = appendIntIncreasing(*out, i)
			} else {
				*out = appendIntDecreasing(*out, i)
			}
		}
	case sppb.TypeCode_FLOAT64:
		f := 0.0
		if value.GetStringValue() != "" {
			switch value.GetStringValue() {
			case "Infinity":
				f = math.Inf(1)
			case "-Infinity":
				f = math.Inf(-1)
			case "NaN":
				f = math.Float64frombits(0x7ff8000000000000)
			default:
				err = fmt.Errorf("invalid FLOAT64 string: %s", value.GetStringValue())
			}
		} else {
			f = value.GetNumberValue()
		}
		if err == nil {
			if part.ascending {
				*out = appendDoubleIncreasing(*out, f)
			} else {
				*out = appendDoubleDecreasing(*out, f)
			}
		}
	case sppb.TypeCode_STRING:
		if part.ascending {
			*out = appendStringIncreasing(*out, value.GetStringValue())
		} else {
			*out = appendStringDecreasing(*out, value.GetStringValue())
		}
	case sppb.TypeCode_BYTES:
		decoded, decodeErr := base64.StdEncoding.DecodeString(value.GetStringValue())
		if decodeErr != nil {
			err = fmt.Errorf("invalid base64 for BYTES type: %w", decodeErr)
			break
		}
		if part.ascending {
			*out = appendBytesIncreasing(*out, decoded)
		} else {
			*out = appendBytesDecreasing(*out, decoded)
		}
	case sppb.TypeCode_TIMESTAMP:
		seconds, nanos, parseErr := parseTimestamp(value.GetStringValue())
		if parseErr != nil {
			err = parseErr
			break
		}
		encoded := encodeTimestamp(seconds, nanos)
		if part.ascending {
			*out = appendBytesIncreasing(*out, encoded)
		} else {
			*out = appendBytesDecreasing(*out, encoded)
		}
	case sppb.TypeCode_DATE:
		epochDays, parseErr := parseDate(value.GetStringValue())
		if parseErr != nil {
			err = parseErr
			break
		}
		if part.ascending {
			*out = appendIntIncreasing(*out, int64(epochDays))
		} else {
			*out = appendIntDecreasing(*out, int64(epochDays))
		}
	case sppb.TypeCode_UUID:
		high, low, parseErr := parseUUID(value.GetStringValue())
		if parseErr != nil {
			err = parseErr
			break
		}
		encoded := encodeUUID(high, low)
		if part.ascending {
			*out = appendBytesIncreasing(*out, encoded)
		} else {
			*out = appendBytesDecreasing(*out, encoded)
		}
	default:
		err = fmt.Errorf("unsupported type code for ssformat encoding: %v", part.typeCode)
	}
	if err != nil {
		*out = (*out)[:initialLen]
		return err
	}
	return nil
}

func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func validatePartValue(part keyRecipePart, value *structpb.Value) error {
	switch part.typeCode {
	case sppb.TypeCode_BOOL:
		if _, ok := value.GetKind().(*structpb.Value_BoolValue); !ok {
			return fmt.Errorf("type mismatch for BOOL")
		}
	case sppb.TypeCode_INT64, sppb.TypeCode_ENUM:
		if _, ok := value.GetKind().(*structpb.Value_StringValue); !ok {
			return fmt.Errorf("type mismatch for INT64/ENUM, expecting decimal string")
		}
		if _, err := strconv.ParseInt(value.GetStringValue(), 10, 64); err != nil {
			return fmt.Errorf("invalid INT64/ENUM string: %s", value.GetStringValue())
		}
	case sppb.TypeCode_FLOAT64:
		switch value.GetKind().(type) {
		case *structpb.Value_NumberValue:
		case *structpb.Value_StringValue:
			if value.GetStringValue() != "Infinity" && value.GetStringValue() != "-Infinity" && value.GetStringValue() != "NaN" {
				return fmt.Errorf("invalid FLOAT64 string: %s", value.GetStringValue())
			}
		default:
			return fmt.Errorf("type mismatch for FLOAT64")
		}
	case sppb.TypeCode_STRING, sppb.TypeCode_BYTES, sppb.TypeCode_TIMESTAMP, sppb.TypeCode_DATE, sppb.TypeCode_UUID:
		if _, ok := value.GetKind().(*structpb.Value_StringValue); !ok {
			return fmt.Errorf("type mismatch for %v", part.typeCode)
		}
		if part.typeCode == sppb.TypeCode_BYTES {
			if _, err := base64.StdEncoding.DecodeString(value.GetStringValue()); err != nil {
				return fmt.Errorf("invalid base64 for BYTES type")
			}
		}
		if part.typeCode == sppb.TypeCode_TIMESTAMP {
			if _, _, err := parseTimestamp(value.GetStringValue()); err != nil {
				return err
			}
		}
		if part.typeCode == sppb.TypeCode_DATE {
			if _, err := parseDate(value.GetStringValue()); err != nil {
				return err
			}
		}
		if part.typeCode == sppb.TypeCode_UUID {
			if _, _, err := parseUUID(value.GetStringValue()); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type code for ssformat encoding: %v", part.typeCode)
	}
	return nil
}

func encodeRandomValuePart(part keyRecipePart, out *[]byte) {
	v := rand.Int63n(math.MaxInt64)
	if part.ascending {
		*out = appendIntIncreasing(*out, v)
	} else {
		*out = appendIntDecreasing(*out, v)
	}
}

func (r *keyRecipe) resolvePartValue(part keyRecipePart, finder valueFinder, valueIndex int) (*structpb.Value, valueLookupStatus) {
	if part.hasConstantValue() {
		return part.constantValue, valueLookupFound
	}
	value, status := finder(valueIndex, part.identifier)
	if status != valueLookupFound {
		return nil, status
	}
	if len(part.structIdentifiers) == 0 {
		return value, valueLookupFound
	}
	current := value
	for _, structIndex := range part.structIdentifiers {
		listValue := current.GetListValue()
		if listValue == nil || int(structIndex) < 0 || int(structIndex) >= len(listValue.GetValues()) {
			return nil, valueLookupFailed
		}
		current = listValue.GetValues()[structIndex]
	}
	return current, valueLookupFound
}

func (r *keyRecipe) encodeKeyInternal(finder valueFinder, keyType keyType) *targetRange {
	target := newTargetRange(nil, nil, false)
	valueIndex := 0
	state := encodeStateOK
	partIndex := 0
	for ; partIndex < len(r.parts); partIndex++ {
		part := r.parts[partIndex]
		switch part.kind {
		case keyRecipePartTag:
			var err error
			target.start, err = appendCompositeTag(target.start, part.tag)
			if err != nil {
				state = encodeStateFailed
			}
		case keyRecipePartValue:
			if part.random {
				encodeRandomValuePart(part, &target.start)
				continue
			}
			lookupIndex := valueIndex
			if part.shouldConsumeValueIndex() {
				valueIndex++
			}
			value, lookupStatus := r.resolvePartValue(part, finder, lookupIndex)
			switch lookupStatus {
			case valueLookupFound:
				// Continue with encoding for the found value.
			case valueLookupMissing:
				state = encodeStateEndOfKeys
			case valueLookupFailed:
				state = encodeStateFailed
			default:
				// Defensive fallback if valueLookupStatus grows in the future.
				state = encodeStateFailed
			}
			if lookupStatus != valueLookupFound {
				break
			}
			if err := encodeSingleValuePart(part, value, &target.start); err != nil {
				state = encodeStateFailed
			}
		case keyRecipePartInvalid:
			state = encodeStateFailed
		}
		if state != encodeStateOK {
			break
		}
	}

	if partIndex == len(r.parts) || (keyType != keyTypeFull && state == encodeStateEndOfKeys) {
		if keyType == keyTypePrefixSuccessor {
			target.start = makePrefixSuccessor(target.start)
		}
		if keyType == keyTypeIndex {
			target.limit = makePrefixSuccessor(target.start)
		}
		return target
	}

	target.approximate = true
	target.limit = makePrefixSuccessor(target.start)
	return target
}

func (r *keyRecipe) keyToTargetRange(in *structpb.ListValue) *targetRange {
	keyType := keyTypeFull
	if r.isIndex {
		keyType = keyTypeIndex
	}
	return r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
		if in == nil || index < 0 || index >= len(in.GetValues()) {
			return nil, valueLookupMissing
		}
		return in.GetValues()[index], valueLookupFound
	}, keyType)
}

func (r *keyRecipe) keyRangeToTargetRange(in *sppb.KeyRange) *targetRange {
	if in == nil {
		return newTargetRange(nil, makePrefixSuccessor(nil), true)
	}
	var start *targetRange
	switch s := in.StartKeyType.(type) {
	case *sppb.KeyRange_StartClosed:
		start = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			if s.StartClosed == nil || index < 0 || index >= len(s.StartClosed.GetValues()) {
				return nil, valueLookupMissing
			}
			return s.StartClosed.GetValues()[index], valueLookupFound
		}, keyTypePrefix)
	case *sppb.KeyRange_StartOpen:
		start = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			if s.StartOpen == nil || index < 0 || index >= len(s.StartOpen.GetValues()) {
				return nil, valueLookupMissing
			}
			return s.StartOpen.GetValues()[index], valueLookupFound
		}, keyTypePrefixSuccessor)
	default:
		start = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			return nil, valueLookupMissing
		}, keyTypePrefix)
		start.approximate = true
	}

	var limit *targetRange
	switch e := in.EndKeyType.(type) {
	case *sppb.KeyRange_EndClosed:
		limit = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			if e.EndClosed == nil || index < 0 || index >= len(e.EndClosed.GetValues()) {
				return nil, valueLookupMissing
			}
			return e.EndClosed.GetValues()[index], valueLookupFound
		}, keyTypePrefixSuccessor)
	case *sppb.KeyRange_EndOpen:
		limit = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			if e.EndOpen == nil || index < 0 || index >= len(e.EndOpen.GetValues()) {
				return nil, valueLookupMissing
			}
			return e.EndOpen.GetValues()[index], valueLookupFound
		}, keyTypePrefix)
	default:
		limit = r.encodeKeyInternal(func(index int, _ string) (*structpb.Value, valueLookupStatus) {
			return nil, valueLookupMissing
		}, keyTypePrefixSuccessor)
		limit.approximate = true
	}

	out := newTargetRange(start.start, limit.start, start.approximate || limit.approximate)
	if limit.approximate {
		out.limit = limit.limit
	}
	return out
}

func (r *keyRecipe) keySetToTargetRange(in *sppb.KeySet) *targetRange {
	if in == nil {
		return newTargetRange(nil, ssInfinity, true)
	}
	if in.GetAll() {
		return r.keyRangeToTargetRange(&sppb.KeyRange{
			StartKeyType: &sppb.KeyRange_StartClosed{StartClosed: &structpb.ListValue{}},
			EndKeyType:   &sppb.KeyRange_EndClosed{EndClosed: &structpb.ListValue{}},
		})
	}
	if len(in.GetRanges()) == 0 {
		switch len(in.GetKeys()) {
		case 0:
			return newTargetRange(nil, ssInfinity, true)
		case 1:
			return r.keyToTargetRange(in.GetKeys()[0])
		}
	}
	target := newTargetRange(append([]byte(nil), ssInfinity...), nil, false)
	for _, key := range in.GetKeys() {
		target.mergeFrom(r.keyToTargetRange(key))
	}
	for _, keyRange := range in.GetRanges() {
		target.mergeFrom(r.keyRangeToTargetRange(keyRange))
	}
	return target
}

func (r *keyRecipe) queryParamsToTargetRange(in *structpb.Struct) *targetRange {
	fields := map[string]*structpb.Value(nil)
	if in != nil {
		fields = in.GetFields()
	}
	lowercaseFields := map[string]*structpb.Value(nil)
	if len(fields) > 0 {
		fieldNames := make([]string, 0, len(fields))
		for fieldName := range fields {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)

		lowercaseFields = make(map[string]*structpb.Value, len(fieldNames))
		for _, fieldName := range fieldNames {
			lowercaseFields[strings.ToLower(fieldName)] = fields[fieldName]
		}
	}
	return r.encodeKeyInternal(func(index int, identifier string) (*structpb.Value, valueLookupStatus) {
		if identifier == "" {
			return nil, valueLookupMissing
		}
		value, ok := lowercaseFields[strings.ToLower(identifier)]
		if !ok {
			return nil, valueLookupMissing
		}
		return value, valueLookupFound
	}, keyTypeFull)
}

func (r *keyRecipe) mutationToTargetRange(in *sppb.Mutation) *targetRange {
	if in == nil {
		return newTargetRange(nil, ssInfinity, true)
	}
	target := newTargetRange(append([]byte(nil), ssInfinity...), nil, false)

	switch op := in.Operation.(type) {
	case *sppb.Mutation_Insert:
		r.addWriteOperationToTarget(target, op.Insert)
	case *sppb.Mutation_Update:
		r.addWriteOperationToTarget(target, op.Update)
	case *sppb.Mutation_InsertOrUpdate:
		r.addWriteOperationToTarget(target, op.InsertOrUpdate)
	case *sppb.Mutation_Replace:
		r.addWriteOperationToTarget(target, op.Replace)
	case *sppb.Mutation_Delete_:
		target.mergeFrom(r.keySetToTargetRange(op.Delete.GetKeySet()))
	case *sppb.Mutation_Send_:
		target.mergeFrom(r.keyToTargetRange(op.Send.GetKey()))
	case *sppb.Mutation_Ack_:
		target.mergeFrom(r.keyToTargetRange(op.Ack.GetKey()))
	default:
		// Unsupported operation.
	}
	if bytes.Equal(target.start, ssInfinity) {
		return newTargetRange(nil, ssInfinity, true)
	}
	return target
}

func (r *keyRecipe) addWriteOperationToTarget(target *targetRange, write *sppb.Mutation_Write) {
	if write == nil {
		target.approximate = true
		return
	}
	columnToIndex := make(map[string]int, len(write.GetColumns()))
	for i, column := range write.GetColumns() {
		columnToIndex[column] = i
	}
	for _, row := range write.GetValues() {
		rowCopy := row
		item := r.encodeKeyInternal(func(index int, identifier string) (*structpb.Value, valueLookupStatus) {
			if identifier == "" {
				return nil, valueLookupMissing
			}
			columnIndex, ok := columnToIndex[identifier]
			if !ok || columnIndex < 0 || columnIndex >= len(rowCopy.GetValues()) {
				return nil, valueLookupMissing
			}
			return rowCopy.GetValues()[columnIndex], valueLookupFound
		}, keyTypeFull)
		target.mergeFrom(item)
	}
}

func parseDate(dateStr string) (int32, error) {
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid DATE string: %s", dateStr)
	}
	// Keep strict canonical DATE format matching.
	if parsed.Format("2006-01-02") != dateStr {
		return 0, fmt.Errorf("invalid DATE string: %s", dateStr)
	}
	return int32(parsed.UTC().Unix() / 86400), nil
}

func parseTimestamp(ts string) (int64, int32, error) {
	// Keep UTC-only semantics used by recipe goldens.
	if len(ts) == 0 || ts[len(ts)-1] != 'Z' {
		return 0, 0, fmt.Errorf("invalid TIMESTAMP string: %s", ts)
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid TIMESTAMP string: %s", ts)
	}
	return parsed.Unix(), int32(parsed.Nanosecond()), nil
}

func parseUUID(raw string) (int64, int64, error) {
	// Only accept canonical (8-4-4-4-12) and braced ({8-4-4-4-12}) UUID formats.
	// uuid.Parse also accepts UUIDs without dashes which we want to reject.
	if len(raw) != 36 && !(len(raw) == 38 && raw[0] == '{' && raw[37] == '}') {
		return 0, 0, fmt.Errorf("invalid UUID string: %s", raw)
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UUID string: %s", raw)
	}
	high := binary.BigEndian.Uint64(parsed[0:8])
	low := binary.BigEndian.Uint64(parsed[8:16])
	return int64(high), int64(low), nil
}

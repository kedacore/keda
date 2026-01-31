// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package column

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

// This JSON type implementation was done for an experimental Object('JSON') type:
// https://clickhouse.com/docs/en/sql-reference/data-types/object-data-type
// It's already deprecated in ClickHouse and will be removed in the future.
// Since ClickHouse 24.8, the Object('JSON') type is no longer alias for JSON type.
// The new JSON type has been introduced: https://clickhouse.com/docs/en/sql-reference/data-types/newjson
// However, the new JSON type is not supported by the driver yet.
//
// This implementation is kept for backward compatibility and will be removed in the future. TODO: remove this

// inverse mapping - go types to clickhouse types
var kindMappings = map[reflect.Kind]string{
	reflect.String:  "String",
	reflect.Int:     "Int64",
	reflect.Int8:    "Int8",
	reflect.Int16:   "Int16",
	reflect.Int32:   "Int32",
	reflect.Int64:   "Int64",
	reflect.Uint:    "UInt64",
	reflect.Uint8:   "UInt8",
	reflect.Uint16:  "UInt16",
	reflect.Uint32:  "UInt32",
	reflect.Uint64:  "UInt64",
	reflect.Float32: "Float32",
	reflect.Float64: "Float64",
	reflect.Bool:    "Bool",
}

// complex types for which a mapping exists - currently we map to String but could enhance in the future for other types
var typeMappings = map[string]struct{}{
	// currently JSON doesn't support DateTime, Decimal or IP so mapped to String
	"time.Time":       {},
	"decimal.Decimal": {},
	"net.IP":          {},
	"uuid.UUID":       {},
}

type JSON interface {
	Interface
	appendEmptyValue() error
}

type JSONParent interface {
	upsertValue(name string, ct string) (*JSONValue, error)
	upsertList(name string) (*JSONList, error)
	upsertObject(name string) (*JSONObject, error)
	insertEmptyColumn(name string) error
	columnNames() []string
	rows() int
}

func parseType(name string, vType reflect.Type, values any, isArray bool, jCol JSONParent, numEmpty int) error {
	_, ok := typeMappings[vType.String()]
	if !ok {
		return &UnsupportedColumnTypeError{
			t: Type(vType.String()),
		}
	}
	ct := "String"
	if isArray {
		ct = fmt.Sprintf("Array(%s)", ct)
	}
	col, err := jCol.upsertValue(name, ct)
	if err != nil {
		return err
	}
	col.origType = vType

	//pre pad with empty - e.g. for new values in maps
	for i := 0; i < numEmpty; i++ {
		if isArray {
			// empty array for nil of the right type
			err = col.AppendRow([]string{})
		} else {
			// empty value of the type
			err = col.AppendRow(fmt.Sprint(reflect.New(vType).Elem().Interface()))
		}
		if err != nil {
			return err
		}
	}
	if isArray {
		iValues := reflect.ValueOf(values)
		sValues := make([]string, iValues.Len(), iValues.Len())
		for i := 0; i < iValues.Len(); i++ {
			sValues[i] = fmt.Sprint(iValues.Index(i).Interface())
		}
		return col.AppendRow(sValues)
	}
	return col.AppendRow(fmt.Sprint(values))
}

func parsePrimitive(name string, kind reflect.Kind, values any, isArray bool, jCol JSONParent, numEmpty int) error {
	ct, ok := kindMappings[kind]
	if !ok {
		return &UnsupportedColumnTypeError{
			t: Type(fmt.Sprintf("%s - %s", kind, reflect.TypeOf(values).String())),
		}
	}
	var err error
	if isArray {
		ct = fmt.Sprintf("Array(%s)", ct)
		// if we have a []any we will need to cast to the target column type - this will be based on the first
		// values types. Inconsistent slices will fail.
		values, err = convertSlice(values)
		if err != nil {
			return err
		}
	}
	col, err := jCol.upsertValue(name, ct)
	if err != nil {
		return err
	}

	//pre pad with empty - e.g. for new values in maps
	for i := 0; i < numEmpty; i++ {
		if isArray {
			// empty array for nil of the right type
			err = col.AppendRow(reflect.MakeSlice(reflect.TypeOf(values), 0, 0).Interface())
		} else {
			err = col.AppendRow(nil)
		}
		if err != nil {
			return err
		}
	}

	return col.AppendRow(values)
}

// converts a []any of primitives to a typed slice
// maybe this can be done with reflection but likely slower. investigate.
// this uses the first value to determine the type - subsequent values must currently be of the same type - we might cast later
// but wider driver doesn't support e.g. int to int64
func convertSlice(values any) (any, error) {
	rValues := reflect.ValueOf(values)
	if rValues.Len() == 0 || rValues.Index(0).Kind() != reflect.Interface {
		return values, nil
	}
	var fType reflect.Type
	for i := 0; i < rValues.Len(); i++ {
		elem := rValues.Index(i).Elem()
		if elem.IsValid() {
			fType = elem.Type()
			break
		}
	}
	if fType == nil {
		return []any{}, nil
	}
	typedSlice := reflect.MakeSlice(reflect.SliceOf(fType), 0, rValues.Len())
	for i := 0; i < rValues.Len(); i++ {
		value := rValues.Index(i)
		if value.IsNil() {
			typedSlice = reflect.Append(typedSlice, reflect.Zero(fType))
			continue
		}
		if rValues.Index(i).Elem().Type() != fType {
			return nil, &Error{
				ColumnType: fmt.Sprint(fType),
				Err:        fmt.Errorf("inconsistent slices are not supported - expected %s got %s", fType, rValues.Index(i).Elem().Type()),
			}
		}
		typedSlice = reflect.Append(typedSlice, rValues.Index(i).Elem())
	}
	return typedSlice.Interface(), nil
}

func (jCol *JSONList) createNewOffsets(num int) {
	for i := 0; i < num; i++ {
		//single depth so can take 1st
		if jCol.offsets[0].values.col.Rows() == 0 {
			// first entry in the column
			jCol.offsets[0].values.col.Append(0)
		} else {
			// entry for this object to see offset from last - offsets are cumulative
			jCol.offsets[0].values.col.Append(jCol.offsets[0].values.col.Row(jCol.offsets[0].values.col.Rows() - 1))
		}
	}
}

func getStructFieldName(field reflect.StructField) (string, bool) {
	name := field.Name
	tag := field.Tag.Get("json")
	// not a standard but we allow - to omit fields
	if tag == "-" {
		return name, true
	}
	if tag != "" {
		return tag, false
	}
	// support ch tag as well as this is used elsewhere
	tag = field.Tag.Get("ch")
	if tag == "-" {
		return name, true
	}
	if tag != "" {
		return tag, false
	}
	return name, false
}

// ensures numeric keys and ` are escaped properly
func getMapFieldName(name string) string {
	if !escapeColRegex.MatchString(name) {
		return fmt.Sprintf("`%s`", colEscape.Replace(name))
	}
	return colEscape.Replace(name)
}

func parseSlice(name string, values any, jCol JSONParent, preFill int) error {
	fType := reflect.TypeOf(values).Elem()
	sKind := fType.Kind()
	rValues := reflect.ValueOf(values)

	if sKind == reflect.Interface {
		//use the first element to determine if it is a complex or primitive map - after this we need consistent dimensions
		if rValues.Len() == 0 {
			return nil
		}
		var value reflect.Value
		for i := 0; i < rValues.Len(); i++ {
			value = rValues.Index(i).Elem()
			if value.IsValid() {
				break
			}
		}
		if !value.IsValid() {
			return nil
		}
		fType = value.Type()
		sKind = value.Kind()
	}

	if _, ok := typeMappings[fType.String()]; ok {
		return parseType(name, fType, values, true, jCol, preFill)
	} else if sKind == reflect.Struct || sKind == reflect.Map || sKind == reflect.Slice {
		if rValues.Len() == 0 {
			return nil
		}
		col, err := jCol.upsertList(name)
		if err != nil {
			return err
		}
		col.createNewOffsets(preFill + 1)
		for i := 0; i < rValues.Len(); i++ {
			// increment offset
			col.offsets[0].values.col[col.offsets[0].values.col.Rows()-1] += 1
			value := rValues.Index(i)
			sKind = value.Kind()
			if sKind == reflect.Interface {
				sKind = value.Elem().Kind()
			}
			switch sKind {
			case reflect.Struct:
				col.isNested = true
				if err = iterateStruct(value, col, 0); err != nil {
					return err
				}
			case reflect.Map:
				col.isNested = true
				if err = iterateMap(value, col, 0); err != nil {
					return err
				}
			case reflect.Slice:
				if err = parseSlice("", value.Interface(), col, 0); err != nil {
					return err
				}
			default:
				// only happens if slice has a primitive mixed with complex types in a []any
				return &Error{
					ColumnType: fmt.Sprint(sKind),
					Err:        fmt.Errorf("slices must be same dimension in column %s", col.Name()),
				}
			}
		}
		return nil
	}
	return parsePrimitive(name, sKind, values, true, jCol, preFill)
}

func parseStruct(name string, structVal reflect.Value, jCol JSONParent, preFill int) error {
	col, err := jCol.upsertObject(name)
	if err != nil {
		return err
	}
	return iterateStruct(structVal, col, preFill)
}

func iterateStruct(structVal reflect.Value, col JSONParent, preFill int) error {
	// structs generally have consistent field counts but we ignore nil values that are any as we can't infer from
	// these until they occur - so we might need to either backfill when to do occur or insert empty based on previous
	if structVal.Kind() == reflect.Interface {
		// can happen if passed from []any
		structVal = structVal.Elem()
	}

	currentColumns := col.columnNames()
	columnLookup := make(map[string]struct{})
	numRows := col.rows()
	for _, name := range currentColumns {
		columnLookup[name] = struct{}{}
	}
	addedColumns := make([]string, structVal.NumField(), structVal.NumField())
	newColumn := false

	for i := 0; i < structVal.NumField(); i++ {
		fName, omit := getStructFieldName(structVal.Type().Field(i))
		if omit {
			continue
		}
		field := structVal.Field(i)
		if !field.CanInterface() {
			// can't interface - likely not exported so ignore the field
			continue
		}
		kind := field.Kind()
		value := field.Interface()
		fType := field.Type()
		//resolve underlying kind
		if kind == reflect.Interface {
			if value == nil {
				// ignore nil fields
				continue
			}
			kind = reflect.TypeOf(value).Kind()
			field = reflect.ValueOf(value)
			fType = field.Type()
		}
		if _, ok := columnLookup[fName]; !ok && len(currentColumns) > 0 {
			// new column - need to handle missing
			preFill = numRows
			newColumn = true
		}
		if _, ok := typeMappings[fType.String()]; ok {
			if err := parseType(fName, fType, value, false, col, preFill); err != nil {
				return err
			}
		} else {
			switch kind {
			case reflect.Slice:
				if reflect.ValueOf(value).Len() == 0 {
					continue
				}
				if err := parseSlice(fName, value, col, preFill); err != nil {
					return err
				}
			case reflect.Struct:
				if err := parseStruct(fName, field, col, preFill); err != nil {
					return err
				}
			case reflect.Map:
				if err := parseMap(fName, field, col, preFill); err != nil {
					return err
				}
			default:
				if err := parsePrimitive(fName, kind, value, false, col, preFill); err != nil {
					return err
				}
			}
		}
		addedColumns[i] = fName
		if newColumn {
			// reset as otherwise prefill overflow to other fields. But don't reset if this prefill has come from
			// a higher level
			preFill = 0
		}
	}
	// handle missing
	missingColumns := difference(currentColumns, addedColumns)
	for _, name := range missingColumns {
		if err := col.insertEmptyColumn(name); err != nil {
			return err
		}
	}
	return nil
}

func parseMap(name string, mapVal reflect.Value, jCol JSONParent, preFill int) error {
	if mapVal.Type().Key().Kind() != reflect.String {
		return &Error{
			ColumnType: fmt.Sprint(mapVal.Type().Key().Kind()),
			Err:        fmt.Errorf("map keys must be string for column %s", name),
		}
	}
	col, err := jCol.upsertObject(name)
	if err != nil {
		return err
	}
	return iterateMap(mapVal, col, preFill)
}

func iterateMap(mapVal reflect.Value, col JSONParent, preFill int) error {
	// maps can have inconsistent numbers of elements - we must ensure they are consistent in the encoding
	// two inconsistent options - 1. new - map has new columns 2. massing - map has missing columns
	// for (1) we need to update previous, for (2) we need to ensure we add a null entry
	if mapVal.Kind() == reflect.Interface {
		// can happen if passed from []any
		mapVal = mapVal.Elem()
	}

	currentColumns := col.columnNames()
	//gives us a fast lookup for large maps
	columnLookup := make(map[string]struct{})
	numRows := col.rows()
	// true if we need nil values
	for _, name := range currentColumns {
		columnLookup[name] = struct{}{}
	}
	addedColumns := make([]string, len(mapVal.MapKeys()), len(mapVal.MapKeys()))
	newColumn := false
	for i, key := range mapVal.MapKeys() {
		if newColumn {
			// reset as otherwise prefill overflow to other fields. But don't reset if this prefill has come from
			// a higher level
			preFill = 0
		}

		name := getMapFieldName(key.Interface().(string))
		if _, ok := columnLookup[name]; !ok && len(currentColumns) > 0 {
			// new column - need to handle
			preFill = numRows
			newColumn = true
		}
		field := mapVal.MapIndex(key)
		kind := field.Kind()
		fType := field.Type()

		if kind == reflect.Interface {
			if field.Interface() == nil {
				// ignore nil fields
				continue
			}
			kind = reflect.TypeOf(field.Interface()).Kind()
			field = reflect.ValueOf(field.Interface())
			fType = field.Type()
		}
		if _, ok := typeMappings[fType.String()]; ok {
			if err := parseType(name, fType, field.Interface(), false, col, preFill); err != nil {
				return err
			}
		} else {
			switch kind {
			case reflect.Struct:
				if err := parseStruct(name, field, col, preFill); err != nil {
					return err
				}
			case reflect.Slice:
				if err := parseSlice(name, field.Interface(), col, preFill); err != nil {
					return err
				}
			case reflect.Map:
				if err := parseMap(name, field, col, preFill); err != nil {
					return err
				}
			default:
				if err := parsePrimitive(name, kind, field.Interface(), false, col, preFill); err != nil {
					return err
				}
			}
		}
		addedColumns[i] = name
	}
	// handle missing
	missingColumns := difference(currentColumns, addedColumns)
	for _, name := range missingColumns {
		if err := col.insertEmptyColumn(name); err != nil {
			return err
		}
	}
	return nil
}

func appendStructOrMap(jCol *JSONObject, data any) error {
	vData := reflect.ValueOf(data)
	kind := vData.Kind()
	if kind == reflect.Struct {
		return iterateStruct(vData, jCol, 0)
	}
	if kind == reflect.Map {
		if reflect.TypeOf(data).Key().Kind() != reflect.String {
			return &Error{
				ColumnType: fmt.Sprint(reflect.TypeOf(data).Key().Kind()),
				Err:        fmt.Errorf("map keys must be string for column %s", jCol.Name()),
			}
		}
		if jCol.columns == nil && vData.Len() == 0 {
			// if map is empty, we need to create an empty Tuple to make sure subcolumns protocol is happy
			// _dummy is a ClickHouse internal name for empty Tuple subcolumn
			// it has the same effect as `INSERT INTO single_json_type_table VALUES ('{}');`
			jCol.upsertValue("_dummy", "Int8")
			return jCol.insertEmptyColumn("_dummy")
		}
		return iterateMap(vData, jCol, 0)
	}
	return &UnsupportedColumnTypeError{
		t: Type(fmt.Sprint(kind)),
	}
}

type JSONValue struct {
	Interface
	// represents the type e.g. uuid - these may have been mapped to a Column type support by JSON e.g. String
	origType reflect.Type
}

func (jCol *JSONValue) Reset() {
	jCol.Interface.Reset()
}

func (jCol *JSONValue) appendEmptyValue() error {
	switch jCol.Interface.(type) {
	case *Array:
		if jCol.Rows() > 0 {
			return jCol.AppendRow(reflect.MakeSlice(reflect.TypeOf(jCol.Row(0, false)), 0, 0).Interface())
		}
		return &Error{
			ColumnType: "unknown",
			Err:        fmt.Errorf("can't add empty value to column %s - no entries to infer type", jCol.Name()),
		}
	default:
		// can't just append nil here as we need a custom nil value for the type
		if jCol.origType != nil {
			return jCol.AppendRow(fmt.Sprint(reflect.New(jCol.origType).Elem().Interface()))
		}
		return jCol.AppendRow(nil)
	}
}

func (jCol *JSONValue) Type() Type {
	return Type(fmt.Sprintf("%s %s", jCol.Name(), jCol.Interface.Type()))
}

type JSONList struct {
	Array
	name     string
	isNested bool // indicates if this a list of objects i.e. a Nested
}

func (jCol *JSONList) Name() string {
	return jCol.name
}

func (jCol *JSONList) columnNames() []string {
	return jCol.Array.values.(*JSONObject).columnNames()
}

func (jCol *JSONList) rows() int {
	return jCol.values.(*JSONObject).Rows()
}

func createJSONList(name string, tz *time.Location) (jCol *JSONList) {
	// lists are represented as Nested which are in turn encoded as Array(Tuple()). We thus pass a Array(JSONObject())
	// as this encodes like a tuple
	lCol := &JSONList{
		name: name,
	}
	lCol.values = &JSONObject{tz: tz}
	// depth should always be one as nested arrays aren't possible
	lCol.depth = 1
	lCol.scanType = scanTypeSlice
	offsetScanTypes := []reflect.Type{lCol.scanType}
	lCol.offsets = []*offset{{
		scanType: offsetScanTypes[0],
	}}
	return lCol
}

func (jCol *JSONList) appendEmptyValue() error {
	// only need to bump the offsets
	jCol.createNewOffsets(1)
	return nil
}

func (jCol *JSONList) insertEmptyColumn(name string) error {
	return jCol.values.(*JSONObject).insertEmptyColumn(name)
}

func (jCol *JSONList) upsertValue(name string, ct string) (*JSONValue, error) {
	// check if column exists and reuse if same type, error if same name and different type
	jObj := jCol.values.(*JSONObject)
	cols := jObj.columns
	for i := range cols {
		sCol := cols[i]
		if sCol.Name() == name {
			vCol, ok := cols[i].(*JSONValue)
			if !ok {
				sType := cols[i].Type()
				return nil, &Error{
					ColumnType: fmt.Sprint(sType),
					Err:        fmt.Errorf("type mismatch in column %s - expected value, got %s", name, sType),
				}
			}
			tType := vCol.Interface.Type()
			if tType != Type(ct) {
				return nil, &Error{
					ColumnType: ct,
					Err:        fmt.Errorf("type mismatch in column %s - expected %s, got %s", name, tType, ct),
				}
			}
			return vCol, nil
		}
	}
	col, err := Type(ct).Column(name, jObj.tz)
	if err != nil {
		return nil, err
	}
	vCol := &JSONValue{
		Interface: col,
	}
	jCol.values.(*JSONObject).columns = append(cols, vCol) // nolint:gocritic
	return vCol, nil
}

func (jCol *JSONList) upsertList(name string) (*JSONList, error) {
	// check if column exists and reuse if same type, error if same name and different type
	jObj := jCol.values.(*JSONObject)
	cols := jCol.values.(*JSONObject).columns
	for i := range cols {
		sCol := cols[i]
		if sCol.Name() == name {
			sCol, ok := cols[i].(*JSONList)
			if !ok {
				return nil, &Error{
					ColumnType: fmt.Sprint(cols[i].Type()),
					Err:        fmt.Errorf("type mismatch in column %s - expected list, got %s", name, cols[i].Type()),
				}
			}
			return sCol, nil
		}
	}
	lCol := createJSONList(name, jObj.tz)
	jCol.values.(*JSONObject).columns = append(cols, lCol) // nolint:gocritic
	return lCol, nil

}

func (jCol *JSONList) upsertObject(name string) (*JSONObject, error) {
	// check if column exists and reuse if same type, error if same name and different type
	jObj := jCol.values.(*JSONObject)
	cols := jObj.columns
	for i := range cols {
		sCol := cols[i]
		if sCol.Name() == name {
			sCol, ok := cols[i].(*JSONObject)
			if !ok {
				sType := cols[i].Type()
				return nil, &Error{
					ColumnType: fmt.Sprint(sType),
					Err:        fmt.Errorf("type mismatch in column %s, expected object got %s", name, sType),
				}
			}
			return sCol, nil
		}
	}
	// lists are represented as Nested which are in turn encoded as Array(Tuple()). We thus pass a Array(JSONObject())
	// as this encodes like a tuple
	oCol := &JSONObject{
		name: name,
		tz:   jObj.tz,
	}
	jCol.values.(*JSONObject).columns = append(cols, oCol) // nolint:gocritic
	return oCol, nil
}

func (jCol *JSONList) Type() Type {
	cols := jCol.values.(*JSONObject).columns
	subTypes := make([]string, len(cols))
	for i, v := range cols {
		subTypes[i] = string(v.Type())
	}
	// can be a list of lists or a nested
	if jCol.isNested {
		return Type(fmt.Sprintf("%s Nested(%s)", jCol.name, strings.Join(subTypes, ", ")))
	}
	return Type(fmt.Sprintf("%s Array(%s)", jCol.name, strings.Join(subTypes, ", ")))
}

type JSONObject struct {
	columns  []JSON
	name     string
	root     bool
	encoding uint8
	tz       *time.Location
}

func (jCol *JSONObject) Reset() {
	for i := range jCol.columns {
		jCol.columns[i].Reset()
	}
}

func (jCol *JSONObject) Name() string {
	return jCol.name
}

func (jCol *JSONObject) columnNames() []string {
	columns := make([]string, len(jCol.columns), len(jCol.columns))
	for i := range jCol.columns {
		columns[i] = jCol.columns[i].Name()
	}
	return columns
}

func (jCol *JSONObject) rows() int {
	return jCol.Rows()
}

func (jCol *JSONObject) appendEmptyValue() error {
	for i := range jCol.columns {
		if err := jCol.columns[i].appendEmptyValue(); err != nil {
			return err
		}
	}
	return nil
}

func (jCol *JSONObject) insertEmptyColumn(name string) error {
	for i := range jCol.columns {
		if jCol.columns[i].Name() == name {
			if err := jCol.columns[i].appendEmptyValue(); err != nil {
				return err
			}
			return nil
		}
	}
	return &Error{
		ColumnType: "unknown",
		Err:        fmt.Errorf("column %s is missing - empty value cannot be appended", name),
	}
}

func (jCol *JSONObject) upsertValue(name string, ct string) (*JSONValue, error) {
	for i := range jCol.columns {
		sCol := jCol.columns[i]
		if sCol.Name() == name {
			vCol, ok := jCol.columns[i].(*JSONValue)
			if !ok {
				sType := jCol.columns[i].Type()
				return nil, &Error{
					ColumnType: fmt.Sprint(sType),
					Err:        fmt.Errorf("type mismatch in column %s, expected value got %s", name, sType),
				}
			}
			if vCol.Interface.Type() != Type(ct) {
				return nil, &Error{
					ColumnType: ct,
					Err:        fmt.Errorf("type mismatch in column %s, expected %s got %s", name, vCol.Interface.Type(), ct),
				}
			}
			return vCol, nil
		}
	}
	col, err := Type(ct).Column(name, jCol.tz)
	if err != nil {
		return nil, err
	}
	vCol := &JSONValue{
		Interface: col,
	}
	jCol.columns = append(jCol.columns, vCol)
	return vCol, nil
}

func (jCol *JSONObject) upsertList(name string) (*JSONList, error) {
	for i := range jCol.columns {
		sCol := jCol.columns[i]
		if sCol.Name() == name {
			sCol, ok := jCol.columns[i].(*JSONList)
			if !ok {
				sType := jCol.columns[i].Type()
				return nil, &Error{
					ColumnType: fmt.Sprint(sType),
					Err:        fmt.Errorf("type mismatch in column %s, expected list got %s", name, sType),
				}
			}
			return sCol, nil
		}
	}
	lCol := createJSONList(name, jCol.tz)
	jCol.columns = append(jCol.columns, lCol)
	return lCol, nil
}

func (jCol *JSONObject) upsertObject(name string) (*JSONObject, error) {
	// check if it exists
	for i := range jCol.columns {
		sCol := jCol.columns[i]
		if sCol.Name() == name {
			sCol, ok := jCol.columns[i].(*JSONObject)
			if !ok {
				sType := jCol.columns[i].Type()
				return nil, &Error{
					ColumnType: fmt.Sprint(sType),
					Err:        fmt.Errorf("type mismatch in column %s, expected object got %s", name, sType),
				}
			}
			return sCol, nil
		}
	}
	// not present so create
	oCol := &JSONObject{
		name: name,
		tz:   jCol.tz,
	}
	jCol.columns = append(jCol.columns, oCol)
	return oCol, nil
}

func (jCol *JSONObject) Type() Type {
	if jCol.root {
		return "Object('json')"
	}
	return jCol.FullType()
}

func (jCol *JSONObject) FullType() Type {
	subTypes := make([]string, len(jCol.columns))
	for i, v := range jCol.columns {
		subTypes[i] = string(v.Type())
	}
	if jCol.root {
		return Type(fmt.Sprintf("Tuple(%s)", strings.Join(subTypes, ", ")))
	}
	return Type(fmt.Sprintf("%s Tuple(%s)", jCol.name, strings.Join(subTypes, ", ")))
}

func (jCol *JSONObject) ScanType() reflect.Type {
	return scanTypeMap
}

func (jCol *JSONObject) Rows() int {
	if len(jCol.columns) != 0 {
		return jCol.columns[0].Rows()
	}
	return 0
}

// ClickHouse returns JSON as a tuple i.e. these will never be invoked

func (jCol *JSONObject) Row(i int, ptr bool) any {
	panic("Not implemented")
}

func (jCol *JSONObject) ScanRow(dest any, row int) error {
	panic("Not implemented")
}

func (jCol *JSONObject) Append(v any) (nulls []uint8, err error) {
	jSlice := reflect.ValueOf(v)
	if jSlice.Kind() != reflect.Slice {
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(jCol.Type()),
			From: fmt.Sprintf("slice of structs/map or strings required - received %T", v),
		}
	}
	for i := 0; i < jSlice.Len(); i++ {
		if err := jCol.AppendRow(jSlice.Index(i).Interface()); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (jCol *JSONObject) AppendRow(v any) error {
	if reflect.ValueOf(v).Kind() == reflect.Struct || reflect.ValueOf(v).Kind() == reflect.Map {
		if jCol.columns != nil && jCol.encoding == 1 {
			return &Error{
				ColumnType: fmt.Sprint(jCol.Type()),
				Err:        fmt.Errorf("encoding of JSON columns cannot be mixed in a batch - %s cannot be added as previously String", reflect.ValueOf(v).Kind()),
			}
		}
		err := appendStructOrMap(jCol, v)
		return err
	}
	switch v := v.(type) {
	case string:
		if jCol.columns != nil && jCol.encoding == 0 {
			return &Error{
				ColumnType: fmt.Sprint(jCol.Type()),
				Err:        fmt.Errorf("encoding of JSON columns cannot be mixed in a batch - %s cannot be added as previously Struct/Map", reflect.ValueOf(v).Kind()),
			}
		}
		jCol.encoding = 1
		if jCol.columns == nil {
			jCol.columns = append(jCol.columns, &JSONValue{Interface: &String{}})
		}
		jCol.columns[0].AppendRow(v)
	default:
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "String",
			From: fmt.Sprintf("json row must be struct, map or string - received %T", v),
		}
	}
	return nil
}

func (jCol *JSONObject) Decode(reader *proto.Reader, rows int) error {
	panic("Not implemented")
}

func (jCol *JSONObject) Encode(buffer *proto.Buffer) {
	if jCol.root && jCol.encoding == 0 {
		buffer.PutString(string(jCol.FullType()))
	}
	for _, c := range jCol.columns {
		c.Encode(buffer)
	}
}

func (jCol *JSONObject) ReadStatePrefix(reader *proto.Reader) error {
	_, err := reader.UInt8()
	return err
}

func (jCol *JSONObject) WriteStatePrefix(buffer *proto.Buffer) error {
	buffer.PutUInt8(jCol.encoding)
	return nil
}

var (
	_ Interface           = (*JSONObject)(nil)
	_ CustomSerialization = (*JSONObject)(nil)
)

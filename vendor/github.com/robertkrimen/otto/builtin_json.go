package otto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type builtinJSONParseContext struct {
	reviver Value
	call    FunctionCall
}

func builtinJSONParse(call FunctionCall) Value {
	ctx := builtinJSONParseContext{
		call: call,
	}
	revive := false
	if reviver := call.Argument(1); reviver.isCallable() {
		revive = true
		ctx.reviver = reviver
	}

	var root interface{}
	err := json.Unmarshal([]byte(call.Argument(0).string()), &root)
	if err != nil {
		panic(call.runtime.panicSyntaxError(err.Error()))
	}
	value, exists := builtinJSONParseWalk(ctx, root)
	if !exists {
		value = Value{}
	}
	if revive {
		root := ctx.call.runtime.newObject()
		root.put("", value, false)
		return builtinJSONReviveWalk(ctx, root, "")
	}
	return value
}

func builtinJSONReviveWalk(ctx builtinJSONParseContext, holder *object, name string) Value {
	value := holder.get(name)
	if obj := value.object(); obj != nil {
		if isArray(obj) {
			length := int64(objectLength(obj))
			for index := range length {
				idxName := arrayIndexToString(index)
				idxValue := builtinJSONReviveWalk(ctx, obj, idxName)
				if idxValue.IsUndefined() {
					obj.delete(idxName, false)
				} else {
					obj.defineProperty(idxName, idxValue, 0o111, false)
				}
			}
		} else {
			obj.enumerate(false, func(name string) bool {
				enumVal := builtinJSONReviveWalk(ctx, obj, name)
				if enumVal.IsUndefined() {
					obj.delete(name, false)
				} else {
					obj.defineProperty(name, enumVal, 0o111, false)
				}
				return true
			})
		}
	}
	return ctx.reviver.call(ctx.call.runtime, objectValue(holder), name, value)
}

func builtinJSONParseWalk(ctx builtinJSONParseContext, rawValue interface{}) (Value, bool) {
	switch value := rawValue.(type) {
	case nil:
		return nullValue, true
	case bool:
		return boolValue(value), true
	case string:
		return stringValue(value), true
	case float64:
		return float64Value(value), true
	case []interface{}:
		arrayValue := make([]Value, len(value))
		for index, rawValue := range value {
			if value, exists := builtinJSONParseWalk(ctx, rawValue); exists {
				arrayValue[index] = value
			}
		}
		return objectValue(ctx.call.runtime.newArrayOf(arrayValue)), true
	case map[string]interface{}:
		obj := ctx.call.runtime.newObject()
		for name, rawValue := range value {
			if value, exists := builtinJSONParseWalk(ctx, rawValue); exists {
				obj.put(name, value, false)
			}
		}
		return objectValue(obj), true
	}
	return Value{}, false
}

type builtinJSONStringifyContext struct {
	replacerFunction *Value
	gap              string
	stack            []*object
	propertyList     []string
	call             FunctionCall
}

func builtinJSONStringify(call FunctionCall) Value {
	ctx := builtinJSONStringifyContext{
		call:  call,
		stack: []*object{nil},
	}
	replacer := call.Argument(1).object()
	if replacer != nil {
		if isArray(replacer) {
			length := objectLength(replacer)
			seen := map[string]bool{}
			propertyList := make([]string, length)
			length = 0
			for index := range propertyList {
				value := replacer.get(arrayIndexToString(int64(index)))
				switch value.kind {
				case valueObject:
					switch value.value.(*object).class {
					case classStringName, classNumberName:
					default:
						continue
					}
				case valueString, valueNumber:
				default:
					continue
				}
				name := value.string()
				if seen[name] {
					continue
				}
				seen[name] = true
				length++
				propertyList[index] = name
			}
			ctx.propertyList = propertyList[0:length]
		} else if replacer.class == classFunctionName {
			value := objectValue(replacer)
			ctx.replacerFunction = &value
		}
	}
	if spaceValue, exists := call.getArgument(2); exists {
		if spaceValue.kind == valueObject {
			switch spaceValue.value.(*object).class {
			case classStringName:
				spaceValue = stringValue(spaceValue.string())
			case classNumberName:
				spaceValue = spaceValue.numberValue()
			}
		}
		switch spaceValue.kind {
		case valueString:
			value := spaceValue.string()
			if len(value) > 10 {
				ctx.gap = value[0:10]
			} else {
				ctx.gap = value
			}
		case valueNumber:
			value := spaceValue.number().int64
			if value > 10 {
				value = 10
			} else if value < 0 {
				value = 0
			}
			ctx.gap = strings.Repeat(" ", int(value))
		}
	}
	holder := call.runtime.newObject()
	holder.put("", call.Argument(0), false)
	value, exists := builtinJSONStringifyWalk(ctx, "", holder)
	if !exists {
		return Value{}
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		panic(call.runtime.panicTypeError("JSON.stringify marshal: %s", err))
	}
	if ctx.gap != "" {
		valueJSON1 := bytes.Buffer{}
		if err = json.Indent(&valueJSON1, valueJSON, "", ctx.gap); err != nil {
			panic(call.runtime.panicTypeError("JSON.stringify indent: %s", err))
		}
		valueJSON = valueJSON1.Bytes()
	}
	return stringValue(string(valueJSON))
}

func builtinJSONStringifyWalk(ctx builtinJSONStringifyContext, key string, holder *object) (interface{}, bool) {
	value := holder.get(key)

	if value.IsObject() {
		obj := value.object()
		if toJSON := obj.get("toJSON"); toJSON.IsFunction() {
			value = toJSON.call(ctx.call.runtime, value, key)
		} else if obj.objectClass.marshalJSON != nil {
			// If the object is a GoStruct or something that implements json.Marshaler
			marshaler := obj.objectClass.marshalJSON(obj)
			if marshaler != nil {
				return marshaler, true
			}
		}
	}

	if ctx.replacerFunction != nil {
		value = ctx.replacerFunction.call(ctx.call.runtime, objectValue(holder), key, value)
	}

	if value.kind == valueObject {
		switch value.value.(*object).class {
		case classBooleanName:
			value = value.object().value.(Value)
		case classStringName:
			value = stringValue(value.string())
		case classNumberName:
			value = value.numberValue()
		}
	}

	switch value.kind {
	case valueBoolean:
		return value.bool(), true
	case valueString:
		return value.string(), true
	case valueNumber:
		integer := value.number()
		switch integer.kind {
		case numberInteger:
			return integer.int64, true
		case numberFloat:
			return integer.float64, true
		default:
			return nil, true
		}
	case valueNull:
		return nil, true
	case valueObject:
		objHolder := value.object()
		if value := value.object(); nil != value {
			for _, obj := range ctx.stack {
				if objHolder == obj {
					panic(ctx.call.runtime.panicTypeError("Converting circular structure to JSON"))
				}
			}
			ctx.stack = append(ctx.stack, value)
			defer func() { ctx.stack = ctx.stack[:len(ctx.stack)-1] }()
		}
		if isArray(objHolder) {
			var length uint32
			switch value := objHolder.get(propertyLength).value.(type) {
			case uint32:
				length = value
			case int:
				if value >= 0 {
					length = uint32(value)
				}
			default:
				panic(ctx.call.runtime.panicTypeError(fmt.Sprintf("JSON.stringify: invalid length: %v (%[1]T)", value)))
			}
			array := make([]interface{}, length)
			for index := range array {
				name := arrayIndexToString(int64(index))
				value, _ := builtinJSONStringifyWalk(ctx, name, objHolder)
				array[index] = value
			}
			return array, true
		} else if objHolder.class != classFunctionName {
			obj := map[string]interface{}{}
			if ctx.propertyList != nil {
				for _, name := range ctx.propertyList {
					value, exists := builtinJSONStringifyWalk(ctx, name, objHolder)
					if exists {
						obj[name] = value
					}
				}
			} else {
				// Go maps are without order, so this doesn't conform to the ECMA ordering
				// standard, but oh well...
				objHolder.enumerate(false, func(name string) bool {
					value, exists := builtinJSONStringifyWalk(ctx, name, objHolder)
					if exists {
						obj[name] = value
					}
					return true
				})
			}
			return obj, true
		}
	}
	return nil, false
}

package otto

// constructFunction.
type constructFunction func(*object, []Value) Value

// 13.2.2 [[Construct]].
func defaultConstruct(fn *object, argumentList []Value) Value {
	obj := fn.runtime.newObject()
	obj.class = classObjectName

	prototype := fn.get("prototype")
	if prototype.kind != valueObject {
		prototype = objectValue(fn.runtime.global.ObjectPrototype)
	}
	obj.prototype = prototype.object()

	this := objectValue(obj)
	value := fn.call(this, argumentList, false, nativeFrame)
	if value.kind == valueObject {
		return value
	}
	return this
}

// nativeFunction.
type nativeFunction func(FunctionCall) Value

// nativeFunctionObject.
type nativeFunctionObject struct {
	call      nativeFunction
	construct constructFunction
	name      string
	file      string
	line      int
}

func (rt *runtime) newNativeFunctionProperty(name, file string, line int, native nativeFunction, length int) *object {
	o := rt.newClassObject(classFunctionName)
	o.value = nativeFunctionObject{
		name:      name,
		file:      file,
		line:      line,
		call:      native,
		construct: defaultConstruct,
	}
	o.defineProperty("name", stringValue(name), 0o000, false)
	o.defineProperty(propertyLength, intValue(length), 0o000, false)
	return o
}

func (rt *runtime) newNativeFunctionObject(name, file string, line int, native nativeFunction, length int) *object {
	o := rt.newNativeFunctionProperty(name, file, line, native, length)
	o.defineOwnProperty("caller", property{
		value: propertyGetSet{
			rt.newNativeFunctionProperty("get", "internal", 0, func(fc FunctionCall) Value {
				for sc := rt.scope; sc != nil; sc = sc.outer {
					if sc.frame.fn == o {
						if sc.outer == nil || sc.outer.frame.fn == nil {
							return nullValue
						}

						return rt.toValue(sc.outer.frame.fn)
					}
				}

				return nullValue
			}, 0),
			&nilGetSetObject,
		},
		mode: 0o000,
	}, false)
	return o
}

// bindFunctionObject.
type bindFunctionObject struct {
	target       *object
	this         Value
	argumentList []Value
}

func (rt *runtime) newBoundFunctionObject(target *object, this Value, argumentList []Value) *object {
	o := rt.newClassObject(classFunctionName)
	o.value = bindFunctionObject{
		target:       target,
		this:         this,
		argumentList: argumentList,
	}
	length := int(toInt32(target.get(propertyLength)))
	length -= len(argumentList)
	if length < 0 {
		length = 0
	}
	o.defineProperty("name", stringValue("bound "+target.get("name").String()), 0o000, false)
	o.defineProperty(propertyLength, intValue(length), 0o000, false)
	o.defineProperty("caller", Value{}, 0o000, false)    // TODO Should throw a TypeError
	o.defineProperty("arguments", Value{}, 0o000, false) // TODO Should throw a TypeError
	return o
}

// [[Construct]].
func (fn bindFunctionObject) construct(argumentList []Value) Value {
	obj := fn.target
	switch value := obj.value.(type) {
	case nativeFunctionObject:
		return value.construct(obj, fn.argumentList)
	case nodeFunctionObject:
		argumentList = append(fn.argumentList, argumentList...)
		return obj.construct(argumentList)
	default:
		panic(fn.target.runtime.panicTypeError("construct unknown type %T", obj.value))
	}
}

// nodeFunctionObject.
type nodeFunctionObject struct {
	node  *nodeFunctionLiteral
	stash stasher
}

func (rt *runtime) newNodeFunctionObject(node *nodeFunctionLiteral, stash stasher) *object {
	o := rt.newClassObject(classFunctionName)
	o.value = nodeFunctionObject{
		node:  node,
		stash: stash,
	}
	o.defineProperty("name", stringValue(node.name), 0o000, false)
	o.defineProperty(propertyLength, intValue(len(node.parameterList)), 0o000, false)
	o.defineOwnProperty("caller", property{
		value: propertyGetSet{
			rt.newNativeFunction("get", "internal", 0, func(fc FunctionCall) Value {
				for sc := rt.scope; sc != nil; sc = sc.outer {
					if sc.frame.fn == o {
						if sc.outer == nil || sc.outer.frame.fn == nil {
							return nullValue
						}

						return rt.toValue(sc.outer.frame.fn)
					}
				}

				return nullValue
			}),
			&nilGetSetObject,
		},
		mode: 0o000,
	}, false)
	return o
}

// _object.
func (o *object) isCall() bool {
	switch fn := o.value.(type) {
	case nativeFunctionObject:
		return fn.call != nil
	case bindFunctionObject:
		return true
	case nodeFunctionObject:
		return true
	default:
		return false
	}
}

func (o *object) call(this Value, argumentList []Value, eval bool, frm frame) Value { //nolint:unparam // Isn't currently used except in recursive self.
	switch fn := o.value.(type) {
	case nativeFunctionObject:
		// Since eval is a native function, we only have to check for it here
		if eval {
			eval = o == o.runtime.eval // If eval is true, then it IS a direct eval
		}

		// Enter a scope, name from the native object...
		rt := o.runtime
		if rt.scope != nil && !eval {
			rt.enterFunctionScope(rt.scope.lexical, this)
			rt.scope.frame = frame{
				native:     true,
				nativeFile: fn.file,
				nativeLine: fn.line,
				callee:     fn.name,
				file:       nil,
				fn:         o,
			}
			defer func() {
				rt.leaveScope()
			}()
		}

		return fn.call(FunctionCall{
			runtime: o.runtime,
			eval:    eval,

			This:         this,
			ArgumentList: argumentList,
			Otto:         o.runtime.otto,
		})

	case bindFunctionObject:
		// TODO Passthrough site, do not enter a scope
		argumentList = append(fn.argumentList, argumentList...)
		return fn.target.call(fn.this, argumentList, false, frm)

	case nodeFunctionObject:
		rt := o.runtime
		stash := rt.enterFunctionScope(fn.stash, this)
		rt.scope.frame = frame{
			callee: fn.node.name,
			file:   fn.node.file,
			fn:     o,
		}
		defer func() {
			rt.leaveScope()
		}()
		callValue := rt.cmplCallNodeFunction(o, stash, fn.node, argumentList)
		if value, valid := callValue.value.(result); valid {
			return value.value
		}
		return callValue
	}

	panic(o.runtime.panicTypeError("%v is not a function", objectValue(o)))
}

func (o *object) construct(argumentList []Value) Value {
	switch fn := o.value.(type) {
	case nativeFunctionObject:
		if fn.call == nil {
			panic(o.runtime.panicTypeError("%v is not a function", objectValue(o)))
		}
		if fn.construct == nil {
			panic(o.runtime.panicTypeError("%v is not a constructor", objectValue(o)))
		}
		return fn.construct(o, argumentList)

	case bindFunctionObject:
		return fn.construct(argumentList)

	case nodeFunctionObject:
		return defaultConstruct(o, argumentList)
	}

	panic(o.runtime.panicTypeError("%v is not a function", objectValue(o)))
}

// 15.3.5.3.
func (o *object) hasInstance(of Value) bool {
	if !o.isCall() {
		// We should not have a hasInstance method
		panic(o.runtime.panicTypeError("Object.hasInstance not callable"))
	}
	if !of.IsObject() {
		return false
	}
	prototype := o.get("prototype")
	if !prototype.IsObject() {
		panic(o.runtime.panicTypeError("Object.hasInstance prototype %q is not an object", prototype))
	}
	prototypeObject := prototype.object()

	value := of.object().prototype
	for value != nil {
		if value == prototypeObject {
			return true
		}
		value = value.prototype
	}
	return false
}

// FunctionCall is an encapsulation of a JavaScript function call.
type FunctionCall struct {
	This         Value
	runtime      *runtime
	thisObj      *object
	Otto         *Otto
	ArgumentList []Value
	eval         bool
}

// Argument will return the value of the argument at the given index.
//
// If no such argument exists, undefined is returned.
func (f FunctionCall) Argument(index int) Value {
	return valueOfArrayIndex(f.ArgumentList, index)
}

func (f FunctionCall) getArgument(index int) (Value, bool) {
	return getValueOfArrayIndex(f.ArgumentList, index)
}

func (f FunctionCall) slice(index int) []Value {
	if index < len(f.ArgumentList) {
		return f.ArgumentList[index:]
	}
	return []Value{}
}

func (f *FunctionCall) thisObject() *object {
	if f.thisObj == nil {
		this := f.This.resolve() // FIXME Is this right?
		f.thisObj = f.runtime.toObject(this)
	}
	return f.thisObj
}

func (f *FunctionCall) thisClassObject(class string) *object {
	if o := f.thisObject(); o.class != class {
		panic(f.runtime.panicTypeError("Function.Class %s != %s", o.class, class))
	}
	return f.thisObj
}

func (f FunctionCall) toObject(value Value) *object {
	return f.runtime.toObject(value)
}

// CallerLocation will return file location information (file:line:pos) where this function is being called.
func (f FunctionCall) CallerLocation() string {
	// see error.go for location()
	return f.runtime.scope.outer.frame.location()
}

package otto

type object struct {
	value         interface{}
	runtime       *runtime
	objectClass   *objectClass
	prototype     *object
	property      map[string]property
	class         string
	propertyOrder []string
	extensible    bool
}

func newObject(rt *runtime, class string) *object {
	o := &object{
		runtime:     rt,
		class:       class,
		objectClass: classObject,
		property:    make(map[string]property),
		extensible:  true,
	}
	return o
}

// 8.12

// 8.12.1.
func (o *object) getOwnProperty(name string) *property {
	return o.objectClass.getOwnProperty(o, name)
}

// 8.12.2.
func (o *object) getProperty(name string) *property {
	return o.objectClass.getProperty(o, name)
}

// 8.12.3.
func (o *object) get(name string) Value {
	return o.objectClass.get(o, name)
}

// 8.12.4.
func (o *object) canPut(name string) bool {
	return o.objectClass.canPut(o, name)
}

// 8.12.5.
func (o *object) put(name string, value Value, throw bool) {
	o.objectClass.put(o, name, value, throw)
}

// 8.12.6.
func (o *object) hasProperty(name string) bool {
	return o.objectClass.hasProperty(o, name)
}

func (o *object) hasOwnProperty(name string) bool {
	return o.objectClass.hasOwnProperty(o, name)
}

type defaultValueHint int

const (
	defaultValueNoHint defaultValueHint = iota
	defaultValueHintString
	defaultValueHintNumber
)

// 8.12.8.
func (o *object) DefaultValue(hint defaultValueHint) Value {
	if hint == defaultValueNoHint {
		if o.class == classDateName {
			// Date exception
			hint = defaultValueHintString
		} else {
			hint = defaultValueHintNumber
		}
	}
	methodSequence := []string{"valueOf", "toString"}
	if hint == defaultValueHintString {
		methodSequence = []string{"toString", "valueOf"}
	}
	for _, methodName := range methodSequence {
		method := o.get(methodName)
		// FIXME This is redundant...
		if method.isCallable() {
			result := method.object().call(objectValue(o), nil, false, nativeFrame)
			if result.IsPrimitive() {
				return result
			}
		}
	}

	panic(o.runtime.panicTypeError("Object.DefaultValue unknown"))
}

func (o *object) String() string {
	return o.DefaultValue(defaultValueHintString).string()
}

func (o *object) defineProperty(name string, value Value, mode propertyMode, throw bool) bool { //nolint:unparam
	return o.defineOwnProperty(name, property{value, mode}, throw)
}

// 8.12.9.
func (o *object) defineOwnProperty(name string, descriptor property, throw bool) bool {
	return o.objectClass.defineOwnProperty(o, name, descriptor, throw)
}

func (o *object) delete(name string, throw bool) bool {
	return o.objectClass.delete(o, name, throw)
}

func (o *object) enumerate(all bool, each func(string) bool) {
	o.objectClass.enumerate(o, all, each)
}

func (o *object) readProperty(name string) (property, bool) {
	prop, exists := o.property[name]
	return prop, exists
}

func (o *object) writeProperty(name string, value interface{}, mode propertyMode) {
	if value == nil {
		value = Value{}
	}
	if _, exists := o.property[name]; !exists {
		o.propertyOrder = append(o.propertyOrder, name)
	}
	o.property[name] = property{value, mode}
}

func (o *object) deleteProperty(name string) {
	if _, exists := o.property[name]; !exists {
		return
	}

	delete(o.property, name)
	for index, prop := range o.propertyOrder {
		if name == prop {
			if index == len(o.propertyOrder)-1 {
				o.propertyOrder = o.propertyOrder[:index]
			} else {
				o.propertyOrder = append(o.propertyOrder[:index], o.propertyOrder[index+1:]...)
			}
		}
	}
}

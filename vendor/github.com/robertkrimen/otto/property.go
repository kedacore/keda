package otto

// property

type propertyMode int

const (
	modeWriteMask     propertyMode = 0o700
	modeEnumerateMask propertyMode = 0o070
	modeConfigureMask propertyMode = 0o007
	modeOnMask        propertyMode = 0o111
	modeSetMask       propertyMode = 0o222 // If value is 2, then mode is neither "On" nor "Off"
)

type propertyGetSet [2]*object

var nilGetSetObject = object{}

type property struct {
	value interface{}
	mode  propertyMode
}

func (p property) writable() bool {
	return p.mode&modeWriteMask == modeWriteMask&modeOnMask
}

func (p *property) writeOn() {
	p.mode = (p.mode & ^modeWriteMask) | (modeWriteMask & modeOnMask)
}

func (p *property) writeOff() {
	p.mode &= ^modeWriteMask
}

func (p *property) writeClear() {
	p.mode = (p.mode & ^modeWriteMask) | (modeWriteMask & modeSetMask)
}

func (p property) writeSet() bool {
	return p.mode&modeWriteMask&modeSetMask == 0
}

func (p property) enumerable() bool {
	return p.mode&modeEnumerateMask == modeEnumerateMask&modeOnMask
}

func (p *property) enumerateOn() {
	p.mode = (p.mode & ^modeEnumerateMask) | (modeEnumerateMask & modeOnMask)
}

func (p *property) enumerateOff() {
	p.mode &= ^modeEnumerateMask
}

func (p property) enumerateSet() bool {
	return p.mode&modeEnumerateMask&modeSetMask == 0
}

func (p property) configurable() bool {
	return p.mode&modeConfigureMask == modeConfigureMask&modeOnMask
}

func (p *property) configureOn() {
	p.mode = (p.mode & ^modeConfigureMask) | (modeConfigureMask & modeOnMask)
}

func (p *property) configureOff() {
	p.mode &= ^modeConfigureMask
}

func (p property) configureSet() bool { //nolint:unused
	return p.mode&modeConfigureMask&modeSetMask == 0
}

func (p property) copy() *property { //nolint:unused
	cpy := p
	return &cpy
}

func (p property) get(this *object) Value {
	switch value := p.value.(type) {
	case Value:
		return value
	case propertyGetSet:
		if value[0] != nil {
			return value[0].call(toValue(this), nil, false, nativeFrame)
		}
	}
	return Value{}
}

func (p property) isAccessorDescriptor() bool {
	setGet, test := p.value.(propertyGetSet)
	return test && (setGet[0] != nil || setGet[1] != nil)
}

func (p property) isDataDescriptor() bool {
	if p.writeSet() { // Either "On" or "Off"
		return true
	}
	value, valid := p.value.(Value)
	return valid && !value.isEmpty()
}

func (p property) isGenericDescriptor() bool {
	return !(p.isDataDescriptor() || p.isAccessorDescriptor())
}

func (p property) isEmpty() bool {
	return p.mode == 0o222 && p.isGenericDescriptor()
}

// _enumerableValue, _enumerableTrue, _enumerableFalse?
// .enumerableValue() .enumerableExists()

func toPropertyDescriptor(rt *runtime, value Value) property {
	objectDescriptor := value.object()
	if objectDescriptor == nil {
		panic(rt.panicTypeError("toPropertyDescriptor on nil"))
	}

	var descriptor property
	descriptor.mode = modeSetMask // Initially nothing is set
	if objectDescriptor.hasProperty("enumerable") {
		if objectDescriptor.get("enumerable").bool() {
			descriptor.enumerateOn()
		} else {
			descriptor.enumerateOff()
		}
	}

	if objectDescriptor.hasProperty("configurable") {
		if objectDescriptor.get("configurable").bool() {
			descriptor.configureOn()
		} else {
			descriptor.configureOff()
		}
	}

	if objectDescriptor.hasProperty("writable") {
		if objectDescriptor.get("writable").bool() {
			descriptor.writeOn()
		} else {
			descriptor.writeOff()
		}
	}

	var getter, setter *object
	getterSetter := false

	if objectDescriptor.hasProperty("get") {
		val := objectDescriptor.get("get")
		if val.IsDefined() {
			if !val.isCallable() {
				panic(rt.panicTypeError("toPropertyDescriptor get not callable"))
			}
			getter = val.object()
			getterSetter = true
		} else {
			getter = &nilGetSetObject
			getterSetter = true
		}
	}

	if objectDescriptor.hasProperty("set") {
		val := objectDescriptor.get("set")
		if val.IsDefined() {
			if !val.isCallable() {
				panic(rt.panicTypeError("toPropertyDescriptor set not callable"))
			}
			setter = val.object()
			getterSetter = true
		} else {
			setter = &nilGetSetObject
			getterSetter = true
		}
	}

	if getterSetter {
		if descriptor.writeSet() {
			panic(rt.panicTypeError("toPropertyDescriptor descriptor writeSet"))
		}
		descriptor.value = propertyGetSet{getter, setter}
	}

	if objectDescriptor.hasProperty("value") {
		if getterSetter {
			panic(rt.panicTypeError("toPropertyDescriptor value getterSetter"))
		}
		descriptor.value = objectDescriptor.get("value")
	}

	return descriptor
}

func (rt *runtime) fromPropertyDescriptor(descriptor property) *object {
	obj := rt.newObject()
	if descriptor.isDataDescriptor() {
		obj.defineProperty("value", descriptor.value.(Value), 0o111, false)
		obj.defineProperty("writable", boolValue(descriptor.writable()), 0o111, false)
	} else if descriptor.isAccessorDescriptor() {
		getSet := descriptor.value.(propertyGetSet)
		get := Value{}
		if getSet[0] != nil {
			get = objectValue(getSet[0])
		}
		set := Value{}
		if getSet[1] != nil {
			set = objectValue(getSet[1])
		}
		obj.defineProperty("get", get, 0o111, false)
		obj.defineProperty("set", set, 0o111, false)
	}
	obj.defineProperty("enumerable", boolValue(descriptor.enumerable()), 0o111, false)
	obj.defineProperty("configurable", boolValue(descriptor.configurable()), 0o111, false)
	return obj
}

package otto

type referencer interface {
	invalid() bool               // IsUnresolvableReference
	getValue() Value             // getValue
	putValue(value Value) string // PutValue
	delete() bool
}

// PropertyReference

type propertyReference struct {
	base    *object
	runtime *runtime
	name    string
	at      at
	strict  bool
}

func newPropertyReference(rt *runtime, base *object, name string, strict bool, atv at) *propertyReference {
	return &propertyReference{
		runtime: rt,
		name:    name,
		strict:  strict,
		base:    base,
		at:      atv,
	}
}

func (pr *propertyReference) invalid() bool {
	return pr.base == nil
}

func (pr *propertyReference) getValue() Value {
	if pr.base == nil {
		panic(pr.runtime.panicReferenceError("'%s' is not defined", pr.name, pr.at))
	}
	return pr.base.get(pr.name)
}

func (pr *propertyReference) putValue(value Value) string {
	if pr.base == nil {
		return pr.name
	}
	pr.base.put(pr.name, value, pr.strict)
	return ""
}

func (pr *propertyReference) delete() bool {
	if pr.base == nil {
		// TODO Throw an error if strict
		return true
	}
	return pr.base.delete(pr.name, pr.strict)
}

type stashReference struct {
	base   stasher
	name   string
	strict bool
}

func (sr *stashReference) invalid() bool {
	return false // The base (an environment) will never be nil
}

func (sr *stashReference) getValue() Value {
	return sr.base.getBinding(sr.name, sr.strict)
}

func (sr *stashReference) putValue(value Value) string {
	sr.base.setValue(sr.name, value, sr.strict)
	return ""
}

func (sr *stashReference) delete() bool {
	if sr.base == nil {
		// This should never be reached, but just in case
		return false
	}
	return sr.base.deleteBinding(sr.name)
}

// getIdentifierReference.
func getIdentifierReference(rt *runtime, stash stasher, name string, strict bool, atv at) referencer {
	if stash == nil {
		return newPropertyReference(rt, nil, name, strict, atv)
	}
	if stash.hasBinding(name) {
		return stash.newReference(name, strict, atv)
	}
	return getIdentifierReference(rt, stash.outer(), name, strict, atv)
}

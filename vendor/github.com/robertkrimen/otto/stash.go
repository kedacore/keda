package otto

import (
	"fmt"
)

// stasher is implemented by types which can stash data.
type stasher interface {
	hasBinding(name string) bool                            //
	createBinding(name string, deletable bool, value Value) // CreateMutableBinding
	setBinding(name string, value Value, strict bool)       // SetMutableBinding
	getBinding(name string, throw bool) Value               // GetBindingValue
	deleteBinding(name string) bool                         //
	setValue(name string, value Value, throw bool)          // createBinding + setBinding

	outer() stasher
	runtime() *runtime

	newReference(name string, strict bool, atv at) referencer

	clone(cloner *cloner) stasher
}

type objectStash struct {
	rt     *runtime
	outr   stasher
	object *object
}

func (s *objectStash) runtime() *runtime {
	return s.rt
}

func (rt *runtime) newObjectStash(obj *object, outer stasher) *objectStash {
	if obj == nil {
		obj = rt.newBaseObject()
		obj.class = "environment"
	}
	return &objectStash{
		rt:     rt,
		outr:   outer,
		object: obj,
	}
}

func (s *objectStash) clone(c *cloner) stasher {
	out, exists := c.objectStash(s)
	if exists {
		return out
	}
	*out = objectStash{
		c.runtime,
		c.stash(s.outr),
		c.object(s.object),
	}
	return out
}

func (s *objectStash) hasBinding(name string) bool {
	return s.object.hasProperty(name)
}

func (s *objectStash) createBinding(name string, deletable bool, value Value) {
	if s.object.hasProperty(name) {
		panic(hereBeDragons())
	}
	mode := propertyMode(0o111)
	if !deletable {
		mode = propertyMode(0o110)
	}
	// TODO False?
	s.object.defineProperty(name, value, mode, false)
}

func (s *objectStash) setBinding(name string, value Value, strict bool) {
	s.object.put(name, value, strict)
}

func (s *objectStash) setValue(name string, value Value, throw bool) {
	if !s.hasBinding(name) {
		s.createBinding(name, true, value) // Configurable by default
	} else {
		s.setBinding(name, value, throw)
	}
}

func (s *objectStash) getBinding(name string, throw bool) Value {
	if s.object.hasProperty(name) {
		return s.object.get(name)
	}
	if throw { // strict?
		panic(s.rt.panicReferenceError("Not Defined", name))
	}
	return Value{}
}

func (s *objectStash) deleteBinding(name string) bool {
	return s.object.delete(name, false)
}

func (s *objectStash) outer() stasher {
	return s.outr
}

func (s *objectStash) newReference(name string, strict bool, atv at) referencer {
	return newPropertyReference(s.rt, s.object, name, strict, atv)
}

type dclStash struct {
	rt       *runtime
	outr     stasher
	property map[string]dclProperty
}

type dclProperty struct {
	value     Value
	mutable   bool
	deletable bool
	readable  bool
}

func (rt *runtime) newDeclarationStash(outer stasher) *dclStash {
	return &dclStash{
		rt:       rt,
		outr:     outer,
		property: map[string]dclProperty{},
	}
}

func (s *dclStash) clone(c *cloner) stasher {
	out, exists := c.dclStash(s)
	if exists {
		return out
	}
	prop := make(map[string]dclProperty, len(s.property))
	for index, value := range s.property {
		prop[index] = c.dclProperty(value)
	}
	*out = dclStash{
		c.runtime,
		c.stash(s.outr),
		prop,
	}
	return out
}

func (s *dclStash) hasBinding(name string) bool {
	_, exists := s.property[name]
	return exists
}

func (s *dclStash) runtime() *runtime {
	return s.rt
}

func (s *dclStash) createBinding(name string, deletable bool, value Value) {
	if _, exists := s.property[name]; exists {
		panic(fmt.Errorf("createBinding: %s: already exists", name))
	}
	s.property[name] = dclProperty{
		value:     value,
		mutable:   true,
		deletable: deletable,
		readable:  false,
	}
}

func (s *dclStash) setBinding(name string, value Value, strict bool) {
	prop, exists := s.property[name]
	if !exists {
		panic(fmt.Errorf("setBinding: %s: missing", name))
	}
	if prop.mutable {
		prop.value = value
		s.property[name] = prop
	} else {
		s.rt.typeErrorResult(strict)
	}
}

func (s *dclStash) setValue(name string, value Value, throw bool) {
	if !s.hasBinding(name) {
		s.createBinding(name, false, value) // NOT deletable by default
	} else {
		s.setBinding(name, value, throw)
	}
}

// FIXME This is called a __lot__.
func (s *dclStash) getBinding(name string, throw bool) Value {
	prop, exists := s.property[name]
	if !exists {
		panic(fmt.Errorf("getBinding: %s: missing", name))
	}
	if !prop.mutable && !prop.readable {
		if throw { // strict?
			panic(s.rt.panicTypeError("getBinding property %s not mutable and not readable", name))
		}
		return Value{}
	}
	return prop.value
}

func (s *dclStash) deleteBinding(name string) bool {
	prop, exists := s.property[name]
	if !exists {
		return true
	}
	if !prop.deletable {
		return false
	}
	delete(s.property, name)
	return true
}

func (s *dclStash) outer() stasher {
	return s.outr
}

func (s *dclStash) newReference(name string, strict bool, _ at) referencer {
	return &stashReference{
		name: name,
		base: s,
	}
}

// ========
// _fnStash
// ========

type fnStash struct {
	dclStash
	arguments           *object
	indexOfArgumentName map[string]string
}

func (rt *runtime) newFunctionStash(outer stasher) *fnStash {
	return &fnStash{
		dclStash: dclStash{
			rt:       rt,
			outr:     outer,
			property: map[string]dclProperty{},
		},
	}
}

func (s *fnStash) clone(c *cloner) stasher {
	out, exists := c.fnStash(s)
	if exists {
		return out
	}
	dclStash := s.dclStash.clone(c).(*dclStash)
	index := make(map[string]string, len(s.indexOfArgumentName))
	for name, value := range s.indexOfArgumentName {
		index[name] = value
	}
	*out = fnStash{
		dclStash:            *dclStash,
		arguments:           c.object(s.arguments),
		indexOfArgumentName: index,
	}
	return out
}

// getStashProperties returns the properties from stash.
func getStashProperties(stash stasher) []string {
	switch vars := stash.(type) {
	case *dclStash:
		keys := make([]string, 0, len(vars.property))
		for k := range vars.property {
			keys = append(keys, k)
		}
		return keys
	case *fnStash:
		keys := make([]string, 0, len(vars.property))
		for k := range vars.property {
			keys = append(keys, k)
		}
		return keys
	case *objectStash:
		keys := make([]string, 0, len(vars.object.property))
		for k := range vars.object.property {
			keys = append(keys, k)
		}
		return keys
	default:
		panic("unknown stash type")
	}
}

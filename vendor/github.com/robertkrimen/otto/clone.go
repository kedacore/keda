package otto

import (
	"fmt"
)

type cloner struct {
	runtime     *runtime
	obj         map[*object]*object
	objectstash map[*objectStash]*objectStash
	dclstash    map[*dclStash]*dclStash
	fnstash     map[*fnStash]*fnStash
}

func (rt *runtime) clone() *runtime {
	rt.lck.Lock()
	defer rt.lck.Unlock()

	out := &runtime{
		debugger:   rt.debugger,
		random:     rt.random,
		stackLimit: rt.stackLimit,
		traceLimit: rt.traceLimit,
	}

	c := cloner{
		runtime:     out,
		obj:         make(map[*object]*object),
		objectstash: make(map[*objectStash]*objectStash),
		dclstash:    make(map[*dclStash]*dclStash),
		fnstash:     make(map[*fnStash]*fnStash),
	}

	globalObject := c.object(rt.globalObject)
	out.globalStash = out.newObjectStash(globalObject, nil)
	out.globalObject = globalObject
	out.global = global{
		c.object(rt.global.Object),
		c.object(rt.global.Function),
		c.object(rt.global.Array),
		c.object(rt.global.String),
		c.object(rt.global.Boolean),
		c.object(rt.global.Number),
		c.object(rt.global.Math),
		c.object(rt.global.Date),
		c.object(rt.global.RegExp),
		c.object(rt.global.Error),
		c.object(rt.global.EvalError),
		c.object(rt.global.TypeError),
		c.object(rt.global.RangeError),
		c.object(rt.global.ReferenceError),
		c.object(rt.global.SyntaxError),
		c.object(rt.global.URIError),
		c.object(rt.global.JSON),

		c.object(rt.global.ObjectPrototype),
		c.object(rt.global.FunctionPrototype),
		c.object(rt.global.ArrayPrototype),
		c.object(rt.global.StringPrototype),
		c.object(rt.global.BooleanPrototype),
		c.object(rt.global.NumberPrototype),
		c.object(rt.global.DatePrototype),
		c.object(rt.global.RegExpPrototype),
		c.object(rt.global.ErrorPrototype),
		c.object(rt.global.EvalErrorPrototype),
		c.object(rt.global.TypeErrorPrototype),
		c.object(rt.global.RangeErrorPrototype),
		c.object(rt.global.ReferenceErrorPrototype),
		c.object(rt.global.SyntaxErrorPrototype),
		c.object(rt.global.URIErrorPrototype),
	}

	out.eval = out.globalObject.property["eval"].value.(Value).value.(*object)
	out.globalObject.prototype = out.global.ObjectPrototype

	// Not sure if this is necessary, but give some help to the GC
	c.runtime = nil
	c.obj = nil
	c.objectstash = nil
	c.dclstash = nil
	c.fnstash = nil

	return out
}

func (c *cloner) object(in *object) *object {
	if out, exists := c.obj[in]; exists {
		return out
	}
	out := &object{}
	c.obj[in] = out
	return in.objectClass.clone(in, out, c)
}

func (c *cloner) dclStash(in *dclStash) (*dclStash, bool) {
	if out, exists := c.dclstash[in]; exists {
		return out, true
	}
	out := &dclStash{}
	c.dclstash[in] = out
	return out, false
}

func (c *cloner) objectStash(in *objectStash) (*objectStash, bool) {
	if out, exists := c.objectstash[in]; exists {
		return out, true
	}
	out := &objectStash{}
	c.objectstash[in] = out
	return out, false
}

func (c *cloner) fnStash(in *fnStash) (*fnStash, bool) {
	if out, exists := c.fnstash[in]; exists {
		return out, true
	}
	out := &fnStash{}
	c.fnstash[in] = out
	return out, false
}

func (c *cloner) value(in Value) Value {
	out := in
	if value, ok := in.value.(*object); ok {
		out.value = c.object(value)
	}
	return out
}

func (c *cloner) valueArray(in []Value) []Value {
	out := make([]Value, len(in))
	for index, value := range in {
		out[index] = c.value(value)
	}
	return out
}

func (c *cloner) stash(in stasher) stasher {
	if in == nil {
		return nil
	}
	return in.clone(c)
}

func (c *cloner) property(in property) property {
	out := in

	switch value := in.value.(type) {
	case Value:
		out.value = c.value(value)
	case propertyGetSet:
		p := propertyGetSet{}
		if value[0] != nil {
			p[0] = c.object(value[0])
		}
		if value[1] != nil {
			p[1] = c.object(value[1])
		}
		out.value = p
	default:
		panic(fmt.Errorf("in.value.(Value) != true; in.value is %T", in.value))
	}

	return out
}

func (c *cloner) dclProperty(in dclProperty) dclProperty {
	out := in
	out.value = c.value(in.value)
	return out
}

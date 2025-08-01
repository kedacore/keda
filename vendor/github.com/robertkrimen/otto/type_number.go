package otto

func (rt *runtime) newNumberObject(value Value) *object {
	return rt.newPrimitiveObject(classNumberName, value.numberValue())
}

package otto

func (rt *runtime) newBooleanObject(value Value) *object {
	return rt.newPrimitiveObject(classBooleanName, boolValue(value.bool()))
}

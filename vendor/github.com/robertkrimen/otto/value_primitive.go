package otto

func toNumberPrimitive(value Value) Value {
	return toPrimitive(value, defaultValueHintNumber)
}

func toPrimitiveValue(value Value) Value {
	return toPrimitive(value, defaultValueNoHint)
}

func toPrimitive(value Value, hint defaultValueHint) Value {
	switch value.kind {
	case valueNull, valueUndefined, valueNumber, valueString, valueBoolean:
		return value
	case valueObject:
		return value.object().DefaultValue(hint)
	default:
		panic(hereBeDragons(value.kind, value))
	}
}

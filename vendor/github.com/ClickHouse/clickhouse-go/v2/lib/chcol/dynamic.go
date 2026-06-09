package chcol

type Dynamic = Variant

// NewDynamic creates a new Dynamic with the given value
func NewDynamic(v any) Dynamic {
	return Dynamic{value: v}
}

// NewDynamicWithType creates a new Dynamic with the given value and ClickHouse type
func NewDynamicWithType(v any, chType string) Dynamic {
	return Dynamic{
		value:  v,
		chType: chType,
	}
}

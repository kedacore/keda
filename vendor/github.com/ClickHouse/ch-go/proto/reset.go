package proto

type Resettable interface {
	Reset()
}

// Reset is helper to reset columns.
func Reset(columns ...Resettable) {
	for _, column := range columns {
		column.Reset()
	}
}

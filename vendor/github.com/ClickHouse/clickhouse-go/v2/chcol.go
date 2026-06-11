package clickhouse

import "github.com/ClickHouse/clickhouse-go/v2/lib/chcol"

// Re-export chcol types/funcs to top level clickhouse package

type (
	// Variant represents a ClickHouse Variant type that can hold multiple possible types
	Variant = chcol.Variant
	// Dynamic is an alias for the Variant type
	Dynamic = chcol.Dynamic
	// JSON represents a ClickHouse JSON type that can hold multiple possible types
	JSON = chcol.JSON

	// JSONSerializer interface allows a struct to be manually converted to an optimized JSON structure instead of relying
	// on recursive reflection.
	// Note that the struct must be a pointer in order for the interface to be matched, reflection will be used otherwise.
	JSONSerializer = chcol.JSONSerializer
	// JSONDeserializer interface allows a struct to load its data from an optimized JSON structure instead of relying
	// on recursive reflection to set its fields.
	JSONDeserializer = chcol.JSONDeserializer
)

// NewVariant creates a new Variant with the given value
func NewVariant(v any) Variant {
	return chcol.NewVariant(v)
}

// NewVariantWithType creates a new Variant with the given value and ClickHouse type
func NewVariantWithType(v any, chType string) Variant {
	return chcol.NewVariantWithType(v, chType)
}

// NewDynamic creates a new Dynamic with the given value
func NewDynamic(v any) Dynamic {
	return chcol.NewDynamic(v)
}

// NewDynamicWithType creates a new Dynamic with the given value and ClickHouse type
func NewDynamicWithType(v any, chType string) Dynamic {
	return chcol.NewDynamicWithType(v, chType)
}

// NewJSON creates a new empty JSON value
func NewJSON() *JSON {
	return chcol.NewJSON()
}

// ExtractJSONPathAs is a convenience function for asserting a path to a specific type.
// The underlying value is also extracted from its Dynamic wrapper if present.
// T cannot be a Dynamic, if you want a Dynamic simply use ExtractJSONPathAsDynamic.
func ExtractJSONPathAs[T any](o *JSON, path string) (valueAs T, ok bool) {
	return chcol.ExtractJSONPathAs[T](o, path)
}

// ExtractJSONPathAsDynamic is a convenience function for asserting a path to a Dynamic.
// If the value is not a Dynamic, the value is wrapped in an untyped Dynamic with false returned.
func ExtractJSONPathAsDynamic(o *JSON, path string) (Dynamic, bool) {
	return chcol.ExtractJSONPathAsDynamic(o, path)
}

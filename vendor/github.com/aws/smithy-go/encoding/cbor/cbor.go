// Package cbor implements partial encoding/decoding of concise binary object
// representation (CBOR) described in [RFC 8949].
//
// This package is intended for use only by the smithy client runtime. The
// exported API therein is not considered stable and is subject to breaking
// changes without notice. More specifically, this package implements a subset
// of the RFC 8949 specification required to support the Smithy RPCv2-CBOR
// protocol and is NOT suitable for general application use.
//
// The following principal restrictions apply:
//   - Map (major type 5) keys can only be strings.
//   - Float16 (major type 7, 25) values can be read but not encoded. Any
//     float16 encountered during decode is converted to float32.
//   - Indefinite-length values can be read but not encoded. Since the encoding
//     API operates strictly off of a constructed syntax tree, the length of each
//     data item in a Value will always be known and the encoder will always
//     generate definite-length variants.
//
// It is the responsibility of the caller to determine whether a decoded CBOR
// integral or floating-point Value is suitable for its target (e.g. whether
// the value of a CBOR Uint fits into a field modeled as a Smithy short).
//
// All CBOR tags (major type 6) are implicitly supported since the
// encoder/decoder does not attempt to interpret a tag's contents. It is the
// responsibility of the caller to both provide valid Tag values to encode and
// to assert that a decoded Tag's contents are valid for its tag ID (e.g.
// ensuring whether a Tag with ID 1, indicating an enclosed epoch timestamp,
// actually contains a valid integral or floating-point CBOR Value).
//
// [RFC 8949]: https://www.rfc-editor.org/rfc/rfc8949.html
package cbor

// Value describes a CBOR data item.
//
// The following types implement Value:
//   - [Uint]
//   - [NegInt]
//   - [Slice]
//   - [String]
//   - [List]
//   - [Map]
//   - [Tag]
//   - [Bool]
//   - [Nil]
//   - [Undefined]
//   - [Float32]
//   - [Float64]
type Value interface {
	len() int
	encode(p []byte) int
}

var (
	_ Value = Uint(0)
	_ Value = NegInt(0)
	_ Value = Slice(nil)
	_ Value = String("")
	_ Value = List(nil)
	_ Value = Map(nil)
	_ Value = (*Tag)(nil)
	_ Value = Bool(false)
	_ Value = (*Nil)(nil)
	_ Value = (*Undefined)(nil)
	_ Value = Float32(0)
	_ Value = Float64(0)
)

// Uint describes a CBOR uint (major type 0) in the range [0, 2^64-1].
type Uint uint64

// NegInt describes a CBOR negative int (major type 1) in the range [-2^64, -1].
//
// The "true negative" value of a type 1 is specified by RFC 8949 to be -1
// minus the encoded value. The encoder/decoder applies this bias
// automatically, e.g. the integral -100 is represented as NegInt(100), which
// will which encode to/from hex 3863 (major 1, minor 24, argument 99).
//
// This implicitly means that the lower bound of this type -2^64 is represented
// as the wraparound value NegInt(0). Deserializer implementations should take
// care to guard against this case when deriving a value for a signed integral
// type which was encoded as NegInt.
type NegInt uint64

// Slice describes a CBOR byte slice (major type 2).
type Slice []byte

// String describes a CBOR text string (major type 3).
type String string

// List describes a CBOR list (major type 4).
type List []Value

// Map describes a CBOR map (major type 5).
//
// The type signature of the map's key is restricted to string as it is in
// Smithy.
type Map map[string]Value

// Tag describes a CBOR-tagged value (major type 6).
type Tag struct {
	ID    uint64
	Value Value
}

// Bool describes a boolean value (major type 7, argument 20/21).
type Bool bool

// Nil is the `nil` / `null` literal (major type 7, argument 22).
type Nil struct{}

// Undefined is the `undefined` literal (major type 7, argument 23).
type Undefined struct{}

// Float32 describes an IEEE 754 single-precision floating-point number
// (major type 7, argument 26).
//
// Go does not natively support float16, all values encoded as such (major type
// 7, argument 25) must be represented by this variant instead.
type Float32 float32

// Float64 describes an IEEE 754 double-precision floating-point number
// (major type 7, argument 27).
type Float64 float64

// Encode returns a byte slice that encodes the given Value.
func Encode(v Value) []byte {
	p := make([]byte, v.len())
	v.encode(p)
	return p
}

// Decode returns the Value encoded in the given byte slice.
func Decode(p []byte) (Value, error) {
	v, _, err := decode(p)
	if err != nil {
		return nil, err
	}
	return v, nil
}

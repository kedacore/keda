package cbor

// major type in LSB position
type majorType byte

const (
	majorTypeUint majorType = iota
	majorTypeNegInt
	majorTypeSlice
	majorTypeString
	majorTypeList
	majorTypeMap
	majorTypeTag
	majorType7
)

// masks for major/minor component in encoded head
const (
	maskMajor = 0b111 << 5
	maskMinor = 0b11111
)

// minor value encodings to represent arg bit length (and indefinite)
const (
	minorArg1       = 24
	minorArg2       = 25
	minorArg4       = 26
	minorArg8       = 27
	minorIndefinite = 31
)

// minor sentinels for everything in major 7
const (
	major7False     = 20
	major7True      = 21
	major7Nil       = 22
	major7Undefined = 23
	major7Float16   = minorArg2
	major7Float32   = minorArg4
	major7Float64   = minorArg8
)

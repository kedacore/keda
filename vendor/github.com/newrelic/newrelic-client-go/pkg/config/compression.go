package config

// CompressionType to use during transport.
type CompressionType string

// Compression is an Enum of compression types available
var Compression = struct {
	None CompressionType
	Gzip CompressionType
}{
	None: "",
	Gzip: "gzip",
}

// String returns the name of the compression type
func (c CompressionType) String() string {
	name := string(c)

	if name == "" {
		return "none"
	}

	return name
}

// ParseCompression takes a named compression and returns the Type
func ParseCompression(name string) CompressionType {
	switch name {
	case "gzip":
		return Compression.Gzip
	default:
		return Compression.None
	}
}

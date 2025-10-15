package otto

//go:generate go run ./tools/gen-jscore -output inline.go
//go:generate stringer -type=valueKind -trimprefix=value -output=value_kind.gen.go

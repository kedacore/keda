package otto

import (
	"bytes"
	"encoding/gob"
	"errors"
)

// ErrVersion is an error which represents a version mismatch.
var ErrVersion = errors.New("version mismatch")

var scriptVersion = "2014-04-13/1"

// Script is a handle for some (reusable) JavaScript.
// Passing a Script value to a run method will evaluate the JavaScript.
type Script struct {
	version  string
	program  *nodeProgram
	filename string
	src      string
}

// Compile will parse the given source and return a Script value or nil and
// an error if there was a problem during compilation.
//
//	script, err := vm.Compile("", `var abc; if (!abc) abc = 0; abc += 2; abc;`)
//	vm.Run(script)
func (o *Otto) Compile(filename string, src interface{}) (*Script, error) {
	return o.CompileWithSourceMap(filename, src, nil)
}

// CompileWithSourceMap does the same thing as Compile, but with the obvious
// difference of applying a source map.
func (o *Otto) CompileWithSourceMap(filename string, src, sm interface{}) (*Script, error) {
	program, err := o.runtime.parse(filename, src, sm)
	if err != nil {
		return nil, err
	}

	node := cmplParse(program)
	script := &Script{
		version:  scriptVersion,
		program:  node,
		filename: filename,
		src:      program.File.Source(),
	}

	return script, nil
}

func (s *Script) String() string {
	return "// " + s.filename + "\n" + s.src
}

// MarshalBinary will marshal a script into a binary form. A marshalled script
// that is later unmarshalled can be executed on the same version of the otto runtime.
//
// The binary format can change at any time and should be considered unspecified and opaque.
func (s *Script) marshalBinary() ([]byte, error) {
	var bfr bytes.Buffer
	encoder := gob.NewEncoder(&bfr)
	err := encoder.Encode(s.version)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(s.program)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(s.filename)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(s.src)
	if err != nil {
		return nil, err
	}
	return bfr.Bytes(), nil
}

// UnmarshalBinary will vivify a marshalled script into something usable. If the script was
// originally marshalled on a different version of the otto runtime, then this method
// will return an error.
//
// The binary format can change at any time and should be considered unspecified and opaque.
func (s *Script) unmarshalBinary(data []byte) (err error) { //nolint:nonamedreturns
	decoder := gob.NewDecoder(bytes.NewReader(data))
	defer func() {
		if err != nil {
			s.version = ""
			s.program = nil
			s.filename = ""
			s.src = ""
		}
	}()
	if err = decoder.Decode(&s.version); err != nil {
		return err
	}
	if s.version != scriptVersion {
		return ErrVersion
	}
	if err = decoder.Decode(&s.program); err != nil {
		return err
	}
	if err = decoder.Decode(&s.filename); err != nil {
		return err
	}
	return decoder.Decode(&s.src)
}

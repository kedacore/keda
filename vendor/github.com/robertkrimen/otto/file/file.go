// Package file encapsulates the file abstractions used by the ast & parser.
package file

import (
	"fmt"
	"strings"

	"gopkg.in/sourcemap.v1"
)

// Idx is a compact encoding of a source position within a file set.
// It can be converted into a Position for a more convenient, but much
// larger, representation.
type Idx int

// Position describes an arbitrary source position
// including the filename, line, and column location.
type Position struct {
	Filename string // The filename where the error occurred, if any
	Offset   int    // The src offset
	Line     int    // The line number, starting at 1
	Column   int    // The column number, starting at 1 (The character count)
}

// A Position is valid if the line number is > 0.

func (p *Position) isValid() bool {
	return p.Line > 0
}

// String returns a string in one of several forms:
//
//	file:line:column    A valid position with filename
//	line:column         A valid position without filename
//	file                An invalid position with filename
//	-                   An invalid position without filename
func (p *Position) String() string {
	str := p.Filename
	if p.isValid() {
		if str != "" {
			str += ":"
		}
		str += fmt.Sprintf("%d:%d", p.Line, p.Column)
	}
	if str == "" {
		str = "-"
	}
	return str
}

// A FileSet represents a set of source files.
type FileSet struct {
	last  *File
	files []*File
}

// AddFile adds a new file with the given filename and src.
//
// This an internal method, but exported for cross-package use.
func (fs *FileSet) AddFile(filename, src string) int {
	base := fs.nextBase()
	file := &File{
		name: filename,
		src:  src,
		base: base,
	}
	fs.files = append(fs.files, file)
	fs.last = file
	return base
}

func (fs *FileSet) nextBase() int {
	if fs.last == nil {
		return 1
	}
	return fs.last.base + len(fs.last.src) + 1
}

// File returns the File at idx or nil if not found.
func (fs *FileSet) File(idx Idx) *File {
	for _, file := range fs.files {
		if idx <= Idx(file.base+len(file.src)) {
			return file
		}
	}
	return nil
}

// Position converts an Idx in the FileSet into a Position.
func (fs *FileSet) Position(idx Idx) *Position {
	for _, file := range fs.files {
		if idx <= Idx(file.base+len(file.src)) {
			return file.Position(idx - Idx(file.base))
		}
	}

	return nil
}

// File represents a file to parse.
type File struct {
	sm   *sourcemap.Consumer
	name string
	src  string
	base int
}

// NewFile returns a new file with the given filename, src and base.
func NewFile(filename, src string, base int) *File {
	return &File{
		name: filename,
		src:  src,
		base: base,
	}
}

// WithSourceMap sets the source map of fl.
func (fl *File) WithSourceMap(sm *sourcemap.Consumer) *File {
	fl.sm = sm
	return fl
}

// Name returns the name of fl.
func (fl *File) Name() string {
	return fl.name
}

// Source returns the source of fl.
func (fl *File) Source() string {
	return fl.src
}

// Base returns the base of fl.
func (fl *File) Base() int {
	return fl.base
}

// Position returns the position at idx or nil if not valid.
func (fl *File) Position(idx Idx) *Position {
	position := &Position{}

	offset := int(idx) - fl.base

	if offset >= len(fl.src) || offset < 0 {
		return nil
	}

	src := fl.src[:offset]

	position.Filename = fl.name
	position.Offset = offset
	position.Line = strings.Count(src, "\n") + 1

	if index := strings.LastIndex(src, "\n"); index >= 0 {
		position.Column = offset - index
	} else {
		position.Column = len(src) + 1
	}

	if fl.sm != nil {
		if f, _, l, c, ok := fl.sm.Source(position.Line, position.Column); ok {
			position.Filename, position.Line, position.Column = f, l, c
		}
	}

	return position
}

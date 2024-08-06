package file

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Error struct {
	Location
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Snippet string `json:"snippet"`
	Prev    error  `json:"prev"`
}

func (e *Error) Error() string {
	return e.format()
}

func (e *Error) Bind(source Source) *Error {
	e.Line = 1
	for i, r := range source {
		if i == e.From {
			break
		}
		if r == '\n' {
			e.Line++
			e.Column = 0
		} else {
			e.Column++
		}
	}
	if snippet, found := source.Snippet(e.Line); found {
		snippet := strings.Replace(snippet, "\t", " ", -1)
		srcLine := "\n | " + snippet
		var bytes = []byte(snippet)
		var indLine = "\n | "
		for i := 0; i < e.Column && len(bytes) > 0; i++ {
			_, sz := utf8.DecodeRune(bytes)
			bytes = bytes[sz:]
			if sz > 1 {
				goto noind
			} else {
				indLine += "."
			}
		}
		if _, sz := utf8.DecodeRune(bytes); sz > 1 {
			goto noind
		} else {
			indLine += "^"
		}
		srcLine += indLine

	noind:
		e.Snippet = srcLine
	}
	return e
}

func (e *Error) Unwrap() error {
	return e.Prev
}

func (e *Error) Wrap(err error) {
	e.Prev = err
}

func (e *Error) format() string {
	if e.Snippet == "" {
		return e.Message
	}
	return fmt.Sprintf(
		"%s (%d:%d)%s",
		e.Message,
		e.Line,
		e.Column+1, // add one to the 0-based column for display
		e.Snippet,
	)
}

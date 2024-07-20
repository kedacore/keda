package gotest

import (
	"io"

	"github.com/jstemmer/go-junit-report/v2/gtr"
	"github.com/jstemmer/go-junit-report/v2/parser/gotest/internal/reader"
)

// NewJSONParser returns a new Go test json output parser.
func NewJSONParser(options ...Option) *JSONParser {
	return &JSONParser{gp: NewParser(options...)}
}

// JSONParser is a `go test -json` output Parser.
type JSONParser struct {
	gp *Parser
}

// Parse parses Go test json output from the given io.Reader r and returns
// gtr.Report.
func (p *JSONParser) Parse(r io.Reader) (gtr.Report, error) {
	return p.gp.parse(reader.NewJSONEventReader(r))
}

// Events returns the events created by the parser.
func (p *JSONParser) Events() []Event {
	return p.gp.Events()
}

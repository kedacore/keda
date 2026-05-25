package v2

import (
	"bytes"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
)

// peekFrameType reads the line directly, so it can be used to determine the frame type without parsing the entire frame.
// We do it by parsing the first json property, which is the frame type.
// We look for a : and then the value between two quotes.
func peekFrameType(line []byte) (FrameType, error) {
	colon := bytes.IndexByte(line, ':')

	if colon == -1 {
		return "", errors.ES(errors.OpUnknown, errors.KInternal, "Missing colon in frame")
	}

	firstQuote := bytes.IndexByte(line[colon+1:], '"')
	if firstQuote == -1 {
		return "", errors.ES(errors.OpUnknown, errors.KInternal, "Missing quote in frame")
	}
	secondQuote := bytes.IndexByte(line[colon+1+firstQuote+1:], '"')
	if secondQuote == -1 {
		return "", errors.ES(errors.OpUnknown, errors.KInternal, "Missing quote in frame")
	}

	return FrameType(line[colon+1+firstQuote+1 : colon+1+firstQuote+1+secondQuote]), nil
}

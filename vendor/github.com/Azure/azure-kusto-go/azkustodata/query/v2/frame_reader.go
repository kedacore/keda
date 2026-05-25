package v2

import (
	"bufio"
	"context"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"io"
)

// frameReader reads frame-by-frame from a Kusto response.
// It is specifically designed for the FragmentedV2 protocol:
// 1. The response is a JSON array of frames, but separated by newlines.
// 2. We convert it into a proper JsonLines format by stripping the first byte of each line.
// 3. We then read each line as a separate frame.
// 4. When we reach the end of the array, it means we have reached the end of the response, and we return io.EOF.
// 5. For every line we read, we check if the context has been cancelled, and if so, return the error.
type frameReader struct {
	orig   io.ReadCloser
	reader bufio.Reader
	ctx    context.Context
}

func newFrameReader(r io.ReadCloser, ctx context.Context) (*frameReader, error) {
	br := bufio.NewReader(r)

	err := validateJsonResponse(br)
	if err != nil {
		return nil, err
	}

	return &frameReader{orig: r, reader: *br, ctx: ctx}, nil
}

// validateJsonResponse reads the first byte of the response to determine if it is in fact valid JSON.
// Kusto may return an error message that is not JSON, and instead will just be a plain string with an error message.
// If the first byte is not '[', then we assume it is an error message and return an error.
func validateJsonResponse(br *bufio.Reader) error {
	first, err := br.Peek(1)
	if err != nil {
		return err
	}
	if len(first) == 0 {
		return errors.ES(errors.OpUnknown, errors.KInternal, "No data")
	}

	if first[0] != '[' {
		all, err := io.ReadAll(br)
		if err != nil {
			return err
		}
		return errors.ES(errors.OpUnknown, errors.KInternal, "Got error: %v", string(all))
	}
	return nil
}

// advance reads the next frame from the response.
func (fr *frameReader) advance() ([]byte, error) {
	// Check if the context has been cancelled, so we won't keep reading after the response is cancelled.
	if fr.ctx.Err() != nil {
		return nil, fr.ctx.Err()
	}

	// Read until the end of the current line, which is the entire frame.
	line, err := fr.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// If the first character is ']', then we have reached the end of the response.
	if len(line) > 0 && line[0] == ']' {
		return nil, io.EOF
	}

	// Trim newline
	line = line[:len(line)-1]

	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	if len(line) < 2 {
		return nil, errors.ES(errors.OpUnknown, errors.KInternal, "Got EOF while reading frame")
	}

	// We skip the first byte of the line, as it is a comma, or the start of the array.
	if line[0] != '[' && line[0] != ',' {
		return nil, errors.ES(errors.OpUnknown, errors.KInternal, "Expected comma or start array, got '%c'", line[0])
	}

	line = line[1:]

	return line, nil
}

// Close closes the underlying reader.
func (fr *frameReader) close() error {
	return fr.orig.Close()
}

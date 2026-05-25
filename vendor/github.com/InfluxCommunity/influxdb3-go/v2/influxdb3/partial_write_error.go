/*
 The MIT License

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package influxdb3

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	msgPartialWriteOccurred = "partial write of line protocol occurred" // v3 endpoint with accept_partial=true error
	msgParsingFailedLp      = "parsing failed for write_lp endpoint"    // v3 endpoint with accept_partial=false
)

// PartialWriteLineError describes a single line-level write failure returned by /api/v3/write_lp.
type PartialWriteLineError struct {
	// ErrorMessage describes why the line failed.
	ErrorMessage string `json:"error_message"`
	// LineNumber is a 1-based line index in the submitted payload.
	LineNumber int `json:"line_number"`
	// OriginalLine is the line content reported by server.
	OriginalLine string `json:"original_line"`
}

// PartialWriteError represents a /api/v3/write_lp error that carries per-line failure details.
type PartialWriteError struct {
	ServerError
	LineErrors []PartialWriteLineError
}

// Unwrap allows errors.As(err, &serverErr) where serverErr is *ServerError
// when the original error is a *PartialWriteError.
func (e *PartialWriteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return &e.ServerError
}

func isPartialWriteMessage(message string) bool {
	return strings.Contains(message, msgPartialWriteOccurred) || strings.Contains(message, msgParsingFailedLp)
}

func parsePartialWriteLineErrorInfo(raw json.RawMessage) ([]PartialWriteLineError, []string) {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil, nil
	}

	if lineErrors, ok := parsePartialWriteDataArray(raw); ok {
		return lineErrors, formatPartialWriteLineErrorDetails(lineErrors)
	}

	if details, ok := parsePartialWriteRawArrayDetails(raw); ok {
		return nil, details
	}

	lineError, ok := parsePartialWriteLineError(raw)
	if ok {
		lineErrors := []PartialWriteLineError{lineError}
		return lineErrors, formatPartialWriteLineErrorDetails(lineErrors)
	}

	return nil, nil
}

type partialWriteDataItem struct {
	ErrorMessage string `json:"error_message"`
	LineNumber   int    `json:"line_number"`
	OriginalLine string `json:"original_line"`
}

func parsePartialWriteDataArray(raw json.RawMessage) ([]PartialWriteLineError, bool) {
	var items []*partialWriteDataItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}

	lineErrors := make([]PartialWriteLineError, 0, len(items))
	for _, item := range items {
		if item == nil || item.ErrorMessage == "" {
			continue
		}
		lineErrors = append(lineErrors, PartialWriteLineError{
			ErrorMessage: item.ErrorMessage,
			LineNumber:   item.LineNumber,
			OriginalLine: item.OriginalLine,
		})
	}
	return lineErrors, len(lineErrors) > 0
}

func formatPartialWriteLineErrorDetails(lineErrors []PartialWriteLineError) []string {
	details := make([]string, 0, len(lineErrors))
	for _, lineError := range lineErrors {
		if lineError.LineNumber != 0 && lineError.OriginalLine != "" {
			details = append(details, fmt.Sprintf(
				"line %d: %s (%s)",
				lineError.LineNumber,
				lineError.ErrorMessage,
				lineError.OriginalLine,
			))
		} else if lineError.ErrorMessage != "" {
			details = append(details, lineError.ErrorMessage)
		}
	}
	return details
}

func parsePartialWriteRawArrayDetails(raw json.RawMessage) ([]string, bool) {
	var rawItems []json.RawMessage
	if err := json.Unmarshal(raw, &rawItems); err != nil {
		return nil, false
	}

	details := make([]string, 0, len(rawItems))
	for _, rawItem := range rawItems {
		s := strings.TrimSpace(string(rawItem))
		if s != "" && !strings.EqualFold(s, "null") {
			details = append(details, s)
		}
	}
	return details, true
}

func parsePartialWriteLineError(raw json.RawMessage) (PartialWriteLineError, bool) {
	var lineError PartialWriteLineError
	if err := json.Unmarshal(raw, &lineError); err != nil {
		return PartialWriteLineError{}, false
	}

	if lineError.LineNumber == 0 && lineError.ErrorMessage == "" && lineError.OriginalLine == "" {
		return PartialWriteLineError{}, false
	}

	return lineError, true
}

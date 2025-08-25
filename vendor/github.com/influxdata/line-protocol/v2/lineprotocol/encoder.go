package lineprotocol

import (
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

// Encoder encapsulates the encoding part of the line protocol.
//
// The zero value of an Encoder is ready to use.
//
// It is associated with a []byte buffer which is appended to
// each time a method is called.
//
// Methods must be called in the same order that their
// respective data appears in the line-protocol point (Encoder
// doesn't reorder anything). That is, for a given entry, methods
// must be called in the following order:
//
//	StartLine
//	AddTag (zero or more times)
//	AddField (one or more times)
//	EndLine (optional)
//
// When an error is encountered encoding a point,
// the Err method returns it, and the erroneous point
// is omitted from the result.
//
type Encoder struct {
	buf        []byte
	prevTagKey []byte
	// lineStart holds the index of the start of the current line.
	lineStart int
	// section holds the section of line that's about to be added.
	section section
	// lax holds whether keys and values are checked for validity
	// when being encoded.
	lax bool
	// lineHasError records whether there's been an error encountered
	// on the current entry, in which case, no further data will be added
	// until the next entry.
	lineHasError bool
	// err holds the most recent error encountered when encoding.
	err error
	// pointIndex holds the index of the current point being encoded.
	pointIndex int
	// precisionMultiplier holds the timestamp precision.
	// Timestamps are divided by this when encoded.
	precisionMultiplier int64
}

// Bytes returns the current line buffer.
func (e *Encoder) Bytes() []byte {
	return e.buf
}

// SetBuffer sets the buffer used for the line,
// clears any current error and resets the line.
//
// Encoded data will be appended to buf.
func (e *Encoder) SetBuffer(buf []byte) {
	e.buf = buf
	e.pointIndex = 0
	e.ClearErr()
	e.section = measurementSection
}

// SetPrecision sets the precision used to encode the time stamps
// in the encoded messages. The default precision is Nanosecond.
// Timestamps are truncated to this precision.
func (e *Encoder) SetPrecision(p Precision) {
	e.precisionMultiplier = int64(p.Duration())
}

// Reset resets the line, clears any error, and starts writing at the start
// of the line buffer slice.
func (e *Encoder) Reset() {
	e.SetBuffer(e.buf[:0])
}

// SetLax sets whether the Encoder methods check fully for validity or not.
// When Lax is true:
//
// - measurement names, tag and field keys aren't checked for invalid characters
// - field values passed to AddRawField are not bounds or syntax checked
// - tag keys are not checked to be in alphabetical order.
//
// This can be used to increase performance in
// places where values are already known to be valid.
func (e *Encoder) SetLax(lax bool) {
	e.lax = lax
}

// Err returns the first encoding error that's been encountered so far,
// if any.
// TODO define a type so that we can get access to the line where it happened.
func (e *Encoder) Err() error {
	return e.err
}

// ClearErr clears any current encoding error.
func (e *Encoder) ClearErr() {
	e.err = nil
}

// StartLine starts writing a line with the given measurement name. If this
// is called when it's not possible to start a new entry, or the
// measurement cannot be encoded, it will return an error.
//
// Starting a new entry is always allowed when there's been an error
// encoding the previous entry.
func (e *Encoder) StartLine(measurement string) {
	section := e.section
	e.pointIndex++
	e.section = tagSection
	if section == tagSection {
		// This error is unusual, because it indicates an error on the previous
		// line, even though there's probably not an error on this line, so
		// don't return here. This means that unfortunately, if you
		// add a line with an invalid measurement immediately after
		// adding a line with no fields, you won't ever see the second
		// of those two errors. Clients can avoid that possibility by making
		// sure to call EndLine even if they don't wish to add a timestamp.
		e.setErrorf("cannot start line without adding at least one field to previous line")
	}
	e.prevTagKey = e.prevTagKey[:0]
	e.lineStart = len(e.buf)
	e.lineHasError = false
	if !e.lax {
		if !validMeasurementOrKey(measurement) {
			e.setErrorf("invalid measurement %q", measurement)
			return
		}
	}
	if section != measurementSection && section != endSection {
		// This isn't the first line, and EndLine hasn't been explicitly called,
		// so we need a newline separator.
		e.buf = append(e.buf, '\n')
	}
	e.buf = measurementEscapes.appendEscaped(e.buf, measurement)
}

// StartLineRaw is the same as Start except that it accepts a byte slice
// instead of a string, which can save allocations.
func (e *Encoder) StartLineRaw(name []byte) {
	e.StartLine(unsafeBytesToString(name))
}

// AddTag adds a tag to the line. Tag keys must be added in lexical order
// and AddTag must be called after StartLine and before AddField.
//
// Neither the key or the value may contain non-printable ASCII
// characters (0x00 to 0x1f and 0x7f) or invalid UTF-8 or
// a trailing backslash character.
func (e *Encoder) AddTag(key, value string) {
	if e.section != tagSection {
		e.setErrorf("tag must be added after adding a measurement and before adding fields")
		return
	}
	if !e.lax {
		if !validMeasurementOrKey(key) {
			e.setErrorf("invalid tag key %q", key)
			return
		}
		if !validMeasurementOrKey(value) {
			e.setErrorf("invalid tag value %s=%q", key, value)
			return
		}
		if key <= string(e.prevTagKey) {
			e.setErrorf("tag key %q out of order (previous key %q)", key, e.prevTagKey)
			return
		}
		// We need to copy the tag key because AddTag can be called
		// by AddTagRaw with a slice of byte which might change from
		// call to call.
		e.prevTagKey = append(e.prevTagKey[:0], key...)
	}
	if e.lineHasError {
		return
	}
	e.buf = append(e.buf, ',')
	e.buf = tagKeyEscapes.appendEscaped(e.buf, key)
	e.buf = append(e.buf, '=')
	e.buf = tagValEscapes.appendEscaped(e.buf, value)
}

// AddTagRaw is like AddTag except that it accepts byte slices
// instead of strings, which can save allocations. Note that
// AddRawTag _will_ escape metacharacters such as "="
// and "," when they're present.
func (e *Encoder) AddTagRaw(key, value []byte) {
	e.AddTag(unsafeBytesToString(key), unsafeBytesToString(value))
}

// AddField adds a field to the line. AddField must be called after AddTag
// or AddMeasurement. At least one field must be added to each line.
func (e *Encoder) AddField(key string, value Value) {
	if e.section != fieldSection && e.section != tagSection {
		e.setErrorf("field must be added after tag or measurement section")
		return
	}
	section := e.section
	e.section = fieldSection
	if !e.lax {
		if !validMeasurementOrKey(key) {
			e.setErrorf("invalid field key %q", key)
			return
		}
	}
	if e.lineHasError {
		return
	}
	if section == tagSection {
		e.buf = append(e.buf, ' ')
	} else {
		e.buf = append(e.buf, ',')
	}
	e.buf = fieldKeyEscapes.appendEscaped(e.buf, key)
	e.buf = append(e.buf, '=')
	e.buf = value.AppendBytes(e.buf)
}

// AddFieldRaw is like AddField except that the key is represented
// as a byte slice instead of a string, which can save allocations.
// TODO would it be better for this to be:
//	AddFieldRaw(key []byte, kind ValueKind, data []byte) error
// so that we could respect lax and be more efficient when reading directly
// from a Decoder?
func (e *Encoder) AddFieldRaw(key []byte, value Value) {
	e.AddField(unsafeBytesToString(key), value)
}

var (
	minTime = time.Unix(0, math.MinInt64)
	maxTime = time.Unix(0, math.MaxInt64)
)

// EndLine adds the timestamp and newline at the end of the line.
// If t is zero, no timestamp will written and this method will do nothing.
// If the time is outside the maximum representable time range,
// an ErrRange error will be returned.
func (e *Encoder) EndLine(t time.Time) {
	if e.section != fieldSection {
		e.setErrorf("timestamp must be added after adding at least one field")
		return
	}
	e.section = endSection
	if t.IsZero() {
		// Zero timestamp. All we need is a newline.
		if !e.lineHasError {
			e.buf = append(e.buf, '\n')
		}
		return
	}
	if t.Before(minTime) || t.After(maxTime) {
		e.setErrorf("timestamp %s: %w", t.Format(time.RFC3339), ErrValueOutOfRange)
		return
	}
	if e.lineHasError {
		return
	}
	e.buf = append(e.buf, ' ')
	timestamp := t.UnixNano()
	if m := e.precisionMultiplier; m > 0 {
		timestamp /= m
	}
	e.buf = strconv.AppendInt(e.buf, timestamp, 10)
	e.buf = append(e.buf, '\n')
}

func (e *Encoder) setErrorf(format string, arg ...interface{}) {
	e.lineHasError = true
	if e.err == nil {
		if e.pointIndex <= 1 {
			e.err = fmt.Errorf(format, arg...)
		} else {
			e.err = fmt.Errorf("encoding point %d: %w", e.pointIndex-1, fmt.Errorf(format, arg...))
		}
	}
	// Remove the partially encoded part of the current line.
	e.buf = e.buf[:e.lineStart]
	if len(e.buf) == 0 {
		// Make sure the next entry doesn't add a newline.
		e.section = measurementSection
	}
}

// validMeasurementOrKey reports whether s can be
// encoded as a valid measurement or key.
func validMeasurementOrKey(s string) bool {
	if s == "" {
		return false
	}
	if !utf8.ValidString(s) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if nonPrintable.get(s[i]) {
			return false
		}
	}
	//lint:ignore S1008 Leave my comment alone!
	if s[len(s)-1] == '\\' {
		// A trailing backslash can't be round-tripped.
		return false
	}
	return true
}

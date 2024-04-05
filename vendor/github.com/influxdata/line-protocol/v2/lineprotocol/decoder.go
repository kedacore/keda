package lineprotocol

import (
	"bytes"
	"fmt"
	"io"
	"time"
	"unicode/utf8"
)

const (
	// When the buffer is grown, it will be grown by a minimum of 8K.
	minGrow = 8192

	// The buffer will be grown if there's less than minRead space available
	// to read into.
	minRead = minGrow / 2

	// maxSlide is the maximum number of bytes that will
	// be copied to the start of the buffer when reset is called.
	// This is a trade-off between copy overhead and the likelihood
	// that a complete line-protocol entry will fit into this size.
	maxSlide = 256
)

var (
	fieldSeparatorSpace   = newByteSet(" ")
	whitespace            = fieldSeparatorSpace.union(newByteSet("\r\n"))
	tagKeyChars           = newByteSet(",=").union(whitespace).union(nonPrintable).invert()
	tagKeyEscapes         = newEscaper(",= ")
	nonPrintable          = newByteSetRange(0, 31).union(newByteSet("\x7f"))
	eolChars              = newByteSet("\r\n")
	measurementChars      = newByteSet(", ").union(nonPrintable).invert()
	measurementEscapes    = newEscaper(" ,")
	tagValChars           = newByteSet(",=").union(whitespace).union(nonPrintable).invert()
	tagValEscapes         = newEscaper(", =")
	fieldKeyChars         = tagKeyChars
	fieldKeyEscapes       = tagKeyEscapes
	fieldStringValChars   = newByteSet(`"`).invert()
	fieldStringValEscapes = newEscaper("\\\"\n\r\t")
	fieldValChars         = newByteSet(",").union(whitespace).invert()
	timeChars             = newByteSet("-0123456789")
	commentChars          = nonPrintable.invert().without(eolChars)
	notEOL                = eolChars.invert()
	notNewline            = newByteSet("\n").invert()
)

// Decoder implements low level parsing of a set of line-protocol entries.
//
// Decoder methods must be called in the same order that their respective
// sections appear in a line-protocol entry. See the documentation on the
// Decoder.Next method for details.
type Decoder struct {
	// rd holds the reader, if any. If there is no reader,
	// complete will be true.
	rd io.Reader

	// buf holds data that's been read.
	buf []byte

	// r0 holds the earliest read position in buf.
	// Data in buf[0:r0] is considered to be discarded.
	r0 int

	// r1 holds the read position in buf. Data in buf[r1:] is
	// next to be read. Data in buf[len(buf):cap(buf)] is
	// available for reading into.
	r1 int

	// complete holds whether the data in buffer
	// is known to be all the data that's available.
	complete bool

	// section holds the current section of the entry that's being
	// read.
	section section

	// skipping holds whether we will need
	// to return the values that we're decoding.
	skipping bool

	// escBuf holds a buffer for unescaped characters.
	escBuf []byte

	// line holds the line number corresponding to the
	// character at buf[r1].
	line int64

	// err holds any non-EOF error that was returned from rd.
	err error
}

// NewDecoder returns a decoder that splits the line-protocol text
// inside buf.
func NewDecoderWithBytes(buf []byte) *Decoder {
	return &Decoder{
		buf:      buf,
		complete: true,
		escBuf:   make([]byte, 0, 512),
		section:  endSection,
		line:     1,
	}
}

// NewDecoder returns a decoder that reads from the given reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		rd:      r,
		escBuf:  make([]byte, 0, 512),
		section: endSection,
		line:    1,
	}
}

// Next advances to the next entry, and reports whether there is an
// entry available. Syntax errors on individual lines do not cause this
// to return false (the decoder attempts to recover from badly
// formatted lines), but I/O errors do. Call d.Err to discover if there
// was any I/O error. Syntax errors are returned as *DecoderError
// errors from Decoder methods.
//
// After calling Next, the various components of a line can be retrieved
// by calling Measurement, NextTag, NextField and Time in that order
// (the same order that the components are held in the entry).
//
// IMPORTANT NOTE: the byte slices returned by the Decoder methods are
// only valid until the next call to any other Decode method.
//
// Decoder will skip earlier components if a later method is called,
// but it doesn't retain the entire entry, so it cannot go backwards.
//
// For example, to retrieve only the timestamp of all lines, this suffices:
//
//	for d.Next() {
//		timestamp, err := d.TimeBytes()
//	}
//
func (d *Decoder) Next() bool {
	if _, err := d.advanceToSection(endSection); err != nil {
		// There was a syntax error and the line might not be
		// fully consumed, so make sure that we do actually
		// consume the rest of the line. This relies on the fact
		// that when we return a syntax error, we abandon the
		// rest of the line by going to newlineSection. If we
		// changed that behaviour (for example to allow obtaining
		// multiple errors per line), then we might need to loop here.
		d.advanceToSection(endSection)
	}
	d.skipEmptyLines()
	d.section = measurementSection
	return d.ensure(1)
}

// Err returns any I/O error encountered when reading
// entries. If d was created with NewDecoderWithBytes,
// Err will always return nil.
func (d *Decoder) Err() error {
	return d.err
}

// Measurement returns the measurement name. It returns nil
// unless called before NextTag, NextField or Time.
func (d *Decoder) Measurement() ([]byte, error) {
	if ok, err := d.advanceToSection(measurementSection); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	d.reset()
	measure, i0, err := d.takeEsc(measurementChars, &measurementEscapes.revTable)
	if err != nil {
		return nil, err
	}
	if len(measure) == 0 {
		if !d.ensure(1) {
			return nil, d.syntaxErrorf(i0, "no measurement name found")
		}
		return nil, d.syntaxErrorf(i0, "invalid character %q found at start of measurement name", d.at(0))
	}
	if measure[0] == '#' {
		// Comments are usually skipped earlier but if a comment contains invalid white space,
		// there's no way for the comment-parsing code to return an error, so instead
		// the read point is set to the start of the comment and we hit this case.
		// TODO find the actual invalid character to give a more accurate position.
		return nil, d.syntaxErrorf(i0, "invalid character found in comment line")
	}
	if err := d.advanceTagComma(); err != nil {
		return nil, err
	}
	d.section = tagSection
	return measure, nil
}

// NextTag returns the next tag in the entry.
// If there are no more tags, it returns nil, nil, nil.
// Note that this must be called before NextField because
// tags precede fields in the line-protocol entry.
func (d *Decoder) NextTag() (key, value []byte, err error) {
	if ok, err := d.advanceToSection(tagSection); err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, nil
	}
	if d.ensure(1) && fieldSeparatorSpace.get(d.at(0)) {
		d.take(fieldSeparatorSpace)
		d.section = fieldSection
		return nil, nil, nil
	}
	tagKey, i0, err := d.takeEsc(tagKeyChars, &tagKeyEscapes.revTable)
	if err != nil {
		return nil, nil, err
	}
	if len(tagKey) == 0 || !d.ensure(1) || d.at(0) != '=' {
		if !d.ensure(1) {
			return nil, nil, d.syntaxErrorf(i0, "empty tag name")
		}
		if len(tagKey) > 0 {
			return nil, nil, d.syntaxErrorf(i0, "expected '=' after tag key %q, but got %q instead", tagKey, d.at(0))
		}
		return nil, nil, d.syntaxErrorf(i0, "expected tag key or field but found %q instead", d.at(0))
	}
	d.advance(1)
	tagVal, i0, err := d.takeEsc(tagValChars, &tagValEscapes.revTable)
	if err != nil {
		return nil, nil, err
	}
	if len(tagVal) == 0 {
		return nil, nil, d.syntaxErrorf(i0, "expected tag value after tag key %q, but none found", tagKey)
	}
	if !d.ensure(1) {
		// There's no more data after the tag value. Instead of returning an error
		// immediately, advance to the field section and return the tag and value.
		// This means that we'll see all the tags even when there's no value,
		// and it also allows a client to parse the tags in isolation even when there
		// are no keys. We'll return an error if the client tries to read values from here.
		d.section = fieldSection
		return tagKey, tagVal, nil
	}
	if err := d.advanceTagComma(); err != nil {
		return nil, nil, err
	}
	return tagKey, tagVal, nil
}

// advanceTagComma consumes a comma after a measurement
// or a tag value, making sure it's not followed by whitespace.
func (d *Decoder) advanceTagComma() error {
	if !d.ensure(1) {
		return nil
	}
	nextc := d.at(0)
	if nextc != ',' {
		return nil
	}
	// If there's a comma, there's a tag, so check that there's the start
	// of a tag name there.
	d.advance(1)
	if !d.ensure(1) {
		return d.syntaxErrorf(d.r1-d.r0, "expected tag key after comma; got end of input")
	}
	if whitespace.get(d.at(0)) {
		return d.syntaxErrorf(d.r1-d.r0, "expected tag key after comma; got white space instead")
	}
	return nil
}

// NextFieldBytes returns the next field in the entry.
// If there are no more fields, it returns all zero values.
// Note that this must be called before Time because
// fields precede the timestamp in the line-protocol entry.
//
// The returned value slice may not be valid: to
// check its validity, use NewValueFromBytes(kind, value), or use NextField.
func (d *Decoder) NextFieldBytes() (key []byte, kind ValueKind, value []byte, err error) {
	if ok, err := d.advanceToSection(fieldSection); err != nil {
		return nil, Unknown, nil, err
	} else if !ok {
		return nil, Unknown, nil, nil
	}
	fieldKey, i0, err := d.takeEsc(fieldKeyChars, &fieldKeyEscapes.revTable)
	if err != nil {
		return nil, Unknown, nil, err
	}
	if len(fieldKey) == 0 {
		if !d.ensure(1) {
			return nil, Unknown, nil, d.syntaxErrorf(i0, "expected field key but none found")
		}
		return nil, Unknown, nil, d.syntaxErrorf(i0, "invalid character %q found at start of field key", d.at(0))
	}
	if !d.ensure(1) {
		return nil, Unknown, nil, d.syntaxErrorf(d.r1-d.r0, "want '=' after field key %q, found end of input", fieldKey)
	}
	if nextc := d.at(0); nextc != '=' {
		return nil, Unknown, nil, d.syntaxErrorf(d.r1-d.r0, "want '=' after field key %q, found %q", fieldKey, nextc)
	}
	d.advance(1)
	if !d.ensure(1) {
		return nil, Unknown, nil, d.syntaxErrorf(d.r1-d.r0, "expected field value, found end of input")
	}
	var fieldVal []byte
	var fieldKind ValueKind
	switch d.at(0) {
	case '"':
		// Skip leading quote.
		d.advance(1)
		var err error
		fieldVal, i0, err = d.takeEsc(fieldStringValChars, &fieldStringValEscapes.revTable)
		if err != nil {
			return nil, Unknown, nil, err
		}
		fieldKind = String
		if !d.ensure(1) {
			return nil, Unknown, nil, d.syntaxErrorf(i0-1, "expected closing quote for string field value, found end of input")
		}
		if d.at(0) != '"' {
			// This can't happen, as all characters are allowed in a string.
			return nil, Unknown, nil, d.syntaxErrorf(i0-1, "unexpected string termination")
		}
		// Skip trailing quote
		d.advance(1)
	case 't', 'T', 'f', 'F':
		fieldVal = d.take(fieldValChars)
		fieldKind = Bool
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		fieldVal = d.take(fieldValChars)
		switch fieldVal[len(fieldVal)-1] {
		case 'i':
			fieldVal = fieldVal[:len(fieldVal)-1]
			fieldKind = Int
		case 'u':
			fieldVal = fieldVal[:len(fieldVal)-1]
			fieldKind = Uint
		default:
			fieldKind = Float
		}
	default:
		return nil, Unknown, nil, d.syntaxErrorf(d.r1-d.r0, "field value has unrecognized type")
	}
	if !d.ensure(1) {
		d.section = endSection
		return fieldKey, fieldKind, fieldVal, nil
	}
	nextc := d.at(0)
	if nextc == ',' {
		d.advance(1)
		return fieldKey, fieldKind, fieldVal, nil
	}
	if !whitespace.get(nextc) {
		return nil, Unknown, nil, d.syntaxErrorf(d.r1-d.r0, "unexpected character %q after field", nextc)
	}
	d.take(fieldSeparatorSpace)
	if d.takeEOL() {
		d.section = endSection
		return fieldKey, fieldKind, fieldVal, nil
	}
	d.section = timeSection
	return fieldKey, fieldKind, fieldVal, nil
}

// takeEOL consumes input up until the next end of line.
func (d *Decoder) takeEOL() bool {
	if !d.ensure(1) {
		// End of input.
		return true
	}
	switch d.at(0) {
	case '\n':
		// Regular NL.
		d.advance(1)
		d.line++
		return true
	case '\r':
		if !d.ensure(2) {
			// CR at end of input.
			d.advance(1)
			return true
		}
		if d.at(1) == '\n' {
			// CR-NL
			d.advance(2)
			d.line++
			return true
		}
	}
	return false
}

// NextField is a wrapper around NextFieldBytes that parses
// the field value. Note: the returned value is only valid
// until the next call method call on Decoder because when
// it's a string, it refers to an internal buffer.
//
// If the value cannot be parsed because it's out of range
// (as opposed to being syntactically invalid),
// the errors.Is(err, ErrValueOutOfRange) will return true.
func (d *Decoder) NextField() (key []byte, val Value, err error) {
	// Even though NextFieldBytes calls advanceToSection,
	// we need to call it here too so that we know exactly where
	// the field starts so that startIndex is accurate.
	if ok, err := d.advanceToSection(fieldSection); err != nil {
		return nil, Value{}, err
	} else if !ok {
		return nil, Value{}, nil
	}
	startIndex := d.r1 - d.r0
	key, kind, data, err := d.NextFieldBytes()
	if err != nil || key == nil {
		return nil, Value{}, err
	}

	v, err := newValueFromBytes(kind, data, false)
	if err != nil {
		// We want to produce an error that points to where the field
		// location, but NextFieldBytes has read past that.
		// However, we know the key length, and we can work out
		// the how many characters it took when escaped, so
		// we can reconstruct the index of the start of the field.
		startIndex += tagKeyEscapes.escapedLen(unsafeBytesToString(key)) + len("=")
		return nil, Value{}, d.syntaxErrorf(startIndex, "cannot parse value for field key %q: %w", key, err)
	}
	return key, v, nil
}

// TimeBytes returns the timestamp of the entry as a byte slice.
// If there is no timestamp, it returns nil, nil.
func (d *Decoder) TimeBytes() ([]byte, error) {
	if ok, err := d.advanceToSection(timeSection); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	start := d.r1 - d.r0
	timeBytes := d.take(timeChars)
	if len(timeBytes) == 0 {
		d.section = endSection
		timeBytes = nil
	}
	if !d.ensure(1) {
		d.section = endSection
		return timeBytes, nil
	}
	if !whitespace.get(d.at(0)) {
		// Absorb the rest of the line so that we get a better error.
		d.take(notEOL)
		return nil, d.syntaxErrorf(start, "invalid timestamp (%q)", d.buf[d.r0+start:d.r1])
	}
	d.take(fieldSeparatorSpace)
	if !d.ensure(1) {
		d.section = endSection
		return timeBytes, nil
	}
	if !d.takeEOL() {
		start := d.r1 - d.r0
		extra := d.take(notEOL)
		return nil, d.syntaxErrorf(start, "unexpected text after timestamp (%q)", extra)
	}
	d.section = endSection
	return timeBytes, nil
}

// Time is a wrapper around TimeBytes that returns the timestamp
// assuming the given precision.
func (d *Decoder) Time(prec Precision, defaultTime time.Time) (time.Time, error) {
	data, err := d.TimeBytes()
	if err != nil {
		return time.Time{}, err
	}
	if data == nil {
		return defaultTime.Truncate(prec.Duration()), nil
	}
	ts, err := parseIntBytes(data, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", maybeOutOfRange(err, "invalid syntax"))
	}
	ns, ok := prec.asNanoseconds(ts)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", ErrValueOutOfRange)
	}
	return time.Unix(0, ns), nil
}

// consumeLine is used to recover from errors by reading an entire
// line even if it contains invalid characters.
func (d *Decoder) consumeLine() {
	d.take(notNewline)
	if d.at(0) == '\n' {
		d.advance(1)
		d.line++
	}
	d.reset()
	d.section = endSection
}

func (d *Decoder) skipEmptyLines() {
	for {
		startLine := d.r1 - d.r0
		d.take(fieldSeparatorSpace)
		switch d.at(0) {
		case '#':
			// Found a comment.
			d.take(commentChars)
			if !d.takeEOL() {
				// Comment has invalid characters.
				// Rewind input to start of comment so
				// that next section will return the error.
				d.r1 = d.r0 + startLine
				return
			}
		case '\n':
			d.line++
			d.advance(1)
		case '\r':
			if !d.takeEOL() {
				// Solitary carriage return.
				// Leave it there and next section will return an error.
				return
			}
		default:
			return
		}
	}
}

func (d *Decoder) advanceToSection(section section) (bool, error) {
	if d.section == section {
		return true, nil
	}
	if d.section > section {
		return false, nil
	}
	// Enable skipping to avoid unnecessary unescaping work.
	d.skipping = true
	for d.section < section {
		if err := d.consumeSection(); err != nil {
			d.skipping = false
			return false, err
		}
	}
	d.skipping = false
	return d.section == section, nil
}

//go:generate stringer -type section

// section represents one decoder section of a line-protocol entry.
// An entry consists of a measurement (measurementSection),
// an optional set of tags (tagSection), one or more fields (fieldSection)
// and an option timestamp (timeSection).
type section byte

const (
	measurementSection section = iota
	tagSection
	fieldSection
	timeSection

	// newlineSection represents the newline at the end of the line.
	// This section also absorbs any invalid characters at the end
	// of the line - it's used as a recovery state if we find an error
	// when parsing an earlier part of an entry.
	newlineSection

	// endSection represents the end of an entry. When we're at this
	// stage, calling More will cycle back to measurementSection again.
	endSection
)

func (d *Decoder) consumeSection() error {
	switch d.section {
	case measurementSection:
		_, err := d.Measurement()
		return err
	case tagSection:
		for {
			key, _, err := d.NextTag()
			if err != nil || key == nil {
				return err
			}
		}
	case fieldSection:
		for {
			key, _, _, err := d.NextFieldBytes()
			if err != nil || key == nil {
				return err
			}
		}
	case timeSection:
		_, err := d.TimeBytes()
		return err
	case newlineSection:
		d.consumeLine()
		return nil
	default:
		return nil
	}
}

// take returns the next slice of bytes that are in the given set
// reading more data as needed. It updates d.r1.
//
// Note: we assume that the set never contains the newline
// character because newlines can only occur when explicitly
// allowed (in string field values and at the end of an entry),
// so we don't need to update d.line.
func (d *Decoder) take(set *byteSet) []byte {
	// Note: use a relative index for start because absolute
	// indexes aren't stable (the contents of the buffer can be
	// moved when reading more data).
	start := d.r1 - d.r0
outer:
	for {
		if !d.ensure(1) {
			break
		}
		buf := d.buf[d.r1:]
		for i, c := range buf {
			if !set.get(c) {
				d.r1 += i
				break outer
			}
		}
		d.r1 += len(buf)
	}
	return d.buf[d.r0+start : d.r1]
}

// takeEsc is like take except that escaped characters also count as
// part of the set. The escapeTable determines which characters
// can be escaped.
//
// It returns the unescaped string (unless d.skipping is true, in which
// case it doesn't need to go to the trouble of unescaping it), and the
// index into buf that corresponds to the start of the taken bytes.
//
// takeEsc also returns the offset of the start of the escaped bytes
// relative to d.r0.
//
// It returns an error if the returned string contains an
// invalid UTF-8 sequence. The other return parameters are unaffected by this.
func (d *Decoder) takeEsc(set *byteSet, escapeTable *[256]byte) ([]byte, int, error) {
	// start holds the offset from r0 of the start of the taken slice.
	// Note that we can't use d.r1 directly, because the offsets can change
	// when the buffer is grown.
	start := d.r1 - d.r0

	// startUnesc holds the offset from t0 of the start of the most recent
	// unescaped segment.
	startUnesc := start

	// startEsc holds the index into r.escBuf of the start of the escape buffer.
	startEsc := len(d.escBuf)
outer:
	for {
		if !d.ensure(1) {
			break
		}
		buf := d.buf[d.r1:]
		for i := 0; i < len(buf); i++ {
			c := buf[i]
			if c != '\\' {
				if !set.get(c) {
					// We've found the end, so we're done here.
					d.r1 += i
					break outer
				}
				continue
			}
			if i+1 >= len(buf) {
				// Not enough room in the buffer. Try reading more so that
				// we can see the next byte (note: ensure(i+2) is asking
				// for exactly one more character, because we know we already
				// have i+1 bytes in the buffer).
				if !d.ensure(i + 2) {
					// No character to escape, so leave the \ intact.
					d.r1 = len(d.buf)
					break outer
				}
				// Note that d.ensure can change d.buf, so we need to
				// update buf to point to the correct buffer.
				buf = d.buf[d.r1:]
			}
			replc := escapeTable[buf[i+1]]
			if replc == 0 {
				// The backslash doesn't precede a value escaped
				// character, so it stays intact.
				continue
			}
			if !d.skipping {
				d.escBuf = append(d.escBuf, d.buf[d.r0+startUnesc:d.r1+i]...)
				d.escBuf = append(d.escBuf, replc)
				startUnesc = d.r1 - d.r0 + i + 2
			}
			i++
		}
		// We've consumed all the bytes in the buffer. Now continue
		// the loop, trying to acquire more.
		d.r1 += len(buf)
	}
	taken := d.buf[d.r0+start : d.r1]
	if set.get('\n') {
		d.line += int64(bytes.Count(taken, newlineBytes))
	}
	if len(d.escBuf) > startEsc {
		// We've got an unescaped result: append any remaining unescaped bytes
		// and return the relevant portion of the escape buffer.
		d.escBuf = append(d.escBuf, d.buf[startUnesc+d.r0:d.r1]...)
		taken = d.escBuf[startEsc:]
	}
	if !utf8.Valid(taken) {
		// TODO point directly to the offending sequence.
		return taken, start, d.syntaxErrorf(start, "invalid utf-8 sequence in token %q", taken)
	}
	return taken, start, nil
}

var newlineBytes = []byte{'\n'}

// at returns the byte at i bytes after the current read position.
// It assumes that the index has already been ensured.
// If there's no byte there, it returns zero.
func (d *Decoder) at(i int) byte {
	if d.r1+i < len(d.buf) {
		return d.buf[d.r1+i]
	}
	return 0
}

// reset discards all the data up to d.r1 and data in d.escBuf
func (d *Decoder) reset() {
	if unread := len(d.buf) - d.r1; unread == 0 {
		// No bytes in the buffer, so we can start from the beginning without
		// needing to copy anything (and get better cache behaviour too).
		d.buf = d.buf[:0]
		d.r1 = 0
	} else if !d.complete && unread <= maxSlide {
		// Slide the unread portion of the buffer to the
		// start so that when we read more data,
		// there's less chance that we'll need to grow the buffer.
		copy(d.buf, d.buf[d.r1:])
		d.r1 = 0
		d.buf = d.buf[:unread]
	}
	d.r0 = d.r1
	d.escBuf = d.escBuf[:0]
}

// advance advances the read point by n.
// This should only be used when it's known that
// there are already n bytes available in the buffer.
func (d *Decoder) advance(n int) {
	d.r1 += n
}

// ensure ensures that there are at least n bytes available in
// d.buf[d.r1:], reading more bytes if necessary.
// It reports whether enough bytes are available.
func (d *Decoder) ensure(n int) bool {
	if d.r1+n <= len(d.buf) {
		// There are enough bytes available.
		return true
	}
	return d.ensure1(n)
}

// ensure1 is factored out of ensure so that ensure
// itself can be inlined.
func (d *Decoder) ensure1(n int) bool {
	for {
		if d.complete {
			// No possibility of more data.
			return false
		}
		d.readMore()
		if d.r1+n <= len(d.buf) {
			// There are enough bytes available.
			return true
		}
	}
}

// readMore reads more data into d.buf.
func (d *Decoder) readMore() {
	if d.complete {
		return
	}
	n := cap(d.buf) - len(d.buf)
	if n < minRead {
		// We need to grow the buffer. Note that we don't have to copy
		// the unused part of the buffer (d.buf[:d.r0]).
		// TODO provide a way to limit the maximum size that
		// the buffer can grow to.
		used := len(d.buf) - d.r0
		n1 := cap(d.buf) * 2
		if n1-used < minGrow {
			n1 = used + minGrow
		}
		buf1 := make([]byte, used, n1)
		copy(buf1, d.buf[d.r0:])
		d.buf = buf1
		d.r1 -= d.r0
		d.r0 = 0
	}
	n, err := d.rd.Read(d.buf[len(d.buf):cap(d.buf)])
	d.buf = d.buf[:len(d.buf)+n]
	if err == nil {
		return
	}
	d.complete = true
	if err != io.EOF {
		d.err = err
	}
}

// syntaxErrorf records a syntax error at the given offset from d.r0
// and the using the given fmt.Sprintf-formatted message.
func (d *Decoder) syntaxErrorf(offset int, f string, a ...interface{}) error {
	// Note: we only ever reset the buffer at the end of an entry,
	// so we can assume that that d.r0 corresponds to column 1.
	buf := d.buf[d.r0 : d.r0+offset]
	var columnBytes []byte
	if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
		columnBytes = buf[i+1:]
	} else {
		columnBytes = buf
	}
	column := len(columnBytes) + 1

	// Note: line corresponds to the current line at d.r1, so if
	// there are any newlines after the location of the error, we need to
	// reduce the line we report accordingly.
	remain := d.buf[d.r0+offset : d.r1]
	line := d.line - int64(bytes.Count(remain, newlineBytes))

	// We'll recover from a syntax error by reading all bytes until
	// the next newline. We don't want to do that if we've already
	// just scanned the end of a line.
	if d.section != endSection {
		d.section = newlineSection
	}
	return &DecodeError{
		Line:   line,
		Column: column,
		Err:    fmt.Errorf(f, a...),
	}
}

// DecodeError represents an error when decoding a line-protocol entry.
type DecodeError struct {
	// Line holds the one-based index of the line where the error occurred.
	Line int64
	// Column holds the one-based index of the column (in bytes) where the error occurred.
	Column int
	// Err holds the underlying error.
	Err error
}

// Error implements the error interface.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("at line %d:%d: %s", e.Line, e.Column, e.Err.Error())
}

// Unwrap implements error unwrapping so that the underlying
// error can be retrieved.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

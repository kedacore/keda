package parser

import (
	"bytes"
	"fmt"
	"strconv"
)

type regExpParser struct {
	goRegexp  *bytes.Buffer
	str       string
	errors    []error
	length    int
	chrOffset int
	offset    int
	chr       rune
	invalid   bool
}

// TransformRegExp transforms a JavaScript pattern into  a Go "regexp" pattern.
//
// re2 (Go) cannot do backtracking, so the presence of a lookahead (?=) (?!) or
// backreference (\1, \2, ...) will cause an error.
//
// re2 (Go) has a different definition for \s: [\t\n\f\r ].
// The JavaScript definition, on the other hand, also includes \v, Unicode "Separator, Space", etc.
//
// If the pattern is invalid (not valid even in JavaScript), then this function
// returns the empty string and an error.
//
// If the pattern is valid, but incompatible (contains a lookahead or backreference),
// then this function returns the transformation (a non-empty string) AND an error.
func TransformRegExp(pattern string) (string, error) {
	if pattern == "" {
		return "", nil
	}

	// TODO If without \, if without (?=, (?!, then another shortcut

	p := regExpParser{
		str:      pattern,
		length:   len(pattern),
		goRegexp: bytes.NewBuffer(make([]byte, 0, 3*len(pattern)/2)),
	}
	p.read() // Pull in the first character
	p.scan()
	var err error
	if len(p.errors) > 0 {
		err = p.errors[0]
	}
	if p.invalid {
		return "", err
	}

	// Might not be re2 compatible, but is still a valid JavaScript RegExp
	return p.goRegexp.String(), err
}

func (p *regExpParser) scan() {
	for p.chr != -1 {
		switch p.chr {
		case '\\':
			p.read()
			p.scanEscape(false)
		case '(':
			p.pass()
			p.scanGroup()
		case '[':
			p.pass()
			p.scanBracket()
		case ')':
			p.error(-1, "Unmatched ')'")
			p.invalid = true
			p.pass()
		default:
			p.pass()
		}
	}
}

// (...)
func (p *regExpParser) scanGroup() {
	str := p.str[p.chrOffset:]
	if len(str) > 1 { // A possibility of (?= or (?!
		if str[0] == '?' {
			if str[1] == '=' || str[1] == '!' {
				p.error(-1, "re2: Invalid (%s) <lookahead>", p.str[p.chrOffset:p.chrOffset+2])
			}
		}
	}
	for p.chr != -1 && p.chr != ')' {
		switch p.chr {
		case '\\':
			p.read()
			p.scanEscape(false)
		case '(':
			p.pass()
			p.scanGroup()
		case '[':
			p.pass()
			p.scanBracket()
		default:
			p.pass()
			continue
		}
	}
	if p.chr != ')' {
		p.error(-1, "Unterminated group")
		p.invalid = true
		return
	}
	p.pass()
}

// [...].
func (p *regExpParser) scanBracket() {
	for p.chr != -1 {
		if p.chr == ']' {
			break
		} else if p.chr == '\\' {
			p.read()
			p.scanEscape(true)
			continue
		}
		p.pass()
	}
	if p.chr != ']' {
		p.error(-1, "Unterminated character class")
		p.invalid = true
		return
	}
	p.pass()
}

// \...
func (p *regExpParser) scanEscape(inClass bool) {
	offset := p.chrOffset

	var length, base uint32
	switch p.chr {
	case '0', '1', '2', '3', '4', '5', '6', '7':
		var value int64
		size := 0
		for {
			digit := int64(digitValue(p.chr))
			if digit >= 8 {
				// Not a valid digit
				break
			}
			value = value*8 + digit
			p.read()
			size++
		}
		if size == 1 { // The number of characters read
			_, err := p.goRegexp.Write([]byte{'\\', byte(value) + '0'})
			if err != nil {
				p.errors = append(p.errors, err)
			}
			if value != 0 {
				// An invalid backreference
				p.error(-1, "re2: Invalid \\%d <backreference>", value)
			}
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		_, err := p.goRegexp.Write(tmp)
		if err != nil {
			p.errors = append(p.errors, err)
		}
		return

	case '8', '9':
		size := 0
		for {
			digit := digitValue(p.chr)
			if digit >= 10 {
				// Not a valid digit
				break
			}
			p.read()
			size++
		}
		err := p.goRegexp.WriteByte('\\')
		if err != nil {
			p.errors = append(p.errors, err)
		}
		_, err = p.goRegexp.WriteString(p.str[offset:p.chrOffset])
		if err != nil {
			p.errors = append(p.errors, err)
		}
		p.error(-1, "re2: Invalid \\%s <backreference>", p.str[offset:p.chrOffset])
		return

	case 'x':
		p.read()
		length, base = 2, 16

	case 'u':
		p.read()
		length, base = 4, 16

	case 'b':
		if inClass {
			_, err := p.goRegexp.Write([]byte{'\\', 'x', '0', '8'})
			if err != nil {
				p.errors = append(p.errors, err)
			}
			p.read()
			return
		}
		fallthrough

	case 'B':
		fallthrough

	case 'd', 'D', 's', 'S', 'w', 'W':
		// This is slightly broken, because ECMAScript
		// includes \v in \s, \S, while re2 does not
		fallthrough

	case '\\':
		fallthrough

	case 'f', 'n', 'r', 't', 'v':
		err := p.goRegexp.WriteByte('\\')
		if err != nil {
			p.errors = append(p.errors, err)
		}
		p.pass()
		return

	case 'c':
		p.read()
		var value int64
		switch {
		case 'a' <= p.chr && p.chr <= 'z':
			value = int64(p.chr) - 'a' + 1
		case 'A' <= p.chr && p.chr <= 'Z':
			value = int64(p.chr) - 'A' + 1
		default:
			err := p.goRegexp.WriteByte('c')
			if err != nil {
				p.errors = append(p.errors, err)
			}
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		_, err := p.goRegexp.Write(tmp)
		if err != nil {
			p.errors = append(p.errors, err)
		}
		p.read()
		return

	default:
		// $ is an identifier character, so we have to have
		// a special case for it here
		if p.chr == '$' || !isIdentifierPart(p.chr) {
			// A non-identifier character needs escaping
			err := p.goRegexp.WriteByte('\\')
			if err != nil {
				p.errors = append(p.errors, err)
			}
		} else { //nolint:staticcheck
			// Unescape the character for re2
		}
		p.pass()
		return
	}

	// Otherwise, we're a \u.... or \x...
	valueOffset := p.chrOffset

	var value uint32
	for length := length; length > 0; length-- {
		digit := uint32(digitValue(p.chr))
		if digit >= base {
			// Not a valid digit
			goto skip
		}
		value = value*base + digit
		p.read()
	}

	switch length {
	case 4:
		if _, err := p.goRegexp.Write([]byte{
			'\\',
			'x',
			'{',
			p.str[valueOffset+0],
			p.str[valueOffset+1],
			p.str[valueOffset+2],
			p.str[valueOffset+3],
			'}',
		}); err != nil {
			p.errors = append(p.errors, err)
		}
	case 2:
		if _, err := p.goRegexp.Write([]byte{
			'\\',
			'x',
			p.str[valueOffset+0],
			p.str[valueOffset+1],
		}); err != nil {
			p.errors = append(p.errors, err)
		}
	default:
		// Should never, ever get here...
		p.error(-1, "re2: Illegal branch in scanEscape")
		goto skip
	}

	return

skip:
	_, err := p.goRegexp.WriteString(p.str[offset:p.chrOffset])
	if err != nil {
		p.errors = append(p.errors, err)
	}
}

func (p *regExpParser) pass() {
	if p.chr != -1 {
		_, err := p.goRegexp.WriteRune(p.chr)
		if err != nil {
			p.errors = append(p.errors, err)
		}
	}
	p.read()
}

// TODO Better error reporting, use the offset, etc.
func (p *regExpParser) error(offset int, msg string, msgValues ...interface{}) { //nolint:unparam
	err := fmt.Errorf(msg, msgValues...)
	p.errors = append(p.errors, err)
}

package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

type chr struct { //nolint:unused
	value rune
	width int
}

var matchIdentifier = regexp.MustCompile(`^[$_\p{L}][$_\p{L}\d}]*$`)

func isDecimalDigit(chr rune) bool {
	return '0' <= chr && chr <= '9'
}

func digitValue(chr rune) int {
	switch {
	case '0' <= chr && chr <= '9':
		return int(chr - '0')
	case 'a' <= chr && chr <= 'f':
		return int(chr - 'a' + 10)
	case 'A' <= chr && chr <= 'F':
		return int(chr - 'A' + 10)
	}
	return 16 // Larger than any legal digit value
}

// See https://www.unicode.org/reports/tr31/ for reference on ID_Start and ID_Continue.
var includeIDStart = []*unicode.RangeTable{
	unicode.Lu,
	unicode.Ll,
	unicode.Lt,
	unicode.Lm,
	unicode.Lo,
	unicode.Nl,
	unicode.Other_ID_Start,
}

var includeIDContinue = []*unicode.RangeTable{
	unicode.Lu,
	unicode.Ll,
	unicode.Lt,
	unicode.Lm,
	unicode.Lo,
	unicode.Nl,
	unicode.Other_ID_Start,
	unicode.Mn,
	unicode.Mc,
	unicode.Nd,
	unicode.Pc,
	unicode.Other_ID_Continue,
}

var exclude = []*unicode.RangeTable{
	unicode.Pattern_Syntax,
	unicode.Pattern_White_Space,
}

func unicodeIDStart(r rune) bool {
	if unicode.In(r, exclude...) {
		return false
	}

	return unicode.In(r, includeIDStart...)
}

func unicodeIDContinue(r rune) bool {
	if unicode.In(r, exclude...) {
		return false
	}

	return unicode.In(r, includeIDContinue...)
}

func isDigit(chr rune, base int) bool {
	return digitValue(chr) < base
}

func isIdentifierStart(chr rune) bool {
	return chr == '$' || chr == '_' || chr == '\\' ||
		'a' <= chr && chr <= 'z' || 'A' <= chr && chr <= 'Z' ||
		chr >= utf8.RuneSelf && unicodeIDStart(chr)
}

func isIdentifierPart(chr rune) bool {
	return chr == '$' || chr == '_' || chr == '\\' ||
		'a' <= chr && chr <= 'z' || 'A' <= chr && chr <= 'Z' ||
		'0' <= chr && chr <= '9' ||
		chr >= utf8.RuneSelf && unicodeIDContinue(chr)
}

func (p *parser) scanIdentifier() (string, error) {
	offset := p.chrOffset
	parse := false
	for isIdentifierPart(p.chr) {
		if p.chr == '\\' {
			distance := p.chrOffset - offset
			p.read()
			if p.chr != 'u' {
				return "", fmt.Errorf("invalid identifier escape character: %c (%s)", p.chr, string(p.chr))
			}
			parse = true
			var value rune
			for range 4 {
				p.read()
				decimal, ok := hex2decimal(byte(p.chr))
				if !ok {
					return "", fmt.Errorf("invalid identifier escape character: %c (%s)", p.chr, string(p.chr))
				}
				value = value<<4 | decimal
			}
			switch {
			case value == '\\':
				return "", fmt.Errorf("invalid identifier escape value: %c (%s)", value, string(value))
			case distance == 0:
				if !isIdentifierStart(value) {
					return "", fmt.Errorf("invalid identifier escape value: %c (%s)", value, string(value))
				}
			case distance > 0:
				if !isIdentifierPart(value) {
					return "", fmt.Errorf("invalid identifier escape value: %c (%s)", value, string(value))
				}
			}
		}
		p.read()
	}
	literal := p.str[offset:p.chrOffset]
	if parse {
		return parseStringLiteral(literal)
	}
	return literal, nil
}

// 7.2.
func isLineWhiteSpace(chr rune) bool { //nolint:unused, deadcode
	switch chr {
	case '\u0009', '\u000b', '\u000c', '\u0020', '\u00a0', '\ufeff':
		return true
	case '\u000a', '\u000d', '\u2028', '\u2029':
		return false
	case '\u0085':
		return false
	}
	return unicode.IsSpace(chr)
}

// 7.3.
func isLineTerminator(chr rune) bool {
	switch chr {
	case '\u000a', '\u000d', '\u2028', '\u2029':
		return true
	}
	return false
}

func (p *parser) scan() (tkn token.Token, literal string, idx file.Idx) { //nolint:nonamedreturns
	p.implicitSemicolon = false

	for {
		p.skipWhiteSpace()

		idx = p.idxOf(p.chrOffset)
		insertSemicolon := false

		switch chr := p.chr; {
		case isIdentifierStart(chr):
			var err error
			literal, err = p.scanIdentifier()
			if err != nil {
				tkn = token.ILLEGAL
				break
			}
			if len(literal) > 1 {
				// Keywords are longer than 1 character, avoid lookup otherwise
				var strict bool
				tkn, strict = token.IsKeyword(literal)

				switch tkn {
				case 0: // Not a keyword
					switch literal {
					case "true", "false":
						p.insertSemicolon = true
						return token.BOOLEAN, literal, idx
					case "null":
						p.insertSemicolon = true
						return token.NULL, literal, idx
					}
				case token.KEYWORD:
					if strict {
						// TODO If strict and in strict mode, then this is not a break
						break
					}
					return token.KEYWORD, literal, idx

				case
					token.THIS,
					token.BREAK,
					token.THROW, // A newline after a throw is not allowed, but we need to detect it
					token.RETURN,
					token.CONTINUE,
					token.DEBUGGER:
					p.insertSemicolon = true
					return tkn, literal, idx

				default:
					return tkn, literal, idx
				}
			}
			p.insertSemicolon = true
			return token.IDENTIFIER, literal, idx
		case '0' <= chr && chr <= '9':
			p.insertSemicolon = true
			tkn, literal = p.scanNumericLiteral(false)
			return tkn, literal, idx
		default:
			p.read()
			switch chr {
			case -1:
				if p.insertSemicolon {
					p.insertSemicolon = false
					p.implicitSemicolon = true
				}
				tkn = token.EOF
			case '\r', '\n', '\u2028', '\u2029':
				p.insertSemicolon = false
				p.implicitSemicolon = true
				p.comments.AtLineBreak()
				continue
			case ':':
				tkn = token.COLON
			case '.':
				if digitValue(p.chr) < 10 {
					insertSemicolon = true
					tkn, literal = p.scanNumericLiteral(true)
				} else {
					tkn = token.PERIOD
				}
			case ',':
				tkn = token.COMMA
			case ';':
				tkn = token.SEMICOLON
			case '(':
				tkn = token.LEFT_PARENTHESIS
			case ')':
				tkn = token.RIGHT_PARENTHESIS
				insertSemicolon = true
			case '[':
				tkn = token.LEFT_BRACKET
			case ']':
				tkn = token.RIGHT_BRACKET
				insertSemicolon = true
			case '{':
				tkn = token.LEFT_BRACE
			case '}':
				tkn = token.RIGHT_BRACE
				insertSemicolon = true
			case '+':
				tkn = p.switch3(token.PLUS, token.ADD_ASSIGN, '+', token.INCREMENT)
				if tkn == token.INCREMENT {
					insertSemicolon = true
				}
			case '-':
				tkn = p.switch3(token.MINUS, token.SUBTRACT_ASSIGN, '-', token.DECREMENT)
				if tkn == token.DECREMENT {
					insertSemicolon = true
				}
			case '*':
				tkn = p.switch2(token.MULTIPLY, token.MULTIPLY_ASSIGN)
			case '/':
				switch p.chr {
				case '/':
					if p.mode&StoreComments != 0 {
						comment := string(p.readSingleLineComment())
						p.comments.AddComment(ast.NewComment(comment, idx))
						continue
					}
					p.skipSingleLineComment()
					continue
				case '*':
					if p.mode&StoreComments != 0 {
						comment := string(p.readMultiLineComment())
						p.comments.AddComment(ast.NewComment(comment, idx))
						continue
					}
					p.skipMultiLineComment()
					continue
				default:
					// Could be division, could be RegExp literal
					tkn = p.switch2(token.SLASH, token.QUOTIENT_ASSIGN)
					insertSemicolon = true
				}
			case '%':
				tkn = p.switch2(token.REMAINDER, token.REMAINDER_ASSIGN)
			case '^':
				tkn = p.switch2(token.EXCLUSIVE_OR, token.EXCLUSIVE_OR_ASSIGN)
			case '<':
				tkn = p.switch4(token.LESS, token.LESS_OR_EQUAL, '<', token.SHIFT_LEFT, token.SHIFT_LEFT_ASSIGN)
			case '>':
				tkn = p.switch6(token.GREATER, token.GREATER_OR_EQUAL, '>', token.SHIFT_RIGHT, token.SHIFT_RIGHT_ASSIGN, '>', token.UNSIGNED_SHIFT_RIGHT, token.UNSIGNED_SHIFT_RIGHT_ASSIGN)
			case '=':
				tkn = p.switch2(token.ASSIGN, token.EQUAL)
				if tkn == token.EQUAL && p.chr == '=' {
					p.read()
					tkn = token.STRICT_EQUAL
				}
			case '!':
				tkn = p.switch2(token.NOT, token.NOT_EQUAL)
				if tkn == token.NOT_EQUAL && p.chr == '=' {
					p.read()
					tkn = token.STRICT_NOT_EQUAL
				}
			case '&':
				if p.chr == '^' {
					p.read()
					tkn = p.switch2(token.AND_NOT, token.AND_NOT_ASSIGN)
				} else {
					tkn = p.switch3(token.AND, token.AND_ASSIGN, '&', token.LOGICAL_AND)
				}
			case '|':
				tkn = p.switch3(token.OR, token.OR_ASSIGN, '|', token.LOGICAL_OR)
			case '~':
				tkn = token.BITWISE_NOT
			case '?':
				tkn = token.QUESTION_MARK
			case '"', '\'':
				insertSemicolon = true
				tkn = token.STRING
				var err error
				literal, err = p.scanString(p.chrOffset - 1)
				if err != nil {
					tkn = token.ILLEGAL
				}
			default:
				p.errorUnexpected(idx, chr)
				tkn = token.ILLEGAL
			}
		}
		p.insertSemicolon = insertSemicolon
		return tkn, literal, idx
	}
}

func (p *parser) switch2(tkn0, tkn1 token.Token) token.Token {
	if p.chr == '=' {
		p.read()
		return tkn1
	}
	return tkn0
}

func (p *parser) switch3(tkn0, tkn1 token.Token, chr2 rune, tkn2 token.Token) token.Token {
	if p.chr == '=' {
		p.read()
		return tkn1
	}
	if p.chr == chr2 {
		p.read()
		return tkn2
	}
	return tkn0
}

func (p *parser) switch4(tkn0, tkn1 token.Token, chr2 rune, tkn2, tkn3 token.Token) token.Token {
	if p.chr == '=' {
		p.read()
		return tkn1
	}
	if p.chr == chr2 {
		p.read()
		if p.chr == '=' {
			p.read()
			return tkn3
		}
		return tkn2
	}
	return tkn0
}

func (p *parser) switch6(tkn0, tkn1 token.Token, chr2 rune, tkn2, tkn3 token.Token, chr3 rune, tkn4, tkn5 token.Token) token.Token {
	if p.chr == '=' {
		p.read()
		return tkn1
	}
	if p.chr == chr2 {
		p.read()
		if p.chr == '=' {
			p.read()
			return tkn3
		}
		if p.chr == chr3 {
			p.read()
			if p.chr == '=' {
				p.read()
				return tkn5
			}
			return tkn4
		}
		return tkn2
	}
	return tkn0
}

func (p *parser) chrAt(index int) chr { //nolint:unused
	value, width := utf8.DecodeRuneInString(p.str[index:])
	return chr{
		value: value,
		width: width,
	}
}

func (p *parser) peek() rune {
	if p.offset+1 < p.length {
		return rune(p.str[p.offset+1])
	}
	return -1
}

func (p *parser) read() {
	if p.offset < p.length {
		p.chrOffset = p.offset
		chr, width := rune(p.str[p.offset]), 1
		if chr >= utf8.RuneSelf { // !ASCII
			chr, width = utf8.DecodeRuneInString(p.str[p.offset:])
			if chr == utf8.RuneError && width == 1 {
				p.error(p.chrOffset, "Invalid UTF-8 character")
			}
		}
		p.offset += width
		p.chr = chr
	} else {
		p.chrOffset = p.length
		p.chr = -1 // EOF
	}
}

// This is here since the functions are so similar.
func (p *regExpParser) read() {
	if p.offset < p.length {
		p.chrOffset = p.offset
		chr, width := rune(p.str[p.offset]), 1
		if chr >= utf8.RuneSelf { // !ASCII
			chr, width = utf8.DecodeRuneInString(p.str[p.offset:])
			if chr == utf8.RuneError && width == 1 {
				p.error(p.chrOffset, "Invalid UTF-8 character")
			}
		}
		p.offset += width
		p.chr = chr
	} else {
		p.chrOffset = p.length
		p.chr = -1 // EOF
	}
}

func (p *parser) readSingleLineComment() []rune {
	var result []rune
	for p.chr != -1 {
		p.read()
		if isLineTerminator(p.chr) {
			return result
		}
		result = append(result, p.chr)
	}

	// Get rid of the trailing -1
	return result[:len(result)-1]
}

func (p *parser) readMultiLineComment() []rune {
	var result []rune
	p.read()
	for p.chr >= 0 {
		chr := p.chr
		p.read()
		if chr == '*' && p.chr == '/' {
			p.read()
			return result
		}

		result = append(result, chr)
	}

	p.errorUnexpected(0, p.chr)

	return result
}

func (p *parser) skipSingleLineComment() {
	for p.chr != -1 {
		p.read()
		if isLineTerminator(p.chr) {
			return
		}
	}
}

func (p *parser) skipMultiLineComment() {
	p.read()
	for p.chr >= 0 {
		chr := p.chr
		p.read()
		if chr == '*' && p.chr == '/' {
			p.read()
			return
		}
	}

	p.errorUnexpected(0, p.chr)
}

func (p *parser) skipWhiteSpace() {
	for {
		switch p.chr {
		case ' ', '\t', '\f', '\v', '\u00a0', '\ufeff':
			p.read()
			continue
		case '\r':
			if p.peek() == '\n' {
				p.comments.AtLineBreak()
				p.read()
			}
			fallthrough
		case '\u2028', '\u2029', '\n':
			if p.insertSemicolon {
				return
			}
			p.comments.AtLineBreak()
			p.read()
			continue
		}
		if p.chr >= utf8.RuneSelf {
			if unicode.IsSpace(p.chr) {
				p.read()
				continue
			}
		}
		break
	}
}

func (p *parser) scanMantissa(base int) {
	for digitValue(p.chr) < base {
		p.read()
	}
}

func (p *parser) scanEscape(quote rune) {
	var length, base uint32
	switch p.chr {
	//    Octal:
	//    length, base, limit = 3, 8, 255
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"', '\'', '0':
		p.read()
		return
	case '\r', '\n', '\u2028', '\u2029':
		p.scanNewline()
		return
	case 'x':
		p.read()
		length, base = 2, 16
	case 'u':
		p.read()
		length, base = 4, 16
	default:
		p.read() // Always make progress
		return
	}

	var value uint32
	for ; length > 0 && p.chr != quote && p.chr >= 0; length-- {
		digit := uint32(digitValue(p.chr))
		if digit >= base {
			break
		}
		value = value*base + digit
		p.read()
	}
}

func (p *parser) scanString(offset int) (string, error) {
	// " ' /
	quote := rune(p.str[offset])

	for p.chr != quote {
		chr := p.chr
		if chr == '\n' || chr == '\r' || chr == '\u2028' || chr == '\u2029' || chr < 0 {
			goto newline
		}
		p.read()
		switch {
		case chr == '\\':
			if quote == '/' {
				if p.chr == '\n' || p.chr == '\r' || p.chr == '\u2028' || p.chr == '\u2029' || p.chr < 0 {
					goto newline
				}
				p.read()
			} else {
				p.scanEscape(quote)
			}
		case chr == '[' && quote == '/':
			// Allow a slash (/) in a bracket character class ([...])
			// TODO Fix this, this is hacky...
			quote = -1
		case chr == ']' && quote == -1:
			quote = '/'
		}
	}

	// " ' /
	p.read()

	return p.str[offset:p.chrOffset], nil

newline:
	p.scanNewline()
	err := "String not terminated"
	if quote == '/' {
		err = "Invalid regular expression: missing /"
		p.error(p.idxOf(offset), err)
	}
	return "", errors.New(err)
}

func (p *parser) scanNewline() {
	if p.chr == '\r' {
		p.read()
		if p.chr != '\n' {
			return
		}
	}
	p.read()
}

func hex2decimal(chr byte) (rune, bool) {
	r := rune(chr)
	switch {
	case '0' <= r && r <= '9':
		return r - '0', true
	case 'a' <= r && r <= 'f':
		return r - 'a' + 10, true
	case 'A' <= r && r <= 'F':
		return r - 'A' + 10, true
	default:
		return 0, false
	}
}

func parseNumberLiteral(literal string) (value interface{}, err error) { //nolint:nonamedreturns
	// TODO Is Uint okay? What about -MAX_UINT
	value, err = strconv.ParseInt(literal, 0, 64)
	if err == nil {
		return value, nil
	}

	parseIntErr := err // Save this first error, just in case

	value, err = strconv.ParseFloat(literal, 64)
	if err == nil {
		return value, nil
	} else if errors.Is(err, strconv.ErrRange) {
		// Infinity, etc.
		return value, nil
	}

	// TODO(steve): Fix as this is assigning to err so we know the type.
	// Need to understand what this was trying to do?
	err = parseIntErr

	if errors.Is(err, strconv.ErrRange) {
		if len(literal) > 2 && literal[0] == '0' && (literal[1] == 'X' || literal[1] == 'x') {
			// Could just be a very large number (e.g. 0x8000000000000000)
			var value float64
			literal = literal[2:]
			for _, chr := range literal {
				digit := digitValue(chr)
				if digit >= 16 {
					return nil, fmt.Errorf("illegal numeric literal: %v (>= 16)", digit)
				}
				value = value*16 + float64(digit)
			}
			return value, nil
		}
	}

	return nil, errors.New("illegal numeric literal")
}

func parseStringLiteral(literal string) (string, error) {
	// Best case scenario...
	if literal == "" {
		return "", nil
	}

	// Slightly less-best case scenario...
	if !strings.ContainsRune(literal, '\\') {
		return literal, nil
	}

	str := literal
	buffer := bytes.NewBuffer(make([]byte, 0, 3*len(literal)/2))

	for len(str) > 0 {
		switch chr := str[0]; {
		// We do not explicitly handle the case of the quote
		// value, which can be: " ' /
		// This assumes we're already passed a partially well-formed literal
		case chr >= utf8.RuneSelf:
			chr, size := utf8.DecodeRuneInString(str)
			buffer.WriteRune(chr)
			str = str[size:]
			continue
		case chr != '\\':
			buffer.WriteByte(chr)
			str = str[1:]
			continue
		}

		if len(str) <= 1 {
			panic("len(str) <= 1")
		}
		chr := str[1]
		var value rune
		if chr >= utf8.RuneSelf {
			str = str[1:]
			var size int
			value, size = utf8.DecodeRuneInString(str)
			str = str[size:] // \ + <character>
		} else {
			str = str[2:] // \<character>
			switch chr {
			case 'b':
				value = '\b'
			case 'f':
				value = '\f'
			case 'n':
				value = '\n'
			case 'r':
				value = '\r'
			case 't':
				value = '\t'
			case 'v':
				value = '\v'
			case 'x', 'u':
				size := 0
				switch chr {
				case 'x':
					size = 2
				case 'u':
					size = 4
				}
				if len(str) < size {
					return "", fmt.Errorf("invalid escape: \\%s: len(%q) != %d", string(chr), str, size)
				}
				for j := range size {
					decimal, ok := hex2decimal(str[j])
					if !ok {
						return "", fmt.Errorf("invalid escape: \\%s: %q", string(chr), str[:size])
					}
					value = value<<4 | decimal
				}
				str = str[size:]
				if chr == 'x' {
					break
				}
				if value > utf8.MaxRune {
					panic("value > utf8.MaxRune")
				}
			case '0':
				if len(str) == 0 || '0' > str[0] || str[0] > '7' {
					value = 0
					break
				}
				fallthrough
			case '1', '2', '3', '4', '5', '6', '7':
				// TODO strict
				value = rune(chr) - '0'
				j := 0
				for ; j < 2; j++ {
					if len(str) < j+1 {
						break
					}

					if ch := str[j]; '0' > ch || ch > '7' {
						break
					}
					decimal := rune(str[j]) - '0'
					value = (value << 3) | decimal
				}
				str = str[j:]
			case '\\':
				value = '\\'
			case '\'', '"':
				value = rune(chr)
			case '\r':
				if len(str) > 0 {
					if str[0] == '\n' {
						str = str[1:]
					}
				}
				fallthrough
			case '\n':
				continue
			default:
				value = rune(chr)
			}
		}
		buffer.WriteRune(value)
	}

	return buffer.String(), nil
}

func (p *parser) scanNumericLiteral(decimalPoint bool) (token.Token, string) {
	offset := p.chrOffset
	tkn := token.NUMBER

	if decimalPoint {
		offset--
		p.scanMantissa(10)
		goto exponent
	}

	if p.chr == '0' {
		chrOffset := p.chrOffset
		p.read()
		switch p.chr {
		case 'x', 'X':
			// Hexadecimal
			p.read()
			if isDigit(p.chr, 16) {
				p.read()
			} else {
				return token.ILLEGAL, p.str[chrOffset:p.chrOffset]
			}
			p.scanMantissa(16)

			if p.chrOffset-chrOffset <= 2 {
				// Only "0x" or "0X"
				p.error(0, "Illegal hexadecimal number")
			}

			goto hexadecimal
		case '.':
			// Float
			goto float
		default:
			// Octal, Float
			if p.chr == 'e' || p.chr == 'E' {
				goto exponent
			}
			p.scanMantissa(8)
			if p.chr == '8' || p.chr == '9' {
				return token.ILLEGAL, p.str[chrOffset:p.chrOffset]
			}
			goto octal
		}
	}

	p.scanMantissa(10)

float:
	if p.chr == '.' {
		p.read()
		p.scanMantissa(10)
	}

exponent:
	if p.chr == 'e' || p.chr == 'E' {
		p.read()
		if p.chr == '-' || p.chr == '+' {
			p.read()
		}
		if isDecimalDigit(p.chr) {
			p.read()
			p.scanMantissa(10)
		} else {
			return token.ILLEGAL, p.str[offset:p.chrOffset]
		}
	}

hexadecimal:
octal:
	if isIdentifierStart(p.chr) || isDecimalDigit(p.chr) {
		return token.ILLEGAL, p.str[offset:p.chrOffset]
	}

	return tkn, p.str[offset:p.chrOffset]
}

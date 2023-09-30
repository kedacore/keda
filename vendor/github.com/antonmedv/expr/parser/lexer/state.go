package lexer

import (
	"strings"

	"github.com/antonmedv/expr/parser/utils"
)

type stateFn func(*lexer) stateFn

func root(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		l.emitEOF()
		return nil
	case utils.IsSpace(r):
		l.ignore()
		return root
	case r == '\'' || r == '"':
		l.scanString(r)
		str, err := unescape(l.word())
		if err != nil {
			l.error("%v", err)
		}
		l.emitValue(String, str)
	case '0' <= r && r <= '9':
		l.backup()
		return number
	case r == '?':
		return questionMark
	case r == '/':
		return slash
	case r == '#':
		return pointer
	case r == '|':
		l.accept("|")
		l.emit(Operator)
	case strings.ContainsRune("([{", r):
		l.emit(Bracket)
	case strings.ContainsRune(")]}", r):
		l.emit(Bracket)
	case strings.ContainsRune(",:;%+-^", r): // single rune operator
		l.emit(Operator)
	case strings.ContainsRune("&!=*<>", r): // possible double rune operator
		l.accept("&=*")
		l.emit(Operator)
	case r == '.':
		l.backup()
		return dot
	case utils.IsAlphaNumeric(r):
		l.backup()
		return identifier
	default:
		return l.error("unrecognized character: %#U", r)
	}
	return root
}

func number(l *lexer) stateFn {
	if !l.scanNumber() {
		return l.error("bad number syntax: %q", l.word())
	}
	l.emit(Number)
	return root
}

func (l *lexer) scanNumber() bool {
	digits := "0123456789_"
	// Is it hex?
	if l.accept("0") {
		// Note: Leading 0 does not mean octal in floats.
		if l.accept("xX") {
			digits = "0123456789abcdefABCDEF_"
		} else if l.accept("oO") {
			digits = "01234567_"
		} else if l.accept("bB") {
			digits = "01_"
		}
	}
	l.acceptRun(digits)
	loc, prev, end := l.loc, l.prev, l.end
	if l.accept(".") {
		// Lookup for .. operator: if after dot there is another dot (1..2), it maybe a range operator.
		if l.peek() == '.' {
			// We can't backup() here, as it would require two backups,
			// and backup() func supports only one for now. So, save and
			// restore it here.
			l.loc, l.prev, l.end = loc, prev, end
			return true
		}
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun(digits)
	}
	// Next thing mustn't be alphanumeric.
	if utils.IsAlphaNumeric(l.peek()) {
		l.next()
		return false
	}
	return true
}

func dot(l *lexer) stateFn {
	l.next()
	if l.accept("0123456789") {
		l.backup()
		return number
	}
	l.accept(".")
	l.emit(Operator)
	return root
}

func identifier(l *lexer) stateFn {
loop:
	for {
		switch r := l.next(); {
		case utils.IsAlphaNumeric(r):
			// absorb
		default:
			l.backup()
			switch l.word() {
			case "not":
				return not
			case "in", "or", "and", "matches", "contains", "startsWith", "endsWith":
				l.emit(Operator)
			case "let":
				l.emit(Operator)
			default:
				l.emit(Identifier)
			}
			break loop
		}
	}
	return root
}

func not(l *lexer) stateFn {
	l.emit(Operator)

	l.skipSpaces()

	pos, loc, prev := l.end, l.loc, l.prev

	// Get the next word.
	for {
		r := l.next()
		if utils.IsAlphaNumeric(r) {
			// absorb
		} else {
			l.backup()
			break
		}
	}

	switch l.word() {
	case "in", "matches", "contains", "startsWith", "endsWith":
		l.emit(Operator)
	default:
		l.end, l.loc, l.prev = pos, loc, prev
	}
	return root
}

func questionMark(l *lexer) stateFn {
	l.accept(".?")
	l.emit(Operator)
	return root
}

func slash(l *lexer) stateFn {
	if l.accept("/") {
		return singleLineComment
	}
	if l.accept("*") {
		return multiLineComment
	}
	l.emit(Operator)
	return root
}

func singleLineComment(l *lexer) stateFn {
	for {
		r := l.next()
		if r == eof || r == '\n' {
			break
		}
	}
	l.ignore()
	return root
}

func multiLineComment(l *lexer) stateFn {
	for {
		r := l.next()
		if r == eof {
			return l.error("unclosed comment")
		}
		if r == '*' && l.accept("/") {
			break
		}
	}
	l.ignore()
	return root
}

func pointer(l *lexer) stateFn {
	l.accept("#")
	l.emit(Operator)
	for {
		switch r := l.next(); {
		case utils.IsAlphaNumeric(r): // absorb
		default:
			l.backup()
			if l.word() != "" {
				l.emit(Identifier)
			}
			return root
		}
	}
}

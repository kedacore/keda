package lexer

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/internal/ring"
)

const ringChunkSize = 10

// Lex will buffer and return the tokens of a disposable *[Lexer].
func Lex(source file.Source) ([]Token, error) {
	tokens := make([]Token, 0, ringChunkSize)
	l := New()
	l.Reset(source)
	for {
		t, err := l.Next()
		switch err {
		case nil:
			tokens = append(tokens, t)
		case io.EOF:
			return tokens, nil
		default:
			return nil, err
		}
	}
}

// New returns a reusable lexer.
func New() *Lexer {
	return &Lexer{
		tokens: ring.New[Token](ringChunkSize),
	}
}

type Lexer struct {
	state      stateFn
	source     file.Source
	tokens     *ring.Ring[Token]
	err        *file.Error
	start, end struct {
		byte, rune int
	}
	eof bool
}

func (l *Lexer) Reset(source file.Source) {
	l.source = source
	l.tokens.Reset()
	l.state = root
}

func (l *Lexer) Next() (Token, error) {
	for l.state != nil && l.err == nil && l.tokens.Len() == 0 {
		l.state = l.state(l)
	}
	if l.err != nil {
		return Token{}, l.err.Bind(l.source)
	}
	if t, ok := l.tokens.Dequeue(); ok {
		return t, nil
	}
	return Token{}, io.EOF
}

const eof rune = -1

func (l *Lexer) commit() {
	l.start = l.end
}

func (l *Lexer) next() rune {
	if l.end.byte >= len(l.source.String()) {
		l.eof = true
		return eof
	}
	r, sz := utf8.DecodeRuneInString(l.source.String()[l.end.byte:])
	l.end.rune++
	l.end.byte += sz
	return r
}

func (l *Lexer) peek() rune {
	if l.end.byte < len(l.source.String()) {
		r, _ := utf8.DecodeRuneInString(l.source.String()[l.end.byte:])
		return r
	}
	return eof
}

func (l *Lexer) peekByte() (byte, bool) {
	if l.end.byte >= 0 && l.end.byte < len(l.source.String()) {
		return l.source.String()[l.end.byte], true
	}
	return 0, false
}

func (l *Lexer) backup() {
	if l.eof {
		l.eof = false
	} else if l.end.rune > 0 {
		_, sz := utf8.DecodeLastRuneInString(l.source.String()[:l.end.byte])
		l.end.byte -= sz
		l.end.rune--
	}
}

func (l *Lexer) emit(t Kind) {
	l.emitValue(t, l.word())
}

func (l *Lexer) emitValue(t Kind, value string) {
	l.tokens.Enqueue(Token{
		Location: file.Location{From: l.start.rune, To: l.end.rune},
		Kind:     t,
		Value:    value,
	})
	l.commit()
}

func (l *Lexer) emitEOF() {
	from := l.end.rune - 1
	if from < 0 {
		from = 0
	}
	to := l.end.rune - 0
	if to < 0 {
		to = 0
	}
	l.tokens.Enqueue(Token{
		Location: file.Location{From: from, To: to},
		Kind:     EOF,
	})
	l.commit()
}

func (l *Lexer) skip() {
	l.commit()
}

func (l *Lexer) word() string {
	return l.source.String()[l.start.byte:l.end.byte]
}

func (l *Lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.peek()) {
		l.next()
		return true
	}
	return false
}

func (l *Lexer) acceptRun(valid string) {
	for l.accept(valid) {
	}
}

func (l *Lexer) skipSpaces() {
	l.acceptRun(" ")
	l.skip()
}

func (l *Lexer) error(format string, args ...any) stateFn {
	if l.err == nil { // show first error
		end := l.end.rune
		if l.eof {
			end++
		}
		l.err = &file.Error{
			Location: file.Location{
				From: end - 1,
				To:   end,
			},
			Message: fmt.Sprintf(format, args...),
		}
	}
	return nil
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= lower(ch) && lower(ch) <= 'f':
		return int(lower(ch) - 'a' + 10)
	}
	return 16 // larger than any legal digit val
}

func lower(ch rune) rune { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter

func (l *Lexer) scanDigits(ch rune, base, n int) rune {
	for n > 0 && digitVal(ch) < base {
		ch = l.next()
		n--
	}
	if n > 0 {
		l.error("invalid char escape")
	}
	return ch
}

func (l *Lexer) scanEscape(quote rune) rune {
	ch := l.next() // read character after '/'
	switch ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		// nothing to do
		ch = l.next()
	case '0', '1', '2', '3', '4', '5', '6', '7':
		ch = l.scanDigits(ch, 8, 3)
	case 'x':
		ch = l.scanDigits(l.next(), 16, 2)
	case 'u':
		ch = l.scanDigits(l.next(), 16, 4)
	case 'U':
		ch = l.scanDigits(l.next(), 16, 8)
	default:
		l.error("invalid char escape")
	}
	return ch
}

func (l *Lexer) scanString(quote rune) (n int) {
	ch := l.next() // read character after quote
	for ch != quote {
		if ch == '\n' || ch == eof {
			l.error("literal not terminated")
			return
		}
		if ch == '\\' {
			ch = l.scanEscape(quote)
		} else {
			ch = l.next()
		}
		n++
	}
	return
}

func (l *Lexer) scanRawString(quote rune) (n int) {
	ch := l.next() // read character after back tick
	for ch != quote {
		if ch == eof {
			l.error("literal not terminated")
			return
		}
		ch = l.next()
		n++
	}
	l.emitValue(String, l.source.String()[l.start.byte+1:l.end.byte-1])
	return
}

// Package token defines constants representing the lexical tokens of JavaScript (ECMA5).
package token

import (
	"strconv"
)

// Token is the set of lexical tokens in JavaScript (ECMA5).
type Token int

// String returns the string corresponding to the token.
// For operators, delimiters, and keywords the string is the actual
// token string (e.g., for the token PLUS, the String() is
// "+"). For all other tokens the string corresponds to the token
// name (e.g. for the token IDENTIFIER, the string is "IDENTIFIER").
func (tkn Token) String() string {
	switch {
	case tkn == 0:
		return "UNKNOWN"
	case tkn < Token(len(token2string)):
		return token2string[tkn]
	default:
		return "token(" + strconv.Itoa(int(tkn)) + ")"
	}
}

type keyword struct {
	token         Token
	futureKeyword bool
	strict        bool
}

// IsKeyword returns the keyword token if literal is a keyword, a KEYWORD token
// if the literal is a future keyword (const, let, class, super, ...), or 0 if the literal is not a keyword.
//
// If the literal is a keyword, IsKeyword returns a second value indicating if the literal
// is considered a future keyword in strict-mode only.
//
// 7.6.1.2 Future Reserved Words:
//
//	const
//	class
//	enum
//	export
//	extends
//	import
//	super
//
// 7.6.1.2 Future Reserved Words (strict):
//
//	implements
//	interface
//	let
//	package
//	private
//	protected
//	public
//	static
func IsKeyword(literal string) (Token, bool) {
	if kw, exists := keywordTable[literal]; exists {
		if kw.futureKeyword {
			return KEYWORD, kw.strict
		}
		return kw.token, false
	}
	return 0, false
}

package parser

import (
	"fmt"
	"sort"

	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

const (
	errUnexpectedToken      = "Unexpected token %v"
	errUnexpectedEndOfInput = "Unexpected end of input"
)

//    UnexpectedNumber:  'Unexpected number',
//    UnexpectedString:  'Unexpected string',
//    UnexpectedIdentifier:  'Unexpected identifier',
//    UnexpectedReserved:  'Unexpected reserved word',
//    NewlineAfterThrow:  'Illegal newline after throw',
//    InvalidRegExp: 'Invalid regular expression',
//    UnterminatedRegExp:  'Invalid regular expression: missing /',
//    InvalidLHSInAssignment:  'invalid left-hand side in assignment',
//    InvalidLHSInForIn:  'Invalid left-hand side in for-in',
//    MultipleDefaultsInSwitch: 'More than one default clause in switch statement',
//    NoCatchOrFinally:  'Missing catch or finally after try',
//    UnknownLabel: 'Undefined label \'%0\'',
//    Redeclaration: '%0 \'%1\' has already been declared',
//    IllegalContinue: 'Illegal continue statement',
//    IllegalBreak: 'Illegal break statement',
//    IllegalReturn: 'Illegal return statement',
//    StrictModeWith:  'Strict mode code may not include a with statement',
//    StrictCatchVariable:  'Catch variable may not be eval or arguments in strict mode',
//    StrictVarName:  'Variable name may not be eval or arguments in strict mode',
//    StrictParamName:  'Parameter name eval or arguments is not allowed in strict mode',
//    StrictParamDupe: 'Strict mode function may not have duplicate parameter names',
//    StrictFunctionName:  'Function name may not be eval or arguments in strict mode',
//    StrictOctalLiteral:  'Octal literals are not allowed in strict mode.',
//    StrictDelete:  'Delete of an unqualified identifier in strict mode.',
//    StrictDuplicateProperty:  'Duplicate data property in object literal not allowed in strict mode',
//    AccessorDataProperty:  'Object literal may not have data and accessor property with the same name',
//    AccessorGetSet:  'Object literal may not have multiple get/set accessors with the same name',
//    StrictLHSAssignment:  'Assignment to eval or arguments is not allowed in strict mode',
//    StrictLHSPostfix:  'Postfix increment/decrement may not have eval or arguments operand in strict mode',
//    StrictLHSPrefix:  'Prefix increment/decrement may not have eval or arguments operand in strict mode',
//    StrictReservedWord:  'Use of future reserved word in strict mode'

// A SyntaxError is a description of an ECMAScript syntax error.

// An Error represents a parsing error. It includes the position where the error occurred and a message/description.
type Error struct {
	Message  string
	Position file.Position
}

// FIXME Should this be "SyntaxError"?

func (e Error) Error() string {
	filename := e.Position.Filename
	if filename == "" {
		filename = "(anonymous)"
	}
	return fmt.Sprintf("%s: Line %d:%d %s",
		filename,
		e.Position.Line,
		e.Position.Column,
		e.Message,
	)
}

func (p *parser) error(place interface{}, msg string, msgValues ...interface{}) {
	var idx file.Idx
	switch place := place.(type) {
	case int:
		idx = p.idxOf(place)
	case file.Idx:
		if place == 0 {
			idx = p.idxOf(p.chrOffset)
		} else {
			idx = place
		}
	default:
		panic(fmt.Errorf("error(%T, ...)", place))
	}

	position := p.position(idx)
	msg = fmt.Sprintf(msg, msgValues...)
	p.errors.Add(position, msg)
}

func (p *parser) errorUnexpected(idx file.Idx, chr rune) {
	if chr == -1 {
		p.error(idx, errUnexpectedEndOfInput)
		return
	}
	p.error(idx, errUnexpectedToken, token.ILLEGAL)
}

func (p *parser) errorUnexpectedToken(tkn token.Token) {
	if tkn == token.EOF {
		p.error(file.Idx(0), errUnexpectedEndOfInput)
		return
	}
	value := tkn.String()
	switch tkn {
	case token.BOOLEAN, token.NULL:
		p.error(p.idx, errUnexpectedToken, p.literal)
	case token.IDENTIFIER:
		p.error(p.idx, "Unexpected identifier")
	case token.KEYWORD:
		// TODO Might be a future reserved word
		p.error(p.idx, "Unexpected reserved word")
	case token.NUMBER:
		p.error(p.idx, "Unexpected number")
	case token.STRING:
		p.error(p.idx, "Unexpected string")
	default:
		p.error(p.idx, errUnexpectedToken, value)
	}
}

// ErrorList is a list of *Errors.
type ErrorList []*Error //nolint:errname

// Add adds an Error with given position and message to an ErrorList.
func (el *ErrorList) Add(position file.Position, msg string) {
	*el = append(*el, &Error{Position: position, Message: msg})
}

// Reset resets an ErrorList to no errors.
func (el *ErrorList) Reset() {
	*el = (*el)[0:0]
}

// Len implement sort.Interface.
func (el *ErrorList) Len() int {
	return len(*el)
}

// Swap implement sort.Interface.
func (el *ErrorList) Swap(i, j int) {
	(*el)[i], (*el)[j] = (*el)[j], (*el)[i]
}

// Less implement sort.Interface.
func (el *ErrorList) Less(i, j int) bool {
	x := (*el)[i].Position
	y := (*el)[j].Position
	if x.Filename < y.Filename {
		return true
	}
	if x.Filename == y.Filename {
		if x.Line < y.Line {
			return true
		}
		if x.Line == y.Line {
			return x.Column < y.Column
		}
	}
	return false
}

// Sort sorts el.
func (el *ErrorList) Sort() {
	sort.Sort(el)
}

// Error implements the Error interface.
func (el *ErrorList) Error() string {
	switch len(*el) {
	case 0:
		return "no errors"
	case 1:
		return (*el)[0].Error()
	default:
		return fmt.Sprintf("%s (and %d more errors)", (*el)[0].Error(), len(*el)-1)
	}
}

// Err returns an error equivalent to this ErrorList.
// If the list is empty, Err returns nil.
func (el *ErrorList) Err() error {
	if len(*el) == 0 {
		return nil
	}
	return el
}

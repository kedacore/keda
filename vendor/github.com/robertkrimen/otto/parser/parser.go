/*
Package parser implements a parser for JavaScript.

	import (
	    "github.com/robertkrimen/otto/parser"
	)

Parse and return an AST

	filename := "" // A filename is optional
	src := `
	    // Sample xyzzy example
	    (function(){
	        if (3.14159 > 0) {
	            console.log("Hello, World.");
	            return;
	        }

	        var xyzzy = NaN;
	        console.log("Nothing happens.");
	        return xyzzy;
	    })();
	`

	// Parse some JavaScript, yielding a *ast.Program and/or an ErrorList
	program, err := parser.ParseFile(nil, filename, src, 0)

# Warning

The parser and AST interfaces are still works-in-progress (particularly where
node types are concerned) and may change in the future.
*/
package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
	"gopkg.in/sourcemap.v1"
)

// A Mode value is a set of flags (or 0). They control optional parser functionality.
type Mode uint

const (
	// IgnoreRegExpErrors ignores RegExp compatibility errors (allow backtracking).
	IgnoreRegExpErrors Mode = 1 << iota

	// StoreComments stores the comments from source to the comments map.
	StoreComments
)

type parser struct {
	comments *ast.Comments
	file     *file.File
	scope    *scope
	literal  string
	str      string
	errors   ErrorList
	recover  struct {
		idx   file.Idx
		count int
	}
	idx               file.Idx
	token             token.Token
	offset            int
	chrOffset         int
	mode              Mode
	base              int
	length            int
	chr               rune
	insertSemicolon   bool
	implicitSemicolon bool // Scratch when trying to seek to the next statement, etc.
}

// Parser is implemented by types which can parse JavaScript Code.
type Parser interface {
	Scan() (tkn token.Token, literal string, idx file.Idx)
}

func newParser(filename, src string, base int, sm *sourcemap.Consumer) *parser {
	return &parser{
		chr:      ' ', // This is set so we can start scanning by skipping whitespace
		str:      src,
		length:   len(src),
		base:     base,
		file:     file.NewFile(filename, src, base).WithSourceMap(sm),
		comments: ast.NewComments(),
	}
}

// NewParser returns a new Parser.
func NewParser(filename, src string) Parser {
	return newParser(filename, src, 1, nil)
}

// ReadSource reads code from src if not nil, otherwise reads from filename.
func ReadSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch src := src.(type) {
		case string:
			return []byte(src), nil
		case []byte:
			return src, nil
		case *bytes.Buffer:
			if src != nil {
				return src.Bytes(), nil
			}
		case io.Reader:
			var bfr bytes.Buffer
			if _, err := io.Copy(&bfr, src); err != nil {
				return nil, err
			}
			return bfr.Bytes(), nil
		default:
			return nil, fmt.Errorf("invalid src type %T", src)
		}
	}
	return os.ReadFile(filename) //nolint:gosec
}

// ReadSourceMap reads the source map from src if not nil, otherwise is a noop.
func ReadSourceMap(filename string, src interface{}) (*sourcemap.Consumer, error) {
	if src == nil {
		return nil, nil //nolint:nilnil
	}

	switch src := src.(type) {
	case string:
		return sourcemap.Parse(filename, []byte(src))
	case []byte:
		return sourcemap.Parse(filename, src)
	case *bytes.Buffer:
		return sourcemap.Parse(filename, src.Bytes())
	case io.Reader:
		var bfr bytes.Buffer
		if _, err := io.Copy(&bfr, src); err != nil {
			return nil, err
		}
		return sourcemap.Parse(filename, bfr.Bytes())
	case *sourcemap.Consumer:
		return src, nil
	default:
		return nil, fmt.Errorf("invalid sourcemap type %T", src)
	}
}

// ParseFileWithSourceMap parses the sourcemap returning the resulting Program.
func ParseFileWithSourceMap(fileSet *file.FileSet, filename string, javascriptSource, sourcemapSource interface{}, mode Mode) (*ast.Program, error) {
	src, err := ReadSource(filename, javascriptSource)
	if err != nil {
		return nil, err
	}

	if sourcemapSource == nil {
		lines := bytes.Split(src, []byte("\n"))
		lastLine := lines[len(lines)-1]
		if bytes.HasPrefix(lastLine, []byte("//# sourceMappingURL=data:application/json")) {
			bits := bytes.SplitN(lastLine, []byte(","), 2)
			if len(bits) == 2 {
				if d, errDecode := base64.StdEncoding.DecodeString(string(bits[1])); errDecode == nil {
					sourcemapSource = d
				}
			}
		}
	}

	sm, err := ReadSourceMap(filename, sourcemapSource)
	if err != nil {
		return nil, err
	}

	base := 1
	if fileSet != nil {
		base = fileSet.AddFile(filename, string(src))
	}

	p := newParser(filename, string(src), base, sm)
	p.mode = mode
	program, err := p.parse()
	program.Comments = p.comments.CommentMap

	return program, err
}

// ParseFile parses the source code of a single JavaScript/ECMAScript source file and returns
// the corresponding ast.Program node.
//
// If fileSet == nil, ParseFile parses source without a FileSet.
// If fileSet != nil, ParseFile first adds filename and src to fileSet.
//
// The filename argument is optional and is used for labelling errors, etc.
//
// src may be a string, a byte slice, a bytes.Buffer, or an io.Reader, but it MUST always be in UTF-8.
//
//	// Parse some JavaScript, yielding a *ast.Program and/or an ErrorList
//	program, err := parser.ParseFile(nil, "", `if (abc > 1) {}`, 0)
func ParseFile(fileSet *file.FileSet, filename string, src interface{}, mode Mode) (*ast.Program, error) {
	return ParseFileWithSourceMap(fileSet, filename, src, nil, mode)
}

// ParseFunction parses a given parameter list and body as a function and returns the
// corresponding ast.FunctionLiteral node.
//
// The parameter list, if any, should be a comma-separated list of identifiers.
func ParseFunction(parameterList, body string) (*ast.FunctionLiteral, error) {
	src := "(function(" + parameterList + ") {\n" + body + "\n})"

	p := newParser("", src, 1, nil)
	program, err := p.parse()
	if err != nil {
		return nil, err
	}

	return program.Body[0].(*ast.ExpressionStatement).Expression.(*ast.FunctionLiteral), nil
}

// Scan reads a single token from the source at the current offset, increments the offset and
// returns the token.Token token, a string literal representing the value of the token (if applicable)
// and it's current file.Idx index.
func (p *parser) Scan() (token.Token, string, file.Idx) {
	return p.scan()
}

func (p *parser) slice(idx0, idx1 file.Idx) string {
	from := int(idx0) - p.base
	to := int(idx1) - p.base
	if from >= 0 && to <= len(p.str) {
		return p.str[from:to]
	}

	return ""
}

func (p *parser) parse() (*ast.Program, error) {
	p.next()
	program := p.parseProgram()
	if false {
		p.errors.Sort()
	}

	if p.mode&StoreComments != 0 {
		p.comments.CommentMap.AddComments(program, p.comments.FetchAll(), ast.TRAILING)
	}

	return program, p.errors.Err()
}

func (p *parser) next() {
	p.token, p.literal, p.idx = p.scan()
}

func (p *parser) optionalSemicolon() {
	if p.token == token.SEMICOLON {
		p.next()
		return
	}

	if p.implicitSemicolon {
		p.implicitSemicolon = false
		return
	}

	if p.token != token.EOF && p.token != token.RIGHT_BRACE {
		p.expect(token.SEMICOLON)
	}
}

func (p *parser) semicolon() {
	if p.token != token.RIGHT_PARENTHESIS && p.token != token.RIGHT_BRACE {
		if p.implicitSemicolon {
			p.implicitSemicolon = false
			return
		}

		p.expect(token.SEMICOLON)
	}
}

func (p *parser) idxOf(offset int) file.Idx {
	return file.Idx(p.base + offset)
}

func (p *parser) expect(value token.Token) file.Idx {
	idx := p.idx
	if p.token != value {
		p.errorUnexpectedToken(p.token)
	}
	p.next()
	return idx
}

func lineCount(str string) (int, int) {
	line, last := 0, -1
	pair := false
	for index, chr := range str {
		switch chr {
		case '\r':
			line++
			last = index
			pair = true
			continue
		case '\n':
			if !pair {
				line++
			}
			last = index
		case '\u2028', '\u2029':
			line++
			last = index + 2
		}
		pair = false
	}
	return line, last
}

func (p *parser) position(idx file.Idx) file.Position {
	position := file.Position{}
	offset := int(idx) - p.base
	str := p.str[:offset]
	position.Filename = p.file.Name()
	line, last := lineCount(str)
	position.Line = 1 + line
	if last >= 0 {
		position.Column = offset - last
	} else {
		position.Column = 1 + len(str)
	}

	return position
}

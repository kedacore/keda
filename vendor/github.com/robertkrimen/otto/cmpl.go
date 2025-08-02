package otto

import (
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
)

type compiler struct {
	file    *file.File
	program *ast.Program
}

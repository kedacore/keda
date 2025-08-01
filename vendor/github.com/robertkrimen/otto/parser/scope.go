package parser

import (
	"github.com/robertkrimen/otto/ast"
)

type scope struct {
	outer           *scope
	declarationList []ast.Declaration
	labels          []string
	allowIn         bool
	inIteration     bool
	inSwitch        bool
	inFunction      bool
}

func (p *parser) openScope() {
	p.scope = &scope{
		outer:   p.scope,
		allowIn: true,
	}
}

func (p *parser) closeScope() {
	p.scope = p.scope.outer
}

func (p *scope) declare(declaration ast.Declaration) {
	p.declarationList = append(p.declarationList, declaration)
}

func (p *scope) hasLabel(name string) bool {
	for _, label := range p.labels {
		if label == name {
			return true
		}
	}
	if p.outer != nil && !p.inFunction {
		// Crossing a function boundary to look for a label is verboten
		return p.outer.hasLabel(name)
	}
	return false
}

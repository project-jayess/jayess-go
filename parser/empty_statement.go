package parser

import (
	"jayess-go/ast"
)

func (p *Parser) parseEmptyStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	return &ast.EmptyStatement{BaseNode: baseFrom(start)}, nil
}

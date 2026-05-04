package parser

import (
	"jayess-go/ast"
)

func (p *Parser) parseDebuggerStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.DebuggerStatement{BaseNode: baseFrom(start)}, nil
}

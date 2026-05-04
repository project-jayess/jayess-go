package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseWhileStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	if err := p.expect(lexer.TokenLParen); err != nil {
		return nil, err
	}
	condition, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRParen); err != nil {
		return nil, err
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStatement{BaseNode: baseFrom(start), Condition: condition, Body: body}, nil
}

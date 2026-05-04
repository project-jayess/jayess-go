package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseDoWhileStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenWhile); err != nil {
		return nil, err
	}
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
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.DoWhileStatement{BaseNode: baseFrom(start), Body: body, Condition: condition}, nil
}

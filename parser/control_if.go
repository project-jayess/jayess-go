package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseIfStatement() (ast.Statement, error) {
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
	consequence, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	alternative, err := p.parseOptionalElse()
	if err != nil {
		return nil, err
	}
	return &ast.IfStatement{
		BaseNode:    baseFrom(start),
		Condition:   condition,
		Consequence: consequence,
		Alternative: alternative,
	}, nil
}

func (p *Parser) parseOptionalElse() ([]ast.Statement, error) {
	if !p.match(lexer.TokenElse) {
		return nil, nil
	}
	if p.current.Type == lexer.TokenIf {
		statement, err := p.parseIfStatement()
		if err != nil {
			return nil, err
		}
		return []ast.Statement{statement}, nil
	}
	return p.parseBlockStatements()
}

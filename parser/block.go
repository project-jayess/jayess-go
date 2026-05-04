package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseBlockStatement() (*ast.BlockStatement, error) {
	start := p.current
	statements, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.BlockStatement{BaseNode: baseFrom(start), Statements: statements}, nil
}

func (p *Parser) parseBlockStatements() ([]ast.Statement, error) {
	if err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	statements := []ast.Statement{}
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		statement, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}
	if err := p.expect(lexer.TokenRBrace); err != nil {
		return nil, err
	}
	return statements, nil
}

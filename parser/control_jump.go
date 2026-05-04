package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseBreakStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	label := p.consumeSameLineLabel(start)
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.BreakStatement{BaseNode: baseFrom(start), Label: label}, nil
}

func (p *Parser) parseContinueStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	label := p.consumeSameLineLabel(start)
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ContinueStatement{BaseNode: baseFrom(start), Label: label}, nil
}

func (p *Parser) consumeSameLineLabel(keyword lexer.Token) string {
	if p.current.Type != lexer.TokenIdent || p.current.Line > keyword.Line {
		return ""
	}
	label := p.current.Literal
	p.advance()
	return label
}

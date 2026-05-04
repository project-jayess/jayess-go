package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isLabeledStatementStart() bool {
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenColon
}

func (p *Parser) parseLabeledStatement() (ast.Statement, error) {
	start := p.current
	label := p.current.Literal
	p.advance()
	if err := p.expect(lexer.TokenColon); err != nil {
		return nil, err
	}
	statement, err := p.ParseStatement()
	if err != nil {
		return nil, err
	}
	return &ast.LabeledStatement{BaseNode: baseFrom(start), Label: label, Statement: statement}, nil
}

package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseReturnStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	if p.current.Type == lexer.TokenSemicolon {
		p.advance()
		return &ast.ReturnStatement{BaseNode: baseFrom(start)}, nil
	}
	if p.current.Type == lexer.TokenEOF || p.current.Type == lexer.TokenRBrace || p.current.Line > start.Line {
		return &ast.ReturnStatement{BaseNode: baseFrom(start)}, nil
	}
	value, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ReturnStatement{BaseNode: baseFrom(start), Value: value}, nil
}

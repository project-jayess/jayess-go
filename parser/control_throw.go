package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseThrowStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	if p.current.Type == lexer.TokenSemicolon || p.current.Type == lexer.TokenRBrace || p.current.Type == lexer.TokenEOF || p.current.Line > start.Line {
		return nil, errorAtToken(p.current, "throw requires an expression on the same line")
	}
	value, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ThrowStatement{BaseNode: baseFrom(start), Value: value}, nil
}

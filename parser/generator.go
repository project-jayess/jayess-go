package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseYieldExpression(token lexer.Token) (ast.Expression, error) {
	p.advance()
	if p.current.Type == lexer.TokenSemicolon || p.current.Type == lexer.TokenEOF || p.current.Type == lexer.TokenRBrace || p.current.Line > token.Line {
		return &ast.YieldExpression{BaseNode: baseFrom(token)}, nil
	}
	delegate := p.match(lexer.TokenStar)
	value, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return &ast.YieldExpression{BaseNode: baseFrom(token), Value: value, Delegate: delegate}, nil
}

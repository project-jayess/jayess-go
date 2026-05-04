package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseArrayLiteral(start ast.BaseNode) (ast.Expression, error) {
	p.advance()
	elements := []ast.Expression{}
	if p.match(lexer.TokenRBracket) {
		return &ast.ArrayLiteral{BaseNode: start}, nil
	}
	for {
		if p.match(lexer.TokenComma) {
			elements = append(elements, nil)
			if p.match(lexer.TokenRBracket) {
				return &ast.ArrayLiteral{BaseNode: start, Elements: elements}, nil
			}
			continue
		}
		element, err := p.parseArrayElement()
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
		if p.match(lexer.TokenRBracket) {
			return &ast.ArrayLiteral{BaseNode: start, Elements: elements}, nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, err
		}
		if p.match(lexer.TokenRBracket) {
			return &ast.ArrayLiteral{BaseNode: start, Elements: elements}, nil
		}
	}
}

func (p *Parser) parseArrayElement() (ast.Expression, error) {
	if p.match(lexer.TokenEllipsis) {
		if p.isMissingSpreadExpression() {
			return nil, p.missingSpreadExpressionError()
		}
		value, err := p.parseConditional()
		if err != nil {
			return nil, err
		}
		return &ast.SpreadExpression{BaseNode: baseOf(value), Value: value}, nil
	}
	return p.parseConditional()
}

func (p *Parser) isMissingSpreadExpression() bool {
	switch p.current.Type {
	case lexer.TokenComma, lexer.TokenRBracket, lexer.TokenRBrace, lexer.TokenRParen, lexer.TokenEOF:
		return true
	default:
		return false
	}
}

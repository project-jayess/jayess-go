package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseArguments() ([]ast.Expression, error) {
	if err := p.expect(lexer.TokenLParen); err != nil {
		return nil, err
	}
	args := []ast.Expression{}
	if p.match(lexer.TokenRParen) {
		return args, nil
	}
	for {
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.match(lexer.TokenRParen) {
			return args, nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, err
		}
		if p.match(lexer.TokenRParen) {
			return args, nil
		}
	}
}

func (p *Parser) parseArgument() (ast.Expression, error) {
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

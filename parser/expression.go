package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseSequence() (ast.Expression, error) {
	left, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.TokenComma) {
		right, err := p.parseConditional()
		if err != nil {
			return nil, err
		}
		left = &ast.CommaExpression{BaseNode: baseOf(left), Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseConditional() (ast.Expression, error) {
	condition, err := p.parseNullish()
	if err != nil {
		return nil, err
	}
	if !p.match(lexer.TokenQuestion) {
		return condition, nil
	}
	consequent, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenColon); err != nil {
		return nil, err
	}
	alternative, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	return &ast.ConditionalExpression{
		BaseNode:    baseOf(condition),
		Condition:   condition,
		Consequent:  consequent,
		Alternative: alternative,
	}, nil
}

func (p *Parser) parseNullish() (ast.Expression, error) {
	left, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.TokenNullish) {
		right, err := p.parseLogicalOr()
		if err != nil {
			return nil, err
		}
		left = &ast.NullishCoalesceExpression{BaseNode: baseOf(left), Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseLogicalOr() (ast.Expression, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.TokenOr) {
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.LogicalExpression{BaseNode: baseOf(left), Operator: ast.OperatorOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseLogicalAnd() (ast.Expression, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.TokenAnd) {
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		left = &ast.LogicalExpression{BaseNode: baseOf(left), Operator: ast.OperatorAnd, Left: left, Right: right}
	}
	return left, nil
}

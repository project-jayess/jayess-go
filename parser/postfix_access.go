package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseIndexExpression(target ast.Expression, optional bool) (ast.Expression, error) {
	start := baseOf(target)
	p.advance()
	index, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return nil, err
	}
	return &ast.IndexExpression{BaseNode: start, Target: target, Index: index, Optional: optional}, nil
}

func (p *Parser) parseMemberExpression(target ast.Expression, optional bool) (ast.Expression, error) {
	p.advance()
	return p.parseMemberProperty(target, optional)
}

func (p *Parser) parseMemberProperty(target ast.Expression, optional bool) (ast.Expression, error) {
	start := baseOf(target)
	private := false
	if p.match(lexer.TokenHash) {
		private = true
	}
	property := p.current
	if !isObjectPropertyNameToken(property.Type) {
		return nil, p.errorAtCurrent("expected property name, got %s", p.current.Type)
	}
	p.advance()
	return &ast.MemberExpression{
		BaseNode: start,
		Target:   target,
		Property: property.Literal,
		Private:  private,
		Optional: optional,
	}, nil
}

func (p *Parser) parseOptionalPostfix(target ast.Expression) (ast.Expression, error) {
	p.advance()
	switch p.current.Type {
	case lexer.TokenLParen:
		args, err := p.parseArguments()
		if err != nil {
			return nil, err
		}
		return callExpression(target, args, true), nil
	case lexer.TokenLBracket:
		return p.parseIndexExpression(target, true)
	default:
		return p.parseMemberProperty(target, true)
	}
}

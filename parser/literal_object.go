package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseObjectLiteral(start ast.BaseNode) (ast.Expression, error) {
	p.advance()
	properties := []ast.ObjectProperty{}
	if p.match(lexer.TokenRBrace) {
		return &ast.ObjectLiteral{BaseNode: start}, nil
	}
	for {
		property, err := p.parseObjectProperty()
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
		if p.match(lexer.TokenRBrace) {
			return &ast.ObjectLiteral{BaseNode: start, Properties: properties}, nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, err
		}
		if p.match(lexer.TokenRBrace) {
			return &ast.ObjectLiteral{BaseNode: start, Properties: properties}, nil
		}
	}
}

func (p *Parser) parseObjectProperty() (ast.ObjectProperty, error) {
	if p.match(lexer.TokenEllipsis) {
		if p.isMissingSpreadExpression() {
			return ast.ObjectProperty{}, p.missingSpreadExpressionError()
		}
		value, err := p.parseConditional()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		return ast.ObjectProperty{Value: value, Spread: true}, nil
	}
	isAsync := p.matchObjectAsyncModifier()
	isGenerator := p.match(lexer.TokenStar)
	if p.match(lexer.TokenLBracket) {
		return p.parseComputedObjectProperty(isAsync, isGenerator)
	}
	key := p.current
	if !isObjectPropertyNameToken(key.Type) {
		return ast.ObjectProperty{}, p.errorAtCurrent("expected object property name, got %s", p.current.Type)
	}
	p.advance()
	if p.current.Type == lexer.TokenQuestion {
		return ast.ObjectProperty{}, p.unsupportedOptionalPropertyError()
	}
	if !isAsync && !isGenerator && isAccessorKeyword(key.Literal) && isObjectAccessorName(p.current.Type) {
		return p.parseObjectAccessor(key.Literal)
	}
	if !isAsync && !isGenerator && isAccessorKeyword(key.Literal) && p.current.Type == lexer.TokenLBracket {
		return p.parseComputedObjectAccessor(key.Literal)
	}
	if p.current.Type == lexer.TokenLt {
		return ast.ObjectProperty{}, p.unsupportedGenericTypeParametersError()
	}
	if p.current.Type == lexer.TokenLParen {
		return p.parseObjectMethod(key.Literal, nil, false, isAsync, isGenerator)
	}
	if isAsync || isGenerator {
		return ast.ObjectProperty{}, p.errorAtCurrent("expected object method parameters, got %s", p.current.Type)
	}
	if key.Type == lexer.TokenIdent && isObjectPropertyEnd(p.current.Type) {
		return ast.ObjectProperty{
			Key:       key.Literal,
			Value:     &ast.Identifier{BaseNode: baseFrom(key), Name: key.Literal},
			Shorthand: true,
		}, nil
	}
	if err := p.expect(lexer.TokenColon); err != nil {
		return ast.ObjectProperty{}, err
	}
	value, err := p.parseConditional()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{Key: key.Literal, Value: value}, nil
}

func (p *Parser) parseComputedObjectProperty(isAsync bool, isGenerator bool) (ast.ObjectProperty, error) {
	key, err := p.parseConditional()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return ast.ObjectProperty{}, err
	}
	if p.current.Type == lexer.TokenQuestion {
		return ast.ObjectProperty{}, p.unsupportedOptionalPropertyError()
	}
	if p.current.Type == lexer.TokenLParen {
		return p.parseObjectMethod("", key, true, isAsync, isGenerator)
	}
	if isAsync || isGenerator {
		return ast.ObjectProperty{}, p.errorAtCurrent("expected computed object method parameters, got %s", p.current.Type)
	}
	if err := p.expect(lexer.TokenColon); err != nil {
		return ast.ObjectProperty{}, err
	}
	value, err := p.parseConditional()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{KeyExpr: key, Value: value, Computed: true}, nil
}

func isObjectAccessorName(tokenType lexer.TokenType) bool {
	return isObjectPropertyNameToken(tokenType)
}

func isObjectPropertyEnd(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenComma || tokenType == lexer.TokenRBrace
}

package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) matchObjectAsyncModifier() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}

	state := p.snapshot()
	start := p.current
	p.advance()
	if p.current.Line > start.Line {
		p.restore(state)
		return false
	}
	next := p.current.Type
	p.restore(state)

	if next == lexer.TokenStar || next == lexer.TokenLBracket || isObjectPropertyNameToken(next) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) parseObjectMethod(name string, key ast.Expression, computed bool, isAsync bool, isGenerator bool) (ast.ObjectProperty, error) {
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ObjectProperty{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{
		Key:      name,
		KeyExpr:  key,
		Value:    &ast.FunctionExpression{Name: name, Params: params, Body: body, IsAsync: isAsync, IsGenerator: isGenerator},
		Computed: computed,
		Method:   true,
	}, nil
}

func (p *Parser) parseObjectAccessor(kind string) (ast.ObjectProperty, error) {
	name := p.current
	p.advance()
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	if err := validateNamedAccessorParameters(kind, name, params); err != nil {
		return ast.ObjectProperty{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ObjectProperty{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{
		Key:    name.Literal,
		Value:  &ast.FunctionExpression{Name: name.Literal, Params: params, Body: body},
		Method: true,
		Getter: kind == "get",
		Setter: kind == "set",
	}, nil
}

func (p *Parser) parseComputedObjectAccessor(kind string) (ast.ObjectProperty, error) {
	if err := p.expect(lexer.TokenLBracket); err != nil {
		return ast.ObjectProperty{}, err
	}
	key, err := p.parseConditional()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return ast.ObjectProperty{}, err
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	if err := validateComputedAccessorParameters(kind, params, p.errorAtCurrent); err != nil {
		return ast.ObjectProperty{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ObjectProperty{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{
		KeyExpr:  key,
		Value:    &ast.FunctionExpression{Params: params, Body: body},
		Computed: true,
		Method:   true,
		Getter:   kind == "get",
		Setter:   kind == "set",
	}, nil
}

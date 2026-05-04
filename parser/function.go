package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseFunctionDeclaration() (ast.Statement, error) {
	start := p.current
	return p.parseFunctionDeclarationWithAsync(start, false)
}

func (p *Parser) parseFunctionDeclarationWithAsync(start lexer.Token, isAsync bool) (ast.Statement, error) {
	if err := p.expect(lexer.TokenFunction); err != nil {
		return nil, err
	}
	isGenerator := p.match(lexer.TokenStar)
	name := p.current
	if name.Type != lexer.TokenIdent {
		return nil, p.errorAtCurrent("expected function name, got %s", p.current.Type)
	}
	p.advance()
	if p.current.Type == lexer.TokenLt {
		return nil, p.unsupportedGenericTypeParametersError()
	}
	params, err := p.parseParameterList()
	if err != nil {
		return nil, err
	}
	if p.current.Type == lexer.TokenSemicolon {
		return nil, p.unsupportedFunctionOverloadDeclarationError()
	}
	if p.current.Type == lexer.TokenColon {
		return nil, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDecl{
		BaseNode:    baseFrom(start),
		IsAsync:     isAsync,
		IsGenerator: isGenerator,
		Name:        name.Literal,
		Params:      params,
		Body:        body,
	}, nil
}

func (p *Parser) parseParameterList() ([]ast.Parameter, error) {
	if err := p.expect(lexer.TokenLParen); err != nil {
		return nil, err
	}
	params := []ast.Parameter{}
	if p.match(lexer.TokenRParen) {
		return params, nil
	}
	for {
		param, err := p.parseParameter()
		if err != nil {
			return nil, err
		}
		params = append(params, param)
		if p.match(lexer.TokenRParen) {
			return params, nil
		}
		if param.Rest {
			return nil, p.errorAtCurrent("rest parameter must be last")
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, err
		}
		if p.match(lexer.TokenRParen) {
			return params, nil
		}
	}
}

func (p *Parser) parseParameter() (ast.Parameter, error) {
	param := ast.Parameter{}
	if p.current.Type == lexer.TokenAt {
		return ast.Parameter{}, p.unsupportedDecoratorError()
	}
	if p.match(lexer.TokenEllipsis) {
		param.Rest = true
	}
	if p.isUnsupportedParameterPropertyModifierStart() {
		return ast.Parameter{}, p.unsupportedParameterPropertyModifierError()
	}
	pattern, name, err := p.parseBindingTarget()
	if err != nil {
		return ast.Parameter{}, err
	}
	param.Name = name
	param.Pattern = pattern
	if p.current.Type == lexer.TokenQuestion {
		return ast.Parameter{}, p.unsupportedOptionalParameterError()
	}
	if p.current.Type == lexer.TokenColon {
		return ast.Parameter{}, p.unsupportedTypeAnnotationError()
	}
	if p.match(lexer.TokenAssign) {
		if param.Rest {
			return ast.Parameter{}, errorAtPosition(ast.PositionOf(pattern), "rest parameter cannot have a default value")
		}
		value, err := p.parseConditional()
		if err != nil {
			return ast.Parameter{}, err
		}
		param.Default = value
	}
	return param, nil
}

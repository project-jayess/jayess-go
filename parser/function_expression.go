package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseFunctionExpression() (ast.Expression, error) {
	start := p.current
	return p.parseFunctionExpressionWithAsync(start, false)
}

func (p *Parser) parseFunctionExpressionWithAsync(start lexer.Token, isAsync bool) (ast.Expression, error) {
	if err := p.expect(lexer.TokenFunction); err != nil {
		return nil, err
	}
	isGenerator := p.match(lexer.TokenStar)
	name := ""
	if p.current.Type == lexer.TokenIdent {
		name = p.current.Literal
		p.advance()
		if p.current.Type == lexer.TokenLt {
			return nil, p.unsupportedGenericTypeParametersError()
		}
	}
	params, err := p.parseParameterList()
	if err != nil {
		return nil, err
	}
	if p.current.Type == lexer.TokenColon {
		return nil, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionExpression{
		BaseNode:    baseFrom(start),
		Name:        name,
		Params:      params,
		Body:        body,
		IsAsync:     isAsync,
		IsGenerator: isGenerator,
	}, nil
}

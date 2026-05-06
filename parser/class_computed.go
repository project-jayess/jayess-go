package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseComputedClassMember(start lexer.Token, static bool, isAsync bool, isGenerator bool) (ast.ClassMember, error) {
	key, err := p.parseComputedClassKey()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenQuestion {
		return ast.ClassMember{}, p.unsupportedOptionalPropertyError()
	}
	if p.current.Type != lexer.TokenLParen {
		if isAsync || isGenerator {
			return ast.ClassMember{}, p.errorAtCurrent("expected computed class method parameters, got %s", p.current.Type)
		}
		return p.parseClassFieldWithKey(start, "", key, true, static, false)
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode:    baseFrom(start),
		KeyExpr:     key,
		Params:      params,
		Body:        body,
		Computed:    true,
		Static:      static,
		IsAsync:     isAsync,
		IsGenerator: isGenerator,
	}, nil
}

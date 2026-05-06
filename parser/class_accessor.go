package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseClassAccessor(start lexer.Token, kind string, static bool, private bool) (ast.ClassMember, error) {
	if !private {
		private = p.match(lexer.TokenHash)
	}
	if p.current.Type == lexer.TokenLBracket {
		if private {
			return ast.ClassMember{}, p.errorAtCurrent("private computed class accessors are not supported")
		}
		return p.parseComputedClassAccessor(start, kind, static)
	}
	name := p.current
	if !isObjectPropertyNameToken(name.Type) {
		return ast.ClassMember{}, p.errorAtCurrent("expected class accessor name, got %s", p.current.Type)
	}
	p.advance()
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if err := validateNamedAccessorParameters(kind, name, params); err != nil {
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
		BaseNode: baseFrom(start),
		Name:     name.Literal,
		Params:   params,
		Body:     body,
		Getter:   kind == "get",
		Setter:   kind == "set",
		Private:  private,
		Static:   static,
	}, nil
}

func (p *Parser) parseComputedClassAccessor(start lexer.Token, kind string, static bool) (ast.ClassMember, error) {
	key, err := p.parseComputedClassKey()
	if err != nil {
		return ast.ClassMember{}, err
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if err := validateComputedAccessorParameters(kind, params, p.errorAtCurrent); err != nil {
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
		BaseNode: baseFrom(start),
		KeyExpr:  key,
		Params:   params,
		Body:     body,
		Computed: true,
		Getter:   kind == "get",
		Setter:   kind == "set",
		Static:   static,
	}, nil
}

func (p *Parser) parseComputedClassKey() (ast.Expression, error) {
	if err := p.expect(lexer.TokenLBracket); err != nil {
		return nil, err
	}
	key, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return nil, err
	}
	return key, nil
}

func isAccessorKeyword(name string) bool {
	return name == "get" || name == "set"
}

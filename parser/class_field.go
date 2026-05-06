package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseClassField(start lexer.Token, name string, static bool, private bool) (ast.ClassMember, error) {
	return p.parseClassFieldWithKey(start, name, nil, false, static, private)
}

func (p *Parser) parseClassFieldWithKey(start lexer.Token, name string, key ast.Expression, computed bool, static bool, private bool) (ast.ClassMember, error) {
	var value ast.Expression
	var err error
	if p.current.Type == lexer.TokenBang {
		return ast.ClassMember{}, p.unsupportedDefiniteAssignmentAssertionError()
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedTypeAnnotationError()
	}
	if p.match(lexer.TokenAssign) {
		value, err = p.parseSequence()
		if err != nil {
			return ast.ClassMember{}, err
		}
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode: baseFrom(start),
		Name:     name,
		KeyExpr:  key,
		Value:    value,
		Computed: computed,
		Field:    true,
		Private:  private,
		Static:   static,
	}, nil
}

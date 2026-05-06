package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isForEachHead(operator lexer.TokenType) bool {
	state := p.snapshot()
	defer p.restore(state)

	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		p.advance()
		if _, _, err := p.parseBindingPattern(); err != nil {
			return false
		}
		return p.current.Type == operator
	}

	target, err := p.parsePostfix()
	if err != nil || !isAssignmentTarget(target) {
		return false
	}
	return p.current.Type == operator
}

func (p *Parser) parseForOfStatement(start lexer.Token, isAwait bool) (ast.Statement, error) {
	kind, name, pattern, target, err := p.parseForEachHead(lexer.TokenOf)
	if err != nil {
		return nil, err
	}
	iterable, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRParen); err != nil {
		return nil, err
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.ForOfStatement{
		BaseNode: baseFrom(start),
		Kind:     kind,
		Name:     name,
		Pattern:  pattern,
		Target:   target,
		Iterable: iterable,
		Body:     body,
		Await:    isAwait,
	}, nil
}

func (p *Parser) parseForInStatement(start lexer.Token) (ast.Statement, error) {
	kind, name, pattern, target, err := p.parseForEachHead(lexer.TokenIn)
	if err != nil {
		return nil, err
	}
	object, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRParen); err != nil {
		return nil, err
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.ForInStatement{
		BaseNode: baseFrom(start),
		Kind:     kind,
		Name:     name,
		Pattern:  pattern,
		Target:   target,
		Object:   object,
		Body:     body,
	}, nil
}

func (p *Parser) parseForEachHead(operator lexer.TokenType) (ast.DeclarationKind, string, ast.BindingPattern, ast.Expression, error) {
	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		kind := declarationKind(p.current.Type)
		p.advance()
		pattern, name, err := p.parseBindingPattern()
		if err != nil {
			return "", "", nil, nil, err
		}
		if err := rejectForEachInitializer(pattern); err != nil {
			return "", "", nil, nil, err
		}
		if err := p.expect(operator); err != nil {
			return "", "", nil, nil, err
		}
		return kind, name, pattern, nil, nil
	}

	target, err := p.parsePostfix()
	if err != nil {
		return "", "", nil, nil, err
	}
	if !isAssignmentTarget(target) {
		return "", "", nil, nil, errorAtPosition(ast.PositionOf(target), "invalid for...in/of assignment target")
	}
	if err := p.expect(operator); err != nil {
		return "", "", nil, nil, err
	}
	return "", "", nil, target, nil
}

func rejectForEachInitializer(pattern ast.BindingPattern) error {
	if _, ok := pattern.(*ast.BindingDefault); ok {
		return errorAtPosition(ast.PositionOf(pattern), "for...in/of binding cannot have an initializer")
	}
	return nil
}

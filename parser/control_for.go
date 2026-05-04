package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseForStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	isAwait := p.match(lexer.TokenAwait)
	if err := p.expect(lexer.TokenLParen); err != nil {
		return nil, err
	}
	if p.isForEachHead(lexer.TokenOf) {
		return p.parseForOfStatement(start, isAwait)
	}
	if p.isForEachHead(lexer.TokenIn) {
		if isAwait {
			return nil, p.errorAtCurrent("for await only supports for...of")
		}
		return p.parseForInStatement(start)
	}
	if isAwait {
		return nil, p.errorAtCurrent("for await only supports for...of")
	}
	init, err := p.parseForInit()
	if err != nil {
		return nil, err
	}
	condition, err := p.parseForCondition()
	if err != nil {
		return nil, err
	}
	update, err := p.parseForUpdate()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	return &ast.ForStatement{BaseNode: baseFrom(start), Init: init, Condition: condition, Update: update, Body: body}, nil
}

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

func (p *Parser) parseForInit() (ast.Statement, error) {
	if p.match(lexer.TokenSemicolon) {
		return nil, nil
	}
	stmt, err := p.parseForClauseStatement()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseForCondition() (ast.Expression, error) {
	if p.match(lexer.TokenSemicolon) {
		return nil, nil
	}
	condition, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return condition, nil
}

func (p *Parser) parseForUpdate() (ast.Statement, error) {
	if p.match(lexer.TokenRParen) {
		return nil, nil
	}
	stmt, err := p.parseForClauseStatement()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRParen); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseForClauseStatement() (ast.Statement, error) {
	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		return p.parseVariableDeclarationTerminated(false)
	}
	expr, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if isAssignmentToken(p.current.Type) {
		return p.parseAssignmentStatementTerminated(expr, false)
	}
	return &ast.ExpressionStatement{BaseNode: baseOf(expr), Expression: expr}, nil
}

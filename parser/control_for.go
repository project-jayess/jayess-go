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

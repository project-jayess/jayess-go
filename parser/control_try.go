package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseTryStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	tryBody, err := p.parseBlockStatements()
	if err != nil {
		return nil, err
	}
	stmt := &ast.TryStatement{BaseNode: baseFrom(start), TryBody: tryBody}
	hasCatch, err := p.parseOptionalCatch(stmt)
	if err != nil {
		return nil, err
	}
	hasFinally, err := p.parseOptionalFinally(stmt)
	if err != nil {
		return nil, err
	}
	if !hasCatch && !hasFinally {
		return nil, errorAtToken(start, "try must include catch, finally, or both")
	}
	return stmt, nil
}

func (p *Parser) parseOptionalCatch(stmt *ast.TryStatement) (bool, error) {
	if !p.match(lexer.TokenCatch) {
		return false, nil
	}
	if p.match(lexer.TokenLParen) {
		pattern, name, err := p.parseBindingTarget()
		if err != nil {
			return false, err
		}
		stmt.CatchName = name
		stmt.CatchPattern = pattern
		if p.current.Type == lexer.TokenQuestion {
			return false, p.unsupportedOptionalBindingError()
		}
		if p.current.Type == lexer.TokenBang {
			return false, p.unsupportedDefiniteAssignmentAssertionError()
		}
		if p.current.Type == lexer.TokenColon {
			return false, p.unsupportedTypeAnnotationError()
		}
		if err := p.expect(lexer.TokenRParen); err != nil {
			return false, err
		}
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return false, err
	}
	stmt.CatchBody = body
	return true, nil
}

func (p *Parser) parseOptionalFinally(stmt *ast.TryStatement) (bool, error) {
	if !p.match(lexer.TokenFinally) {
		return false, nil
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return false, err
	}
	stmt.FinallyBody = body
	return true, nil
}

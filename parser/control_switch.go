package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseSwitchStatement() (ast.Statement, error) {
	start := p.current
	p.advance()
	if err := p.expect(lexer.TokenLParen); err != nil {
		return nil, err
	}
	discriminant, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRParen); err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	stmt := &ast.SwitchStatement{BaseNode: baseFrom(start), Discriminant: discriminant}
	hasDefault := false
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		var err error
		hasDefault, err = p.parseSwitchClause(stmt, hasDefault)
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(lexer.TokenRBrace); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *Parser) parseSwitchClause(stmt *ast.SwitchStatement, hasDefault bool) (bool, error) {
	switch p.current.Type {
	case lexer.TokenCase:
		p.advance()
		test, err := p.parseSequence()
		if err != nil {
			return hasDefault, err
		}
		if err := p.expect(lexer.TokenColon); err != nil {
			return hasDefault, err
		}
		consequent, err := p.parseSwitchConsequent()
		if err != nil {
			return hasDefault, err
		}
		stmt.Cases = append(stmt.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		return hasDefault, nil
	case lexer.TokenDefault:
		if hasDefault {
			return hasDefault, p.errorAtCurrent("duplicate default clause in switch")
		}
		p.advance()
		if err := p.expect(lexer.TokenColon); err != nil {
			return hasDefault, err
		}
		consequent, err := p.parseSwitchConsequent()
		if err != nil {
			return hasDefault, err
		}
		stmt.Default = consequent
		return true, nil
	default:
		return hasDefault, p.errorAtCurrent("expected case or default in switch, got %s", p.current.Type)
	}
}

func (p *Parser) parseSwitchConsequent() ([]ast.Statement, error) {
	statements := []ast.Statement{}
	for p.current.Type != lexer.TokenCase &&
		p.current.Type != lexer.TokenDefault &&
		p.current.Type != lexer.TokenRBrace &&
		p.current.Type != lexer.TokenEOF {
		statement, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

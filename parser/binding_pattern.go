package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseBindingPattern() (ast.BindingPattern, string, error) {
	pattern, name, err := p.parseBindingTarget()
	if err != nil {
		return nil, "", err
	}
	if p.current.Type == lexer.TokenQuestion {
		return nil, "", p.unsupportedOptionalBindingError()
	}
	if p.current.Type == lexer.TokenBang {
		return nil, "", p.unsupportedDefiniteAssignmentAssertionError()
	}
	if p.current.Type == lexer.TokenColon {
		return nil, "", p.unsupportedTypeAnnotationError()
	}
	if !p.match(lexer.TokenAssign) {
		return pattern, name, nil
	}
	value, err := p.parseConditional()
	if err != nil {
		return nil, "", err
	}
	return &ast.BindingDefault{Pattern: pattern, Value: value}, name, nil
}

func (p *Parser) parseBindingTarget() (ast.BindingPattern, string, error) {
	switch p.current.Type {
	case lexer.TokenIdent:
		name := p.current.Literal
		p.advance()
		return &ast.BindingName{Name: name}, name, nil
	case lexer.TokenLBracket:
		return p.parseArrayBindingPattern()
	case lexer.TokenLBrace:
		return p.parseObjectBindingPattern()
	default:
		return nil, "", p.errorAtCurrent("expected binding name or pattern, got %s", p.current.Type)
	}
}

func (p *Parser) isMissingRestBindingTarget() bool {
	switch p.current.Type {
	case lexer.TokenComma, lexer.TokenRBracket, lexer.TokenRBrace, lexer.TokenAssign, lexer.TokenEOF:
		return true
	default:
		return false
	}
}

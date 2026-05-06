package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseArrayBindingPattern() (ast.BindingPattern, string, error) {
	p.advance()
	elements := []ast.BindingPattern{}
	if p.match(lexer.TokenRBracket) {
		return &ast.ArrayBindingPattern{}, "", nil
	}
	for {
		if p.match(lexer.TokenComma) {
			elements = append(elements, nil)
			if p.match(lexer.TokenRBracket) {
				return &ast.ArrayBindingPattern{Elements: elements}, "", nil
			}
			continue
		}
		if p.match(lexer.TokenEllipsis) {
			if p.isMissingRestBindingTarget() {
				return nil, "", p.errorAtCurrent("rest binding requires a target")
			}
			rest, _, err := p.parseBindingTarget()
			if err != nil {
				return nil, "", err
			}
			elements = append(elements, &ast.BindingRest{Pattern: rest})
			if p.current.Type == lexer.TokenQuestion {
				return nil, "", p.unsupportedOptionalBindingError()
			}
			if p.current.Type == lexer.TokenBang {
				return nil, "", p.unsupportedDefiniteAssignmentAssertionError()
			}
			if p.current.Type == lexer.TokenColon {
				return nil, "", p.unsupportedTypeAnnotationError()
			}
			if p.current.Type == lexer.TokenComma {
				return nil, "", p.errorAtCurrent("rest binding must be last")
			}
			if p.current.Type == lexer.TokenAssign {
				return nil, "", p.errorAtCurrent("rest binding cannot have a default value")
			}
			if err := p.expect(lexer.TokenRBracket); err != nil {
				return nil, "", err
			}
			return &ast.ArrayBindingPattern{Elements: elements}, "", nil
		}
		element, _, err := p.parseBindingPattern()
		if err != nil {
			return nil, "", err
		}
		elements = append(elements, element)
		if p.match(lexer.TokenRBracket) {
			return &ast.ArrayBindingPattern{Elements: elements}, "", nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, "", err
		}
		if p.match(lexer.TokenRBracket) {
			return &ast.ArrayBindingPattern{Elements: elements}, "", nil
		}
	}
}

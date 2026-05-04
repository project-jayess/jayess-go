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

func (p *Parser) parseObjectBindingPattern() (ast.BindingPattern, string, error) {
	p.advance()
	properties := []ast.ObjectBindingProperty{}
	if p.match(lexer.TokenRBrace) {
		return &ast.ObjectBindingPattern{}, "", nil
	}
	for {
		if p.match(lexer.TokenEllipsis) {
			if p.isMissingRestBindingTarget() {
				return nil, "", p.errorAtCurrent("rest binding requires a target")
			}
			rest, _, err := p.parseBindingTarget()
			if err != nil {
				return nil, "", err
			}
			properties = append(properties, ast.ObjectBindingProperty{
				Pattern: &ast.BindingRest{Pattern: rest},
				Rest:    true,
			})
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
			if err := p.expect(lexer.TokenRBrace); err != nil {
				return nil, "", err
			}
			return &ast.ObjectBindingPattern{Properties: properties}, "", nil
		}
		property, err := p.parseObjectBindingProperty()
		if err != nil {
			return nil, "", err
		}
		properties = append(properties, property)
		if p.match(lexer.TokenRBrace) {
			return &ast.ObjectBindingPattern{Properties: properties}, "", nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, "", err
		}
		if p.match(lexer.TokenRBrace) {
			return &ast.ObjectBindingPattern{Properties: properties}, "", nil
		}
	}
}

func (p *Parser) parseObjectBindingProperty() (ast.ObjectBindingProperty, error) {
	if p.match(lexer.TokenLBracket) {
		return p.parseComputedObjectBindingProperty()
	}
	key := p.current
	if !isObjectPropertyNameToken(key.Type) {
		return ast.ObjectBindingProperty{}, p.errorAtCurrent("expected object binding property name, got %s", p.current.Type)
	}
	p.advance()
	if p.current.Type == lexer.TokenQuestion {
		return ast.ObjectBindingProperty{}, p.unsupportedOptionalBindingError()
	}
	if p.current.Type == lexer.TokenBang {
		return ast.ObjectBindingProperty{}, p.unsupportedDefiniteAssignmentAssertionError()
	}
	if p.match(lexer.TokenColon) {
		pattern, _, err := p.parseBindingPattern()
		if err != nil {
			return ast.ObjectBindingProperty{}, err
		}
		return ast.ObjectBindingProperty{Key: key.Literal, Pattern: pattern}, nil
	}
	if key.Type != lexer.TokenIdent {
		return ast.ObjectBindingProperty{}, p.errorAtCurrent("expected : after object binding property name")
	}
	pattern := ast.BindingPattern(&ast.BindingName{Name: key.Literal})
	if p.match(lexer.TokenAssign) {
		value, err := p.parseConditional()
		if err != nil {
			return ast.ObjectBindingProperty{}, err
		}
		pattern = &ast.BindingDefault{Pattern: pattern, Value: value}
	}
	return ast.ObjectBindingProperty{
		Key:     key.Literal,
		Pattern: pattern,
	}, nil
}

func (p *Parser) parseComputedObjectBindingProperty() (ast.ObjectBindingProperty, error) {
	key, err := p.parseConditional()
	if err != nil {
		return ast.ObjectBindingProperty{}, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return ast.ObjectBindingProperty{}, err
	}
	if err := p.expect(lexer.TokenColon); err != nil {
		return ast.ObjectBindingProperty{}, err
	}
	pattern, _, err := p.parseBindingPattern()
	if err != nil {
		return ast.ObjectBindingProperty{}, err
	}
	return ast.ObjectBindingProperty{KeyExpr: key, Pattern: pattern, Computed: true}, nil
}

func (p *Parser) isMissingRestBindingTarget() bool {
	switch p.current.Type {
	case lexer.TokenComma, lexer.TokenRBracket, lexer.TokenRBrace, lexer.TokenAssign, lexer.TokenEOF:
		return true
	default:
		return false
	}
}

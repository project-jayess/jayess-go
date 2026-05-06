package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isClassStaticBlockStart() bool {
	if p.current.Type != lexer.TokenStatic {
		return false
	}

	state := p.snapshot()
	p.advance()
	next := p.current.Type
	p.restore(state)
	return next == lexer.TokenLBrace
}

func (p *Parser) parseClassStaticBlock(start lexer.Token) (ast.ClassMember, error) {
	p.advance()
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode:    baseFrom(start),
		Body:        body,
		Static:      true,
		StaticBlock: true,
	}, nil
}

func (p *Parser) matchClassStaticModifier() bool {
	if p.current.Type != lexer.TokenStatic {
		return false
	}

	state := p.snapshot()
	p.advance()
	next := p.current.Type
	p.restore(state)

	if next == lexer.TokenHash || next == lexer.TokenLBracket || isObjectPropertyNameToken(next) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchClassAsyncModifier() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}

	state := p.snapshot()
	start := p.current
	p.advance()
	if p.current.Line > start.Line {
		p.restore(state)
		return false
	}
	next := p.current.Type
	p.restore(state)

	if next == lexer.TokenStar || next == lexer.TokenHash || next == lexer.TokenLBracket || isObjectPropertyNameToken(next) {
		p.advance()
		return true
	}
	return false
}

package parser

import "jayess-go/lexer"

func (p *Parser) isUnsupportedImplementsClauseStart() bool {
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "implements"
}

func (p *Parser) isUnsupportedAbstractClassDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "abstract" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenClass
}

func (p *Parser) isUnsupportedReadonlyClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("readonly")
}

func (p *Parser) isUnsupportedAbstractClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("abstract")
}

func (p *Parser) isUnsupportedClassAccessModifierStart() bool {
	return p.isUnsupportedClassMemberModifierStart("public") ||
		p.isUnsupportedClassMemberModifierStart("private") ||
		p.isUnsupportedClassMemberModifierStart("protected")
}

func (p *Parser) isUnsupportedOverrideClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("override")
}

func (p *Parser) isUnsupportedAccessorClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("accessor")
}

func (p *Parser) isUnsupportedClassMemberModifierStart(name string) bool {
	if p.current.Literal != name {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isClassMemberModifierTargetStart(p.current.Type)
}

func isClassMemberModifierTargetStart(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenHash || tokenType == lexer.TokenLBracket || isObjectPropertyNameToken(tokenType)
}

func (p *Parser) isUnsupportedParameterPropertyModifierStart() bool {
	return p.isUnsupportedParameterModifierStart("public") ||
		p.isUnsupportedParameterModifierStart("private") ||
		p.isUnsupportedParameterModifierStart("protected") ||
		p.isUnsupportedParameterModifierStart("readonly")
}

func (p *Parser) isUnsupportedParameterModifierStart(name string) bool {
	if p.current.Literal != name {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isParameterPropertyTargetStart(p.current.Type)
}

func isParameterPropertyTargetStart(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent, lexer.TokenLBrace, lexer.TokenLBracket:
		return true
	default:
		return false
	}
}

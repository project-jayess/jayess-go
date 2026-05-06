package parser

import "jayess-go/lexer"

func (p *Parser) isUnsupportedTypeAliasStart() bool {
	if p.current.Literal != "type" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenAssign
}

func (p *Parser) isUnsupportedConstEnumDeclarationStart() bool {
	if p.current.Type != lexer.TokenConst {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenEnum
}

func (p *Parser) isUnsupportedInterfaceStart() bool {
	if p.current.Literal != "interface" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedAmbientDeclarationStart() bool {
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "declare"
}

func (p *Parser) isUnsupportedNamespaceDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "namespace" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedModuleDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "module" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenString {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedImportEqualsDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenAssign
}

func (p *Parser) isUnsupportedExportAsNamespaceDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "as" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "namespace"
}

func (p *Parser) isUnsupportedTypeOnlyModuleSpecifierStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "type" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "as" {
		return false
	}
	return isModuleSpecifierNameToken(p.current.Type)
}

func (p *Parser) isUnsupportedUsingDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "using" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isUsingDeclarationBindingStart(p.current.Type)
}

func (p *Parser) isUnsupportedAwaitUsingDeclarationStart() bool {
	if p.current.Type != lexer.TokenAwait {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "using" {
		return false
	}
	p.advance()
	return isUsingDeclarationBindingStart(p.current.Type)
}

func isUsingDeclarationBindingStart(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent, lexer.TokenLBrace, lexer.TokenLBracket:
		return true
	default:
		return false
	}
}

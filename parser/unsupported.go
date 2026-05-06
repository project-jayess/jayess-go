package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseUnsupportedLetDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("let declarations are not supported; use var or const")
}

func (p *Parser) parseUnsupportedPublicModifier() (ast.Statement, error) {
	return nil, p.errorAtCurrent("public is not supported; use export for module visibility")
}

func (p *Parser) parseUnsupportedTopLevelPrivate() (ast.Statement, error) {
	return nil, p.errorAtCurrent("top-level private is not supported; use #members inside classes")
}

func (p *Parser) parseUnsupportedWithStatement() (ast.Statement, error) {
	return nil, p.errorAtCurrent("with statements are not supported")
}

func (p *Parser) parseUnsupportedEnumDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("enum declarations are not supported")
}

func (p *Parser) parseUnsupportedUsingDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("using declarations are not supported")
}

func (p *Parser) parseUnsupportedAbstractModifier() (ast.Statement, error) {
	return nil, p.unsupportedAbstractModifierError()
}

func (p *Parser) parseUnsupportedDecorator() (ast.Statement, error) {
	return nil, p.unsupportedDecoratorError()
}

func (p *Parser) parseUnsupportedTypeAlias() (ast.Statement, error) {
	return nil, p.errorAtCurrent("type aliases are not supported")
}

func (p *Parser) parseUnsupportedInterfaceDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("interface declarations are not supported")
}

func (p *Parser) parseUnsupportedAmbientDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("ambient declarations are not supported")
}

func (p *Parser) parseUnsupportedNamespaceDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("namespace declarations are not supported")
}

func (p *Parser) parseUnsupportedModuleDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("module declarations are not supported")
}

func (p *Parser) isUnsupportedConstAssertionSuffix() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "as" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenConst
}

func (p *Parser) isUnsupportedReturnTypePredicateStart() bool {
	if p.current.Type != lexer.TokenColon {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "asserts" {
		return true
	}
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenThis {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenIs
}

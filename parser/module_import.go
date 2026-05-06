package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseImportDeclaration() (ast.Statement, error) {
	start := p.current
	p.advance()
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "type" {
		return nil, p.unsupportedTypeOnlyModuleDeclarationError()
	}
	if p.isUnsupportedImportEqualsDeclarationStart() {
		return nil, p.unsupportedImportEqualsDeclarationError()
	}
	if p.current.Type == lexer.TokenString {
		source := p.current.Literal
		p.advance()
		if err := p.rejectUnsupportedImportAttributes(); err != nil {
			return nil, err
		}
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
		return &ast.ImportDecl{BaseNode: baseFrom(start), Source: source, SideEffect: true}, nil
	}
	specifiers, err := p.parseImportClause()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "from" {
		return nil, p.errorAtCurrent("expected from in import declaration")
	}
	p.advance()
	source := p.current
	if source.Type != lexer.TokenString {
		return nil, p.errorAtCurrent("expected import source string, got %s", p.current.Type)
	}
	p.advance()
	if err := p.rejectUnsupportedImportAttributes(); err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ImportDecl{BaseNode: baseFrom(start), Source: source.Literal, Specifiers: specifiers}, nil
}

func (p *Parser) rejectUnsupportedImportAttributes() error {
	if p.current.Type == lexer.TokenWith {
		return p.unsupportedImportAttributesError()
	}
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "assert" {
		return p.unsupportedImportAttributesError()
	}
	return nil
}

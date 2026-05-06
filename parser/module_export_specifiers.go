package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseExportSpecifiers() ([]ast.ExportSpecifier, error) {
	if err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	specifiers := []ast.ExportSpecifier{}
	if p.match(lexer.TokenRBrace) {
		return specifiers, nil
	}
	for {
		specifier, err := p.parseExportSpecifier()
		if err != nil {
			return nil, err
		}
		specifiers = append(specifiers, specifier)
		if p.match(lexer.TokenRBrace) {
			return specifiers, nil
		}
		if err := p.expect(lexer.TokenComma); err != nil {
			return nil, err
		}
		if p.match(lexer.TokenRBrace) {
			return specifiers, nil
		}
	}
}

func (p *Parser) parseExportSpecifier() (ast.ExportSpecifier, error) {
	if p.isUnsupportedTypeOnlyModuleSpecifierStart() {
		return ast.ExportSpecifier{}, p.unsupportedTypeOnlyModuleDeclarationError()
	}
	local, err := p.parseModuleSpecifierName("exported local name")
	if err != nil {
		return ast.ExportSpecifier{}, err
	}
	exported := local
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "as" {
		p.advance()
		alias, err := p.parseModuleSpecifierName("exported alias")
		if err != nil {
			return ast.ExportSpecifier{}, err
		}
		exported = alias
	}
	return ast.ExportSpecifier{Local: local, Exported: exported}, nil
}

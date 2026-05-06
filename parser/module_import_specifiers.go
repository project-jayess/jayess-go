package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseImportClause() ([]ast.ImportSpecifier, error) {
	if p.current.Type == lexer.TokenIdent {
		defaultImport := ast.ImportSpecifier{
			Imported: "default",
			Local:    p.current.Literal,
			Default:  true,
		}
		p.advance()
		if !p.match(lexer.TokenComma) {
			return []ast.ImportSpecifier{defaultImport}, nil
		}
		specifiers, err := p.parseNamedOrNamespaceImport()
		if err != nil {
			return nil, err
		}
		return append([]ast.ImportSpecifier{defaultImport}, specifiers...), nil
	}
	return p.parseNamedOrNamespaceImport()
}

func (p *Parser) parseNamedOrNamespaceImport() ([]ast.ImportSpecifier, error) {
	if p.current.Type == lexer.TokenStar {
		specifier, err := p.parseNamespaceImportSpecifier()
		if err != nil {
			return nil, err
		}
		return []ast.ImportSpecifier{specifier}, nil
	}
	return p.parseImportSpecifiers()
}

func (p *Parser) parseNamespaceImportSpecifier() (ast.ImportSpecifier, error) {
	if err := p.expect(lexer.TokenStar); err != nil {
		return ast.ImportSpecifier{}, err
	}
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "as" {
		return ast.ImportSpecifier{}, p.errorAtCurrent("expected as in namespace import")
	}
	p.advance()
	local := p.current
	if local.Type != lexer.TokenIdent {
		return ast.ImportSpecifier{}, p.errorAtCurrent("expected namespace import alias, got %s", p.current.Type)
	}
	p.advance()
	return ast.ImportSpecifier{Imported: "*", Local: local.Literal, Namespace: true}, nil
}

func (p *Parser) parseImportSpecifiers() ([]ast.ImportSpecifier, error) {
	if err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	specifiers := []ast.ImportSpecifier{}
	if p.match(lexer.TokenRBrace) {
		return specifiers, nil
	}
	for {
		specifier, err := p.parseImportSpecifier()
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

func (p *Parser) parseImportSpecifier() (ast.ImportSpecifier, error) {
	if p.isUnsupportedTypeOnlyModuleSpecifierStart() {
		return ast.ImportSpecifier{}, p.unsupportedTypeOnlyModuleDeclarationError()
	}
	importedType := p.current.Type
	imported, err := p.parseModuleSpecifierName("imported name")
	if err != nil {
		return ast.ImportSpecifier{}, err
	}
	local := imported
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "as" {
		p.advance()
		alias := p.current
		if alias.Type != lexer.TokenIdent {
			return ast.ImportSpecifier{}, p.errorAtCurrent("expected import alias, got %s", p.current.Type)
		}
		local = alias.Literal
		p.advance()
	} else if importedType != lexer.TokenIdent {
		return ast.ImportSpecifier{}, errorAtToken(p.previous, "non-identifier import name requires an alias")
	}
	return ast.ImportSpecifier{Imported: imported, Local: local}, nil
}

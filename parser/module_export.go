package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseExportDeclaration() (ast.Statement, error) {
	start := p.current
	p.advance()
	if p.current.Type == lexer.TokenAssign {
		return nil, p.unsupportedExportEqualsDeclarationError()
	}
	if p.isUnsupportedExportAsNamespaceDeclarationStart() {
		return nil, p.unsupportedExportAsNamespaceDeclarationError()
	}
	if p.match(lexer.TokenDefault) {
		return p.parseDefaultExport(start)
	}
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "type" {
		return nil, p.unsupportedTypeOnlyModuleDeclarationError()
	}
	if p.isUnsupportedAbstractClassDeclarationStart() {
		return nil, p.unsupportedAbstractModifierError()
	}
	if p.current.Type == lexer.TokenStar {
		return p.parseExportStar(start)
	}
	if p.current.Type == lexer.TokenLBrace {
		specifiers, err := p.parseExportSpecifiers()
		if err != nil {
			return nil, err
		}
		source, err := p.parseOptionalExportSource()
		if err != nil {
			return nil, err
		}
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
		return &ast.ExportDecl{BaseNode: baseFrom(start), Specifiers: specifiers, Source: source}, nil
	}
	declaration, err := p.ParseStatement()
	if err != nil {
		return nil, err
	}
	return &ast.ExportDecl{BaseNode: baseFrom(start), Declaration: declaration}, nil
}

func (p *Parser) parseDefaultExport(start lexer.Token) (ast.Statement, error) {
	if p.isAsyncFunctionStart() {
		if p.isAnonymousAsyncFunctionExpressionStart() {
			value, err := p.parseAsyncFunctionExpression()
			if err != nil {
				return nil, err
			}
			return &ast.ExportDecl{BaseNode: baseFrom(start), Value: value, Default: true}, nil
		}
		declaration, err := p.parseAsyncFunctionDeclaration()
		if err != nil {
			return nil, err
		}
		return &ast.ExportDecl{BaseNode: baseFrom(start), Declaration: declaration, Default: true}, nil
	}
	if p.current.Type == lexer.TokenFunction {
		if p.isAnonymousFunctionExpressionStart() {
			value, err := p.parseFunctionExpression()
			if err != nil {
				return nil, err
			}
			return &ast.ExportDecl{BaseNode: baseFrom(start), Value: value, Default: true}, nil
		}
		declaration, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		return &ast.ExportDecl{BaseNode: baseFrom(start), Declaration: declaration, Default: true}, nil
	}
	if p.current.Type == lexer.TokenClass {
		declaration, err := p.parseClassDeclarationWithName(false)
		if err != nil {
			return nil, err
		}
		return &ast.ExportDecl{BaseNode: baseFrom(start), Declaration: declaration, Default: true}, nil
	}
	if p.isUnsupportedAbstractClassDeclarationStart() {
		return nil, p.unsupportedAbstractModifierError()
	}
	value, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ExportDecl{BaseNode: baseFrom(start), Value: value, Default: true}, nil
}

func (p *Parser) isAnonymousFunctionExpressionStart() bool {
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type == lexer.TokenStar {
		p.advance()
	}
	return p.current.Type == lexer.TokenLParen
}

func (p *Parser) parseExportStar(start lexer.Token) (ast.Statement, error) {
	p.advance()
	namespace := ""
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "as" {
		p.advance()
		alias, err := p.parseModuleSpecifierName("namespace export alias")
		if err != nil {
			return nil, err
		}
		namespace = alias
	}
	source, err := p.parseRequiredExportSource()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ExportDecl{BaseNode: baseFrom(start), Source: source, All: namespace == "", Namespace: namespace}, nil
}

func (p *Parser) parseOptionalExportSource() (string, error) {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "from" {
		return "", nil
	}
	return p.parseRequiredExportSource()
}

func (p *Parser) parseRequiredExportSource() (string, error) {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "from" {
		return "", p.errorAtCurrent("expected from in export declaration")
	}
	p.advance()
	source := p.current
	if source.Type != lexer.TokenString {
		return "", p.errorAtCurrent("expected export source string, got %s", p.current.Type)
	}
	p.advance()
	if err := p.rejectUnsupportedImportAttributes(); err != nil {
		return "", err
	}
	return source.Literal, nil
}

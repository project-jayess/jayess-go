package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) ParseStatement() (ast.Statement, error) {
	if p.isLabeledStatementStart() {
		return p.parseLabeledStatement()
	}
	switch p.current.Type {
	case lexer.TokenAt:
		return p.parseUnsupportedDecorator()
	case lexer.TokenSemicolon:
		return p.parseEmptyStatement()
	case lexer.TokenDebugger:
		return p.parseDebuggerStatement()
	case lexer.TokenImport:
		if p.isImportMetaStart() {
			return p.parseExpressionStatement()
		}
		if p.isDynamicImportStart() {
			return nil, p.unsupportedDynamicImportError()
		}
		return p.parseImportDeclaration()
	case lexer.TokenExport:
		return p.parseExportDeclaration()
	case lexer.TokenClass:
		return p.parseClassDeclaration()
	case lexer.TokenEnum:
		return p.parseUnsupportedEnumDeclaration()
	case lexer.TokenFunction:
		return p.parseFunctionDeclaration()
	case lexer.TokenAsync:
		if p.isUnsupportedAsyncLineTerminatorStart() {
			return nil, p.unsupportedAsyncLineTerminatorError()
		}
		if p.isAsyncFunctionStart() {
			return p.parseAsyncFunctionDeclaration()
		}
		return p.parseExpressionStatement()
	case lexer.TokenAwait:
		if p.isUnsupportedAwaitUsingDeclarationStart() {
			return p.parseUnsupportedUsingDeclaration()
		}
		return p.parseExpressionStatement()
	case lexer.TokenLet:
		return p.parseUnsupportedLetDeclaration()
	case lexer.TokenPublic:
		return p.parseUnsupportedPublicModifier()
	case lexer.TokenPrivate:
		return p.parseUnsupportedTopLevelPrivate()
	case lexer.TokenConst:
		if p.isUnsupportedConstEnumDeclarationStart() {
			return p.parseUnsupportedEnumDeclaration()
		}
		return p.parseVariableDeclaration()
	case lexer.TokenVar:
		return p.parseVariableDeclaration()
	case lexer.TokenLBrace:
		return p.parseBlockStatement()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenSwitch:
		return p.parseSwitchStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenDo:
		return p.parseDoWhileStatement()
	case lexer.TokenWith:
		return p.parseUnsupportedWithStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenContinue:
		return p.parseContinueStatement()
	case lexer.TokenThrow:
		return p.parseThrowStatement()
	case lexer.TokenTry:
		return p.parseTryStatement()
	default:
		if p.isUnsupportedAbstractClassDeclarationStart() {
			return p.parseUnsupportedAbstractModifier()
		}
		if p.isUnsupportedUsingDeclarationStart() {
			return p.parseUnsupportedUsingDeclaration()
		}
		if p.isUnsupportedTypeAliasStart() {
			return p.parseUnsupportedTypeAlias()
		}
		if p.isUnsupportedInterfaceStart() {
			return p.parseUnsupportedInterfaceDeclaration()
		}
		if p.isUnsupportedAmbientDeclarationStart() {
			return p.parseUnsupportedAmbientDeclaration()
		}
		if p.isUnsupportedModuleDeclarationStart() {
			return p.parseUnsupportedModuleDeclaration()
		}
		if p.isUnsupportedNamespaceDeclarationStart() {
			return p.parseUnsupportedNamespaceDeclaration()
		}
		return p.parseExpressionStatement()
	}
}

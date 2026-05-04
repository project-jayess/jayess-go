package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

type Parser struct {
	lexer    *lexer.Lexer
	previous lexer.Token
	current  lexer.Token
}

type parserState struct {
	lexer    lexer.State
	previous lexer.Token
	current  lexer.Token
}

func New(l *lexer.Lexer) *Parser {
	return &Parser{lexer: l, current: l.NextToken()}
}

func (p *Parser) ParseProgram() (*ast.Program, error) {
	program := &ast.Program{BaseNode: baseFrom(p.current)}
	for p.current.Type != lexer.TokenEOF {
		statement, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		program.Statements = append(program.Statements, statement)
	}
	return program, nil
}

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

func (p *Parser) ParseExpression() (ast.Expression, error) {
	expr, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenEOF {
		return nil, p.errorAtCurrent("expected end of expression, got %s", p.current.Type)
	}
	return expr, nil
}

func (p *Parser) advance() {
	p.previous = p.current
	p.current = p.lexer.NextToken()
}

func (p *Parser) snapshot() parserState {
	return parserState{
		lexer:    p.lexer.Snapshot(),
		previous: p.previous,
		current:  p.current,
	}
}

func (p *Parser) restore(state parserState) {
	p.lexer.Restore(state.lexer)
	p.previous = state.previous
	p.current = state.current
}

func (p *Parser) match(tokenType lexer.TokenType) bool {
	if p.current.Type != tokenType {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) expect(tokenType lexer.TokenType) error {
	if p.current.Type != tokenType {
		if p.current.Type == lexer.TokenEOF {
			return p.errorAtCurrent("expected %s before end of file", tokenType)
		}
		return p.errorAtCurrent("expected %s, got %s", tokenType, p.current.Type)
	}
	p.advance()
	return nil
}

func baseFrom(token lexer.Token) ast.BaseNode {
	return ast.BaseNode{Pos: ast.SourcePos{Line: token.Line, Column: token.Column}}
}

func baseOf(expr ast.Expression) ast.BaseNode {
	return ast.BaseNode{Pos: ast.PositionOf(expr)}
}

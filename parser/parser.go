package parser

import (
	"fmt"
	"strconv"
	"strings"

	"jayess-go/ast"
	"jayess-go/lexer"
)

type Parser struct {
	lexer   *lexer.Lexer
	current lexer.Token
	peek    lexer.Token
}

type DiagnosticError struct {
	Line    int
	Column  int
	Message string
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	if e.Line > 0 {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

type state struct {
	lexer   lexer.State
	current lexer.Token
	peek    lexer.Token
}

func sourcePos(token lexer.Token) ast.SourcePos {
	return ast.SourcePos{Line: token.Line, Column: token.Column}
}

func (p *Parser) currentBase() ast.BaseNode {
	return ast.BaseNode{Pos: sourcePos(p.current)}
}

func (p *Parser) peekBase() ast.BaseNode {
	return ast.BaseNode{Pos: sourcePos(p.peek)}
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{lexer: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) ParseProgram() (*ast.Program, error) {
	program := &ast.Program{BaseNode: p.currentBase()}
	for p.current.Type != lexer.TokenEOF {
		switch p.current.Type {
		case lexer.TokenPrivate, lexer.TokenPublic:
			return nil, p.errorAtCurrent("top-level private/public are not supported; module visibility is controlled by export")
		case lexer.TokenLet:
			return nil, p.errorAtCurrent("let is not supported; use var or const")
		case lexer.TokenExtern:
			fn, err := p.parseExternFunction()
			if err != nil {
				return nil, err
			}
			program.ExternFunctions = append(program.ExternFunctions, fn)
			p.nextToken()
		case lexer.TokenVar, lexer.TokenConst:
			stmt, err := p.parseVariableDeclaration()
			if err != nil {
				return nil, err
			}
			decl, ok := stmt.(*ast.VariableDecl)
			if !ok {
				return nil, fmt.Errorf("top-level destructuring is not supported yet")
			}
			program.Globals = append(program.Globals, decl)
			p.nextToken()
		case lexer.TokenFunction:
			fn, err := p.parseFunction()
			if err != nil {
				return nil, err
			}
			program.Functions = append(program.Functions, fn)
		case lexer.TokenClass:
			classDecl, err := p.parseClass()
			if err != nil {
				return nil, err
			}
			program.Classes = append(program.Classes, classDecl)
		default:
			return nil, p.errorAtCurrent("unexpected top-level token %q", p.current.Literal)
		}
	}
	return program, nil
}

func (p *Parser) parseExternFunction() (*ast.ExternFunctionDecl, error) {
	start := p.currentBase()
	if err := p.expectCurrent(lexer.TokenExtern); err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenFunction); err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	for _, param := range params {
		if param.Pattern != nil {
			return nil, fmt.Errorf("extern functions do not support destructured parameters")
		}
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ExternFunctionDecl{BaseNode: start, Name: name, NativeSymbol: name, Params: params}, nil
}

func (p *Parser) parseFunction() (*ast.FunctionDecl, error) {
	start := p.currentBase()
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, fmt.Errorf("top-level private/public are not supported; module visibility is controlled by export")
	}
	if err := p.expectCurrent(lexer.TokenFunction); err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Name: name, Params: params, Body: body}, nil
}

func (p *Parser) parseClass() (*ast.ClassDecl, error) {
	start := p.currentBase()
	if err := p.expectCurrent(lexer.TokenClass); err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	classDecl := &ast.ClassDecl{BaseNode: start, Name: p.current.Literal}
	if p.peek.Type == lexer.TokenExtends {
		p.nextToken()
		if err := p.expectPeek(lexer.TokenIdent); err != nil {
			return nil, err
		}
		classDecl.SuperClass = p.current.Literal
	}
	if err := p.expectPeek(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	p.nextToken()
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		member, err := p.parseClassMember()
		if err != nil {
			return nil, err
		}
		classDecl.Members = append(classDecl.Members, member)
		if _, ok := member.(*ast.ClassMethodDecl); !ok {
			p.nextToken()
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' to close class %s", classDecl.Name)
	}
	p.nextToken()
	return classDecl, nil
}

func (p *Parser) parseClassMember() (ast.ClassMember, error) {
	start := p.currentBase()
	static := false
	if p.current.Type == lexer.TokenStatic {
		static = true
		p.nextToken()
	}

	private := false
	switch p.current.Type {
	case lexer.TokenHash:
		private = true
		if err := p.expectPeek(lexer.TokenIdent); err != nil {
			return nil, err
		}
	case lexer.TokenIdent:
	default:
		return nil, fmt.Errorf("unexpected class member token %q", p.current.Literal)
	}

	name := p.current.Literal
	if p.peek.Type == lexer.TokenLParen {
		p.nextToken()
		params, err := p.parseParameters()
		if err != nil {
			return nil, err
		}
		if err := p.expectPeek(lexer.TokenLBrace); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &ast.ClassMethodDecl{
			BaseNode:      start,
			Name:          name,
			Private:       private,
			Static:        static,
			IsConstructor: !private && !static && name == "constructor",
			Params:        params,
			Body:          body,
		}, nil
	}

	var initializer ast.Expression
	if p.peek.Type == lexer.TokenAssign {
		p.nextToken()
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		initializer = value
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ClassFieldDecl{BaseNode: start, Name: name, Private: private, Static: static, Initializer: initializer}, nil
}

func (p *Parser) parseParameters() ([]ast.Parameter, error) {
	var params []ast.Parameter
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return params, nil
	}
	for {
		rest := false
		if p.current.Type == lexer.TokenEllipsis {
			rest = true
			p.nextToken()
		}
		param := ast.Parameter{Rest: rest}
		switch p.current.Type {
		case lexer.TokenIdent:
			param.Name = p.current.Literal
		case lexer.TokenLBrace, lexer.TokenLBracket:
			if rest {
				return nil, fmt.Errorf("rest parameter must be an identifier")
			}
			pattern, err := p.parsePattern()
			if err != nil {
				return nil, err
			}
			param.Pattern = pattern
		default:
			return nil, fmt.Errorf("expected parameter name or pattern at %d:%d", p.current.Line, p.current.Column)
		}
		if p.peek.Type == lexer.TokenAssign {
			if rest {
				return nil, fmt.Errorf("rest parameter cannot have a default value")
			}
			p.nextToken()
			p.nextToken()
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			param.Default = value
		}
		params = append(params, param)
		if rest && p.peek.Type == lexer.TokenComma {
			return nil, fmt.Errorf("rest parameter must be last")
		}
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after parameters at %d:%d", p.peek.Line, p.peek.Column)
	}
	p.nextToken()
	return params, nil
}

func (p *Parser) parseBlock() ([]ast.Statement, error) {
	var statements []ast.Statement
	p.nextToken()
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
		switch stmt.(type) {
		case *ast.IfStatement, *ast.WhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement, *ast.SwitchStatement, *ast.TryStatement:
		default:
			p.nextToken()
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' to close block, got %q", p.current.Literal)
	}
	p.nextToken()
	return statements, nil
}

func (p *Parser) parseBlockExpression() ([]ast.Statement, error) {
	var statements []ast.Statement
	p.nextToken()
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
		switch stmt.(type) {
		case *ast.IfStatement, *ast.WhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement, *ast.SwitchStatement, *ast.TryStatement:
		default:
			p.nextToken()
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' to close block, got %q", p.current.Literal)
	}
	return statements, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.current.Type {
	case lexer.TokenPrivate, lexer.TokenPublic:
		return nil, fmt.Errorf("private/public are not supported here; use #private for class members")
	case lexer.TokenLet:
		return nil, fmt.Errorf("let is not supported; use var or const")
	case lexer.TokenVar, lexer.TokenConst:
		return p.parseVariableDeclaration()
	case lexer.TokenReturn:
		return p.parseReturn()
	case lexer.TokenIf:
		return p.parseIf()
	case lexer.TokenWhile:
		return p.parseWhile()
	case lexer.TokenFor:
		return p.parseFor()
	case lexer.TokenSwitch:
		return p.parseSwitch()
	case lexer.TokenBreak:
		return p.parseBreak()
	case lexer.TokenContinue:
		return p.parseContinue()
	case lexer.TokenDelete:
		return p.parseDelete()
	case lexer.TokenThrow:
		return p.parseThrow()
	case lexer.TokenTry:
		return p.parseTry()
	default:
		return p.parseExpressionOrAssignmentStatement()
	}
}

func (p *Parser) parseVariableDeclaration() (ast.Statement, error) {
	start := p.currentBase()
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, fmt.Errorf("private/public variable declarations are not supported; module visibility is controlled by export")
	}
	var kind ast.DeclarationKind
	switch p.current.Type {
	case lexer.TokenVar:
		kind = ast.DeclarationVar
	case lexer.TokenConst:
		kind = ast.DeclarationConst
	default:
		return nil, fmt.Errorf("expected var or const")
	}
	switch p.peek.Type {
	case lexer.TokenIdent:
		p.nextToken()
		name := p.current.Literal
		if err := p.expectPeek(lexer.TokenAssign); err != nil {
			return nil, err
		}
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
			return nil, err
		}
		return &ast.VariableDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Kind: kind, Name: name, Value: value}, nil
	case lexer.TokenLBrace, lexer.TokenLBracket:
		p.nextToken()
		pattern, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		if err := p.expectPeek(lexer.TokenAssign); err != nil {
			return nil, err
		}
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
			return nil, err
		}
		return &ast.DestructuringDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Kind: kind, Pattern: pattern, Value: value}, nil
	default:
		return nil, fmt.Errorf("expected variable name or destructuring pattern")
	}
}

func (p *Parser) parseAssignment(target ast.Expression) (ast.Statement, error) {
	start := ast.BaseNode{Pos: ast.PositionOf(target)}
	operator, err := parseAssignmentOperatorToken(p.peek.Type)
	if err != nil {
		return nil, err
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.AssignmentStatement{BaseNode: start, Target: target, Operator: operator, Value: value}, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	start := p.currentBase()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ReturnStatement{BaseNode: start, Value: value}, nil
}

func (p *Parser) parseBreak() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.BreakStatement{BaseNode: start}, nil
}

func (p *Parser) parseContinue() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ContinueStatement{BaseNode: start}, nil
}

func (p *Parser) parseDelete() (ast.Statement, error) {
	start := p.currentBase()
	p.nextToken()
	target, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.DeleteStatement{BaseNode: start, Target: target}, nil
}

func (p *Parser) parseThrow() (ast.Statement, error) {
	start := p.currentBase()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ThrowStatement{BaseNode: start, Value: value}, nil
}

func (p *Parser) parseTry() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after try")
	}
	p.nextToken()
	tryBody, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	stmt := &ast.TryStatement{BaseNode: start, TryBody: tryBody}
	if p.current.Type == lexer.TokenCatch {
		if p.peek.Type == lexer.TokenLParen {
			p.nextToken()
			if err := p.expectPeek(lexer.TokenIdent); err != nil {
				return nil, err
			}
			stmt.CatchName = p.current.Literal
			if p.peek.Type != lexer.TokenRParen {
				return nil, fmt.Errorf("expected ')' after catch binding")
			}
			p.nextToken()
		}
		if p.peek.Type != lexer.TokenLBrace {
			return nil, fmt.Errorf("expected '{' after catch")
		}
		p.nextToken()
		stmt.CatchBody, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	if p.current.Type == lexer.TokenFinally {
		if p.peek.Type != lexer.TokenLBrace {
			return nil, fmt.Errorf("expected '{' after finally")
		}
		p.nextToken()
		stmt.FinallyBody, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	if len(stmt.CatchBody) == 0 && len(stmt.FinallyBody) == 0 {
		return nil, fmt.Errorf("try must include catch, finally, or both")
	}
	return stmt, nil
}

func (p *Parser) parseIf() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	p.nextToken()
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after if condition")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after if condition")
	}
	p.nextToken()
	consequence, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	var alternative []ast.Statement
	if p.current.Type == lexer.TokenElse {
		if p.peek.Type == lexer.TokenIf {
			p.nextToken()
			elseIf, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			alternative = []ast.Statement{elseIf}
		} else {
			if p.peek.Type != lexer.TokenLBrace {
				return nil, fmt.Errorf("expected '{' after else")
			}
			p.nextToken()
			alternative, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
	}
	return &ast.IfStatement{BaseNode: start, Condition: condition, Consequence: consequence, Alternative: alternative}, nil
}

func (p *Parser) parseWhile() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	p.nextToken()
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after while condition")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after while condition")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStatement{BaseNode: start, Condition: condition, Body: body}, nil
}

func (p *Parser) parseFor() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	if p.isForEachLoopStart() {
		return p.parseForEach()
	}

	init, err := p.parseForInit()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenSemicolon {
		return nil, fmt.Errorf("expected ';' after for initializer")
	}

	condition, err := p.parseForCondition()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenSemicolon {
		return nil, fmt.Errorf("expected ';' after for condition")
	}

	update, err := p.parseForUpdate()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after for update")
	}
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after for clause")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.ForStatement{BaseNode: start, Init: init, Condition: condition, Update: update, Body: body}, nil
}

func (p *Parser) parseForEach() (ast.Statement, error) {
	start := p.currentBase()
	p.nextToken()
	if p.current.Type == lexer.TokenLet {
		return nil, fmt.Errorf("let is not supported; use var or const")
	}
	if p.current.Type != lexer.TokenVar && p.current.Type != lexer.TokenConst {
		return nil, fmt.Errorf("for...of and for...in require var or const")
	}
	kind := ast.DeclarationVar
	if p.current.Type == lexer.TokenConst {
		kind = ast.DeclarationConst
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	p.nextToken()
	mode := p.current.Type
	if mode != lexer.TokenOf && mode != lexer.TokenIn {
		return nil, fmt.Errorf("expected of or in in for-each loop")
	}
	p.nextToken()
	iterable, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after for-each iterable")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after for-each clause")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if mode == lexer.TokenOf {
		return &ast.ForOfStatement{BaseNode: start, Kind: kind, Name: name, Iterable: iterable, Body: body}, nil
	}
	return &ast.ForInStatement{BaseNode: start, Kind: kind, Name: name, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseSwitch() (ast.Statement, error) {
	start := p.currentBase()
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	p.nextToken()
	discriminant, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after switch discriminant")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after switch")
	}
	p.nextToken()
	p.nextToken()

	stmt := &ast.SwitchStatement{BaseNode: start, Discriminant: discriminant}
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		switch p.current.Type {
		case lexer.TokenCase:
			p.nextToken()
			test, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.peek.Type != lexer.TokenColon {
				return nil, fmt.Errorf("expected ':' after switch case")
			}
			p.nextToken()
			p.nextToken()
			consequent, err := p.parseSwitchConsequent()
			if err != nil {
				return nil, err
			}
			stmt.Cases = append(stmt.Cases, ast.SwitchCase{Test: test, Consequent: consequent})
		case lexer.TokenDefault:
			if p.peek.Type != lexer.TokenColon {
				return nil, fmt.Errorf("expected ':' after switch default")
			}
			p.nextToken()
			p.nextToken()
			consequent, err := p.parseSwitchConsequent()
			if err != nil {
				return nil, err
			}
			stmt.Default = consequent
		default:
			return nil, fmt.Errorf("unexpected token %q in switch", p.current.Literal)
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' to close switch")
	}
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseSwitchConsequent() ([]ast.Statement, error) {
	var statements []ast.Statement
	for p.current.Type != lexer.TokenCase && p.current.Type != lexer.TokenDefault && p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
		switch stmt.(type) {
		case *ast.IfStatement, *ast.WhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement, *ast.SwitchStatement, *ast.TryStatement:
		default:
			p.nextToken()
		}
	}
	return statements, nil
}

func (p *Parser) parseExpressionOrAssignmentStatement() (ast.Statement, error) {
	if p.current.Type == lexer.TokenLBrace || p.current.Type == lexer.TokenLBracket {
		if p.isDestructuringAssignmentStart() {
			pattern, err := p.parsePattern()
			if err != nil {
				return nil, err
			}
			if p.peek.Type != lexer.TokenAssign {
				return nil, fmt.Errorf("expected '=' after destructuring pattern")
			}
			p.nextToken()
			p.nextToken()
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
				return nil, err
			}
			return &ast.DestructuringAssignment{BaseNode: p.currentBase(), Pattern: pattern, Value: value}, nil
		}
	}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if isAssignmentToken(p.peek.Type) {
		return p.parseAssignment(expr)
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ExpressionStatement{BaseNode: p.currentBase(), Expression: expr}, nil
}

func (p *Parser) parsePattern() (ast.Pattern, error) {
	switch p.current.Type {
	case lexer.TokenIdent:
		return &ast.IdentifierPattern{BaseNode: p.currentBase(), Name: p.current.Literal}, nil
	case lexer.TokenLBrace:
		return p.parseObjectPattern()
	case lexer.TokenLBracket:
		return p.parseArrayPattern()
	default:
		return nil, fmt.Errorf("expected binding pattern at %d:%d", p.current.Line, p.current.Column)
	}
}

func (p *Parser) parseObjectPattern() (ast.Pattern, error) {
	pattern := &ast.ObjectPattern{BaseNode: p.currentBase()}
	p.nextToken()
	if p.current.Type == lexer.TokenRBrace {
		return pattern, nil
	}
	for {
		if p.current.Type == lexer.TokenEllipsis {
			p.nextToken()
			if p.current.Type != lexer.TokenIdent {
				return nil, fmt.Errorf("object rest element must be an identifier")
			}
			pattern.Rest = p.current.Literal
			if p.peek.Type != lexer.TokenRBrace {
				return nil, fmt.Errorf("object rest element must be last")
			}
			break
		}
		if p.current.Type != lexer.TokenIdent {
			return nil, fmt.Errorf("expected object pattern key at %d:%d", p.current.Line, p.current.Column)
		}
		key := p.current.Literal
		var valuePattern ast.Pattern
		if p.peek.Type == lexer.TokenColon {
			p.nextToken()
			p.nextToken()
			nested, err := p.parsePattern()
			if err != nil {
				return nil, err
			}
			valuePattern = nested
		} else {
			valuePattern = &ast.IdentifierPattern{Name: key}
		}
		var defaultValue ast.Expression
		if p.peek.Type == lexer.TokenAssign {
			p.nextToken()
			p.nextToken()
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			defaultValue = value
		}
		pattern.Properties = append(pattern.Properties, ast.ObjectPatternProperty{Key: key, Pattern: valuePattern, Default: defaultValue})
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' after object pattern at %d:%d", p.peek.Line, p.peek.Column)
	}
	p.nextToken()
	return pattern, nil
}

func (p *Parser) parseArrayPattern() (ast.Pattern, error) {
	pattern := &ast.ArrayPattern{BaseNode: p.currentBase()}
	p.nextToken()
	for p.current.Type != lexer.TokenRBracket {
		if p.current.Type == lexer.TokenEOF {
			return nil, fmt.Errorf("expected ']' after array pattern")
		}
		if p.current.Type == lexer.TokenComma {
			pattern.Elements = append(pattern.Elements, ast.ArrayPatternElement{})
			p.nextToken()
			continue
		}
		if p.current.Type == lexer.TokenEllipsis {
			p.nextToken()
			if p.current.Type != lexer.TokenIdent {
				return nil, fmt.Errorf("array rest element must be an identifier")
			}
			pattern.Elements = append(pattern.Elements, ast.ArrayPatternElement{
				Pattern: &ast.IdentifierPattern{Name: p.current.Literal},
				Rest:    true,
			})
			if p.peek.Type != lexer.TokenRBracket {
				return nil, fmt.Errorf("array rest element must be last")
			}
			p.nextToken()
			break
		}
		element, err := p.parsePattern()
		if err != nil {
			return nil, err
		}
		arrayElement := ast.ArrayPatternElement{Pattern: element}
		if p.peek.Type == lexer.TokenAssign {
			p.nextToken()
			p.nextToken()
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			arrayElement.Default = value
		}
		pattern.Elements = append(pattern.Elements, arrayElement)
		if p.peek.Type == lexer.TokenComma {
			p.nextToken()
			p.nextToken()
			continue
		}
		if p.peek.Type != lexer.TokenRBracket {
			return nil, fmt.Errorf("expected ']' after array pattern at %d:%d", p.peek.Line, p.peek.Column)
		}
		p.nextToken()
	}
	return pattern, nil
}

func (p *Parser) parseExpression() (ast.Expression, error) {
	return p.parseNullish()
}

func (p *Parser) parseNullish() (ast.Expression, error) {
	left, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenNullish {
		p.nextToken()
		p.nextToken()
		right, err := p.parseLogicalOr()
		if err != nil {
			return nil, err
		}
		left = &ast.NullishCoalesceExpression{Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseLogicalOr() (ast.Expression, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenOr {
		p.nextToken()
		p.nextToken()
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.LogicalExpression{Operator: ast.OperatorOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseLogicalAnd() (ast.Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenAnd {
		p.nextToken()
		p.nextToken()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ast.LogicalExpression{Operator: ast.OperatorAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseComparison() (ast.Expression, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for isComparisonToken(p.peek.Type) {
		p.nextToken()
		tokenType := p.current.Type
		p.nextToken()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		if tokenType == lexer.TokenInstanceof {
			left = &ast.InstanceofExpression{Left: left, Right: right}
			continue
		}
		operator := parseComparisonOperator(tokenType)
		left = &ast.ComparisonExpression{Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAdditive() (ast.Expression, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenPlus || p.peek.Type == lexer.TokenMinus {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMultiplicative() (ast.Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenStar || p.peek.Type == lexer.TokenSlash {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (ast.Expression, error) {
	if p.current.Type == lexer.TokenTypeof {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{BaseNode: p.currentBase(), Value: right}, nil
	}
	if p.current.Type == lexer.TokenBang {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{BaseNode: p.currentBase(), Operator: ast.OperatorNot, Right: right}, nil
	}
	if p.current.Type == lexer.TokenMinus {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{BaseNode: p.currentBase(), Operator: ast.OperatorSub, Left: &ast.NumberLiteral{BaseNode: p.currentBase(), Value: 0}, Right: right}, nil
	}
	return p.parsePostfix()
}

func tokenCanBePropertyName(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent,
		lexer.TokenFunction,
		lexer.TokenClass,
		lexer.TokenExtends,
		lexer.TokenExtern,
		lexer.TokenImport,
		lexer.TokenNative,
		lexer.TokenStatic,
		lexer.TokenNew,
		lexer.TokenTypeof,
		lexer.TokenInstanceof,
		lexer.TokenThis,
		lexer.TokenSuper,
		lexer.TokenVar,
		lexer.TokenLet,
		lexer.TokenConst,
		lexer.TokenPrivate,
		lexer.TokenPublic,
		lexer.TokenReturn,
		lexer.TokenIf,
		lexer.TokenElse,
		lexer.TokenWhile,
		lexer.TokenFor,
		lexer.TokenOf,
		lexer.TokenIn,
		lexer.TokenSwitch,
		lexer.TokenCase,
		lexer.TokenDefault,
		lexer.TokenBreak,
		lexer.TokenContinue,
		lexer.TokenDelete,
		lexer.TokenTry,
		lexer.TokenCatch,
		lexer.TokenFinally,
		lexer.TokenThrow,
		lexer.TokenTrue,
		lexer.TokenFalse,
		lexer.TokenNull,
		lexer.TokenUndefined:
		return true
	default:
		return false
	}
}

func (p *Parser) parseDotPropertyName() (string, error) {
	if !tokenCanBePropertyName(p.peek.Type) {
		return "", fmt.Errorf("expected property name after '.', got %s", p.peek.Type)
	}
	p.nextToken()
	return p.current.Literal, nil
}

func (p *Parser) parsePostfix() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek.Type {
		case lexer.TokenLParen:
			p.nextToken()
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			callBase := ast.BaseNode{Pos: ast.PositionOf(expr)}
			if ident, ok := expr.(*ast.Identifier); ok {
				expr = &ast.CallExpression{BaseNode: callBase, Callee: ident.Name, Arguments: args}
			} else {
				expr = &ast.InvokeExpression{BaseNode: callBase, Callee: expr, Arguments: args}
			}
		case lexer.TokenLBracket:
			p.nextToken()
			p.nextToken()
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.peek.Type != lexer.TokenRBracket {
				return nil, fmt.Errorf("expected ']' after index")
			}
			p.nextToken()
			expr = &ast.IndexExpression{BaseNode: p.currentBase(), Target: expr, Index: index}
		case lexer.TokenDot:
			p.nextToken()
			private := false
			if p.peek.Type == lexer.TokenHash {
				private = true
				p.nextToken()
			}
			property, err := p.parseDotPropertyName()
			if err != nil {
				return nil, err
			}
			expr = &ast.MemberExpression{BaseNode: p.currentBase(), Target: expr, Property: property, Private: private}
		case lexer.TokenQuestionDot:
			p.nextToken()
			switch p.peek.Type {
			case lexer.TokenLBracket:
				p.nextToken()
				p.nextToken()
				index, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				if p.peek.Type != lexer.TokenRBracket {
					return nil, fmt.Errorf("expected ']' after optional index")
				}
				p.nextToken()
				expr = &ast.IndexExpression{BaseNode: p.currentBase(), Target: expr, Index: index, Optional: true}
			case lexer.TokenLParen:
				p.nextToken()
				args, err := p.parseArguments()
				if err != nil {
					return nil, err
				}
				callBase := ast.BaseNode{Pos: ast.PositionOf(expr)}
				if ident, ok := expr.(*ast.Identifier); ok {
					expr = &ast.CallExpression{BaseNode: callBase, Callee: ident.Name, Arguments: args}
				} else {
					expr = &ast.InvokeExpression{BaseNode: callBase, Callee: expr, Arguments: args, Optional: true}
				}
			default:
				private := false
				if p.peek.Type == lexer.TokenHash {
					private = true
					p.nextToken()
				}
				property, err := p.parseDotPropertyName()
				if err != nil {
					return nil, err
				}
				expr = &ast.MemberExpression{BaseNode: p.currentBase(), Target: expr, Property: property, Private: private, Optional: true}
			}
		default:
			return expr, nil
		}
	}
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	switch p.current.Type {
	case lexer.TokenNumber:
		value, err := strconv.ParseFloat(p.current.Literal, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number literal %q", p.current.Literal)
		}
		return &ast.NumberLiteral{BaseNode: p.currentBase(), Value: value}, nil
	case lexer.TokenTrue:
		return &ast.BooleanLiteral{BaseNode: p.currentBase(), Value: true}, nil
	case lexer.TokenFalse:
		return &ast.BooleanLiteral{BaseNode: p.currentBase(), Value: false}, nil
	case lexer.TokenNull:
		return &ast.NullLiteral{BaseNode: p.currentBase()}, nil
	case lexer.TokenUndefined:
		return &ast.UndefinedLiteral{BaseNode: p.currentBase()}, nil
	case lexer.TokenString:
		return &ast.StringLiteral{BaseNode: p.currentBase(), Value: p.current.Literal}, nil
	case lexer.TokenTemplate:
		return parseTemplateLiteral(p.current.Literal)
	case lexer.TokenIdent:
		if p.peek.Type == lexer.TokenArrow {
			return p.parseSingleParamArrowFunction()
		}
		return &ast.Identifier{BaseNode: p.currentBase(), Name: p.current.Literal}, nil
	case lexer.TokenFunction:
		return p.parseFunctionExpression()
	case lexer.TokenThis:
		return &ast.ThisExpression{BaseNode: p.currentBase()}, nil
	case lexer.TokenSuper:
		return &ast.SuperExpression{BaseNode: p.currentBase()}, nil
	case lexer.TokenNew:
		return p.parseNewExpression()
	case lexer.TokenLParen:
		if p.isArrowFunctionStart() {
			return p.parseParenthesizedArrowFunction()
		}
		p.nextToken()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek.Type != lexer.TokenRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.nextToken()
		return expr, nil
	case lexer.TokenLBrace:
		return p.parseObjectLiteral()
	case lexer.TokenLBracket:
		return p.parseArrayLiteral()
	default:
		return nil, fmt.Errorf("unexpected expression token %q", p.current.Literal)
	}
}

func (p *Parser) parseFunctionExpression() (ast.Expression, error) {
	if err := p.expectCurrent(lexer.TokenFunction); err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	body, err := p.parseBlockExpression()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, Body: body}, nil
}

func (p *Parser) parseSingleParamArrowFunction() (ast.Expression, error) {
	param := ast.Parameter{Name: p.current.Literal}
	if err := p.expectPeek(lexer.TokenArrow); err != nil {
		return nil, err
	}
	p.nextToken()
	return p.parseArrowFunctionBody([]ast.Parameter{param})
}

func (p *Parser) parseParenthesizedArrowFunction() (ast.Expression, error) {
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenArrow); err != nil {
		return nil, err
	}
	p.nextToken()
	return p.parseArrowFunctionBody(params)
}

func (p *Parser) parseArrowFunctionBody(params []ast.Parameter) (ast.Expression, error) {
	if p.current.Type == lexer.TokenLBrace {
		body, err := p.parseBlockExpression()
		if err != nil {
			return nil, err
		}
		return &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, Body: body, IsArrowFunction: true}, nil
	}
	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, ExpressionBody: body, IsArrowFunction: true}, nil
}

func (p *Parser) parseNewExpression() (ast.Expression, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenDot {
		if err := p.expectPeek(lexer.TokenIdent); err != nil {
			return nil, err
		}
		if p.current.Literal != "target" {
			return nil, fmt.Errorf("expected target after new.")
		}
		return &ast.NewTargetExpression{BaseNode: p.currentBase()}, nil
	}
	callee, err := p.parsePostfix()
	if err != nil {
		return nil, err
	}
	switch expr := callee.(type) {
	case *ast.CallExpression:
		return &ast.NewExpression{BaseNode: p.currentBase(), Callee: &ast.Identifier{BaseNode: p.currentBase(), Name: expr.Callee}, Arguments: expr.Arguments}, nil
	case *ast.InvokeExpression:
		return &ast.NewExpression{BaseNode: p.currentBase(), Callee: expr.Callee, Arguments: expr.Arguments}, nil
	default:
		return &ast.NewExpression{BaseNode: p.currentBase(), Callee: callee}, nil
	}
}

func (p *Parser) parseObjectLiteral() (ast.Expression, error) {
	var properties []ast.ObjectProperty
	if p.peek.Type == lexer.TokenRBrace {
		p.nextToken()
		return &ast.ObjectLiteral{BaseNode: p.currentBase()}, nil
	}
	for {
		p.nextToken()
		property, err := p.parseObjectProperty()
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' after object literal")
	}
	p.nextToken()
	return &ast.ObjectLiteral{BaseNode: p.currentBase(), Properties: properties}, nil
}

func (p *Parser) parseObjectProperty() (ast.ObjectProperty, error) {
	if p.current.Type == lexer.TokenLBracket {
		p.nextToken()
		keyExpr, err := p.parseExpression()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		if p.peek.Type != lexer.TokenRBracket {
			return ast.ObjectProperty{}, fmt.Errorf("expected ']' after computed object key")
		}
		p.nextToken()
		if p.peek.Type != lexer.TokenColon {
			return ast.ObjectProperty{}, fmt.Errorf("expected ':' after computed object key")
		}
		p.nextToken()
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		return ast.ObjectProperty{KeyExpr: keyExpr, Value: value, Computed: true}, nil
	}
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenString {
		return ast.ObjectProperty{}, fmt.Errorf("expected object property name")
	}
	key := p.current.Literal
	if p.peek.Type == lexer.TokenLParen {
		p.nextToken()
		params, err := p.parseParameters()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		if err := p.expectPeek(lexer.TokenLBrace); err != nil {
			return ast.ObjectProperty{}, err
		}
		body, err := p.parseBlockExpression()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		return ast.ObjectProperty{Key: key, Value: &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, Body: body}}, nil
	}
	if p.peek.Type != lexer.TokenColon {
		return ast.ObjectProperty{}, fmt.Errorf("expected ':' after object property name")
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return ast.ObjectProperty{}, err
	}
	return ast.ObjectProperty{Key: key, Value: value}, nil
}

func (p *Parser) parseArrayLiteral() (ast.Expression, error) {
	var elements []ast.Expression
	if p.peek.Type == lexer.TokenRBracket {
		p.nextToken()
		return &ast.ArrayLiteral{BaseNode: p.currentBase()}, nil
	}
	for {
		p.nextToken()
		var (
			element ast.Expression
			err     error
		)
		if p.current.Type == lexer.TokenEllipsis {
			p.nextToken()
			value, parseErr := p.parseExpression()
			if parseErr != nil {
				return nil, parseErr
			}
			element = &ast.SpreadExpression{BaseNode: p.currentBase(), Value: value}
		} else {
			element, err = p.parseExpression()
		}
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' after array literal")
	}
	p.nextToken()
	return &ast.ArrayLiteral{BaseNode: p.currentBase(), Elements: elements}, nil
}

func (p *Parser) parseArguments() ([]ast.Expression, error) {
	var args []ast.Expression
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return args, nil
	}
	for {
		var (
			arg ast.Expression
			err error
		)
		if p.current.Type == lexer.TokenEllipsis {
			p.nextToken()
			value, parseErr := p.parseExpression()
			if parseErr != nil {
				return nil, parseErr
			}
			arg = &ast.SpreadExpression{BaseNode: p.currentBase(), Value: value}
		} else {
			arg, err = p.parseExpression()
		}
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	p.nextToken()
	return args, nil
}

func (p *Parser) parseForInit() (ast.Statement, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenSemicolon {
		return nil, nil
	}
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, fmt.Errorf("private/public are not supported in for initializers")
	}
	if p.current.Type == lexer.TokenLet {
		return nil, fmt.Errorf("let is not supported; use var or const")
	}
	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		stmt, err := p.parseInlineVariableDeclaration()
		if err != nil {
			return nil, err
		}
		if p.peek.Type != lexer.TokenSemicolon {
			return nil, fmt.Errorf("expected ';' after for initializer")
		}
		p.nextToken()
		return stmt, nil
	}
	stmt, err := p.parseInlineStatement()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenSemicolon {
		return nil, fmt.Errorf("expected ';' after for initializer")
	}
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseForCondition() (ast.Expression, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenSemicolon {
		return nil, nil
	}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenSemicolon {
		return nil, fmt.Errorf("expected ';' after for condition")
	}
	p.nextToken()
	return expr, nil
}

func (p *Parser) parseForUpdate() (ast.Statement, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return nil, nil
	}
	stmt, err := p.parseInlineStatement()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, fmt.Errorf("expected ')' after for update")
	}
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseInlineVariableDeclaration() (ast.Statement, error) {
	var kind ast.DeclarationKind
	switch p.current.Type {
	case lexer.TokenVar:
		kind = ast.DeclarationVar
	case lexer.TokenConst:
		kind = ast.DeclarationConst
	default:
		return nil, fmt.Errorf("expected var or const")
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	if err := p.expectPeek(lexer.TokenAssign); err != nil {
		return nil, err
	}
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.VariableDecl{BaseNode: p.currentBase(), Visibility: ast.VisibilityPublic, Kind: kind, Name: name, Value: value}, nil
}

func (p *Parser) parseInlineStatement() (ast.Statement, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type == lexer.TokenAssign {
		return p.parseInlineAssignment(expr)
	}
	return &ast.ExpressionStatement{BaseNode: p.currentBase(), Expression: expr}, nil
}

func (p *Parser) parseInlineAssignment(target ast.Expression) (ast.Statement, error) {
	if p.peek.Type != lexer.TokenAssign {
		return nil, p.errorAtPeek("expected '=' after assignment target")
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.AssignmentStatement{BaseNode: p.currentBase(), Target: target, Value: value}, nil
}

func (p *Parser) expectCurrent(expected lexer.TokenType) error {
	if p.current.Type != expected {
		return p.errorAtCurrent("expected %s, got %s", expected, p.current.Type)
	}
	return nil
}

func (p *Parser) expectPeek(expected lexer.TokenType) error {
	if p.peek.Type != expected {
		return p.errorAtPeek("expected next token %s, got %s", expected, p.peek.Type)
	}
	p.nextToken()
	return nil
}

func (p *Parser) errorAtCurrent(format string, args ...any) error {
	return &DiagnosticError{
		Line:    p.current.Line,
		Column:  p.current.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *Parser) errorAtPeek(format string, args ...any) error {
	return &DiagnosticError{
		Line:    p.peek.Line,
		Column:  p.peek.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) snapshot() state {
	return state{
		lexer:   p.lexer.Snapshot(),
		current: p.current,
		peek:    p.peek,
	}
}

func (p *Parser) restore(s state) {
	p.lexer.Restore(s.lexer)
	p.current = s.current
	p.peek = s.peek
}

func (p *Parser) isArrowFunctionStart() bool {
	saved := p.snapshot()
	defer p.restore(saved)

	if p.current.Type != lexer.TokenLParen {
		return false
	}
	if _, err := p.parseParameters(); err != nil {
		return false
	}
	return p.peek.Type == lexer.TokenArrow
}

func (p *Parser) isForEachLoopStart() bool {
	saved := p.snapshot()
	defer p.restore(saved)

	p.nextToken()
	if p.current.Type == lexer.TokenLet {
		return false
	}
	if p.current.Type != lexer.TokenVar && p.current.Type != lexer.TokenConst {
		return false
	}
	if p.peek.Type != lexer.TokenIdent {
		return false
	}
	p.nextToken()
	return p.peek.Type == lexer.TokenOf || p.peek.Type == lexer.TokenIn
}

func (p *Parser) isDestructuringAssignmentStart() bool {
	saved := p.snapshot()
	defer p.restore(saved)

	if p.current.Type != lexer.TokenLBrace && p.current.Type != lexer.TokenLBracket {
		return false
	}
	if _, err := p.parsePattern(); err != nil {
		return false
	}
	return p.peek.Type == lexer.TokenAssign
}

func isAssignmentToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenAssign, lexer.TokenAddAssign, lexer.TokenSubAssign, lexer.TokenMulAssign, lexer.TokenDivAssign, lexer.TokenNullishAssign, lexer.TokenOrAssign, lexer.TokenAndAssign:
		return true
	default:
		return false
	}
}

func parseAssignmentOperatorToken(tokenType lexer.TokenType) (ast.AssignmentOperator, error) {
	switch tokenType {
	case lexer.TokenAssign:
		return ast.AssignmentAssign, nil
	case lexer.TokenAddAssign:
		return ast.AssignmentAddAssign, nil
	case lexer.TokenSubAssign:
		return ast.AssignmentSubAssign, nil
	case lexer.TokenMulAssign:
		return ast.AssignmentMulAssign, nil
	case lexer.TokenDivAssign:
		return ast.AssignmentDivAssign, nil
	case lexer.TokenNullishAssign:
		return ast.AssignmentNullishAssign, nil
	case lexer.TokenOrAssign:
		return ast.AssignmentOrAssign, nil
	case lexer.TokenAndAssign:
		return ast.AssignmentAndAssign, nil
	default:
		return "", fmt.Errorf("expected assignment operator")
	}
}

func parseTemplateLiteral(raw string) (ast.Expression, error) {
	parts := []string{}
	values := []ast.Expression{}
	var text strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == '$' && i+1 < len(raw) && raw[i+1] == '{' {
			parts = append(parts, text.String())
			text.Reset()
			i += 2
			start := i
			depth := 1
			for i < len(raw) {
				if raw[i] == '{' {
					depth++
				} else if raw[i] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				i++
			}
			if depth != 0 {
				return nil, fmt.Errorf("unterminated template expression")
			}
			expr, err := parseEmbeddedExpression(raw[start:i])
			if err != nil {
				return nil, err
			}
			values = append(values, expr)
			continue
		}
		text.WriteByte(raw[i])
	}
	parts = append(parts, text.String())
	return &ast.TemplateLiteral{Parts: parts, Values: values}, nil
}

func parseEmbeddedExpression(source string) (ast.Expression, error) {
	p := New(lexer.New(strings.TrimSpace(source)))
	return p.parseExpression()
}

func parseOperator(tokenType lexer.TokenType) ast.BinaryOperator {
	switch tokenType {
	case lexer.TokenPlus:
		return ast.OperatorAdd
	case lexer.TokenMinus:
		return ast.OperatorSub
	case lexer.TokenStar:
		return ast.OperatorMul
	default:
		return ast.OperatorDiv
	}
}

func isComparisonToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenStrictEq, lexer.TokenStrictNe, lexer.TokenLt, lexer.TokenLte, lexer.TokenGt, lexer.TokenGte, lexer.TokenInstanceof:
		return true
	default:
		return false
	}
}

func parseComparisonOperator(tokenType lexer.TokenType) ast.ComparisonOperator {
	switch tokenType {
	case lexer.TokenEq:
		return ast.OperatorEq
	case lexer.TokenNe:
		return ast.OperatorNe
	case lexer.TokenStrictEq:
		return ast.OperatorStrictEq
	case lexer.TokenStrictNe:
		return ast.OperatorStrictNe
	case lexer.TokenLt:
		return ast.OperatorLt
	case lexer.TokenLte:
		return ast.OperatorLte
	case lexer.TokenGt:
		return ast.OperatorGt
	default:
		return ast.OperatorGte
	}
}

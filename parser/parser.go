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
		case lexer.TokenIdent:
			if p.current.Literal == "type" {
				alias, err := p.parseTypeAlias()
				if err != nil {
					return nil, err
				}
				program.TypeAliases = append(program.TypeAliases, alias)
				p.nextToken()
				continue
			}
			if p.current.Literal == "enum" {
				alias, global, err := p.parseEnum()
				if err != nil {
					return nil, err
				}
				program.TypeAliases = append(program.TypeAliases, alias)
				program.Globals = append(program.Globals, global)
				p.nextToken()
				continue
			}
			if p.current.Literal == "interface" {
				alias, err := p.parseInterfaceAlias()
				if err != nil {
					return nil, err
				}
				program.TypeAliases = append(program.TypeAliases, alias)
				p.nextToken()
				continue
			}
			return nil, p.errorAtCurrent("unexpected top-level token %q", p.current.Literal)
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
				return nil, p.errorAtCurrent("top-level destructuring is not supported yet")
			}
			program.Globals = append(program.Globals, decl)
			p.nextToken()
		case lexer.TokenFunction:
			fn, err := p.parseFunction()
			if err != nil {
				return nil, err
			}
			program.Functions = append(program.Functions, fn)
		case lexer.TokenAsync:
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

func (p *Parser) parseTypeAlias() (*ast.TypeAliasDecl, error) {
	start := p.currentBase()
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "type" {
		return nil, p.errorAtCurrent("expected type alias declaration")
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	typeParams, err := p.parseOptionalTypeParameters()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenAssign); err != nil {
		return nil, err
	}
	p.nextToken()
	target, err := p.parseTypeExpression(func(token lexer.TokenType) bool {
		return token == lexer.TokenSemicolon
	})
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenSemicolon {
		if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
			return nil, err
		}
	}
	return &ast.TypeAliasDecl{BaseNode: start, Name: name, TypeParams: typeParams, Target: target}, nil
}

func (p *Parser) parseInterfaceAlias() (*ast.TypeAliasDecl, error) {
	start := p.currentBase()
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "interface" {
		return nil, p.errorAtCurrent("expected interface declaration")
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, err
	}
	name := p.current.Literal
	typeParams, err := p.parseOptionalTypeParameters()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	p.nextToken()
	target := "{}"
	if p.current.Type != lexer.TokenRBrace {
		body, err := p.parseTypeExpression(func(token lexer.TokenType) bool {
			return token == lexer.TokenRBrace
		})
		if err != nil {
			return nil, err
		}
		target = "{" + body + "}"
		if err := p.expectPeek(lexer.TokenRBrace); err != nil {
			return nil, err
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, p.errorAtCurrent("expected '}' to close interface %s", name)
	}
	return &ast.TypeAliasDecl{BaseNode: start, Name: name, TypeParams: typeParams, Target: target}, nil
}

func (p *Parser) parseEnum() (*ast.TypeAliasDecl, *ast.VariableDecl, error) {
	start := p.currentBase()
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "enum" {
		return nil, nil, p.errorAtCurrent("expected enum declaration")
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return nil, nil, err
	}
	name := p.current.Literal
	if err := p.expectPeek(lexer.TokenLBrace); err != nil {
		return nil, nil, err
	}
	p.nextToken()
	properties := []ast.ObjectProperty{}
	members := []string{}
	nextNumber := 0
	canAutoNumber := true
	for p.current.Type != lexer.TokenRBrace && p.current.Type != lexer.TokenEOF {
		if p.current.Type != lexer.TokenIdent {
			return nil, nil, p.errorAtCurrent("expected enum member name")
		}
		memberName := p.current.Literal
		var value ast.Expression
		var memberType string
		if p.peek.Type == lexer.TokenAssign {
			p.nextToken()
			p.nextToken()
			switch p.current.Type {
			case lexer.TokenNumber:
				number, err := strconv.ParseFloat(p.current.Literal, 64)
				if err != nil {
					return nil, nil, p.errorAtCurrent("invalid enum numeric literal %q", p.current.Literal)
				}
				value = &ast.NumberLiteral{BaseNode: p.currentBase(), Value: number}
				memberType = strconv.FormatFloat(number, 'f', -1, 64)
				nextNumber = int(number) + 1
				canAutoNumber = true
			case lexer.TokenString:
				value = &ast.StringLiteral{BaseNode: p.currentBase(), Value: p.current.Literal}
				memberType = strconv.Quote(p.current.Literal)
				canAutoNumber = false
			default:
				return nil, nil, p.errorAtCurrent("enum member %s must be initialized with a number or string literal", memberName)
			}
		} else {
			if !canAutoNumber {
				return nil, nil, p.errorAtCurrent("enum member %s requires an explicit initializer after a non-numeric member", memberName)
			}
			value = &ast.NumberLiteral{BaseNode: start, Value: float64(nextNumber)}
			memberType = strconv.Itoa(nextNumber)
			nextNumber++
		}
		properties = append(properties, ast.ObjectProperty{Key: memberName, Value: value})
		members = append(members, memberType)
		if p.peek.Type == lexer.TokenComma {
			p.nextToken()
			p.nextToken()
			continue
		}
		if p.peek.Type == lexer.TokenRBrace {
			p.nextToken()
			break
		}
		return nil, nil, p.errorAtPeek("expected ',' or '}' after enum member")
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, nil, p.errorAtCurrent("expected '}' to close enum %s", name)
	}
	target := "never"
	if len(members) > 0 {
		target = strings.Join(members, "|")
	}
	alias := &ast.TypeAliasDecl{BaseNode: start, Name: name, Target: target}
	objectTypeParts := make([]string, len(properties))
	for i, property := range properties {
		objectTypeParts[i] = property.Key + ":" + members[i]
	}
	global := &ast.VariableDecl{
		BaseNode:       start,
		Visibility:     ast.VisibilityPublic,
		Kind:           ast.DeclarationConst,
		Name:           name,
		TypeAnnotation: "{" + strings.Join(objectTypeParts, ",") + "}",
		Value:          &ast.ObjectLiteral{BaseNode: start, Properties: properties},
	}
	return alias, global, nil
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
			return nil, p.errorAtCurrent("extern functions do not support destructured parameters")
		}
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ExternFunctionDecl{BaseNode: start, Name: name, NativeSymbol: name, Params: params}, nil
}

func (p *Parser) parseFunction() (*ast.FunctionDecl, error) {
	start := p.currentBase()
	isAsync := false
	isGenerator := false
	if p.current.Type == lexer.TokenAsync {
		isAsync = true
		if err := p.expectPeek(lexer.TokenFunction); err != nil {
			return nil, err
		}
	}
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, p.errorAtCurrent("top-level private/public are not supported; module visibility is controlled by export")
	}
	if err := p.expectCurrent(lexer.TokenFunction); err != nil {
		return nil, err
	}
	if p.peek.Type == lexer.TokenStar {
		p.nextToken()
		isGenerator = true
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
	returnType, err := p.parseOptionalReturnType()
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
	return &ast.FunctionDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Name: name, Params: params, ReturnType: returnType, IsAsync: isAsync, IsGenerator: isGenerator, Body: body}, nil
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
		return nil, p.errorAtCurrent("expected '}' to close class %s", classDecl.Name)
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
		return nil, p.errorAtCurrent("unexpected class member token %q", p.current.Literal)
	}

	name := p.current.Literal
	isGetter := false
	isSetter := false
	if !private && p.current.Type == lexer.TokenIdent && (p.current.Literal == "get" || p.current.Literal == "set") && p.peek.Type == lexer.TokenIdent {
		kind := p.current.Literal
		p.nextToken()
		name = p.current.Literal
		if p.peek.Type == lexer.TokenLParen {
			isGetter = kind == "get"
			isSetter = kind == "set"
		}
	}
	if p.peek.Type == lexer.TokenLParen {
		p.nextToken()
		params, err := p.parseParameters()
		if err != nil {
			return nil, err
		}
		if _, err := p.parseOptionalReturnType(); err != nil {
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
			IsGetter:      isGetter,
			IsSetter:      isSetter,
			Params:        params,
			Body:          body,
		}, nil
	}

	annotation, err := p.parseTypeAnnotation()
	if err != nil {
		return nil, err
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
	return &ast.ClassFieldDecl{BaseNode: start, Name: name, TypeAnnotation: annotation, Private: private, Static: static, Initializer: initializer}, nil
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
			if p.peek.Type == lexer.TokenColon {
				annotation, err := p.parseTypeAnnotation()
				if err != nil {
					return nil, err
				}
				param.TypeAnnotation = annotation
			}
		case lexer.TokenLBrace, lexer.TokenLBracket:
			if rest {
				return nil, p.errorAtCurrent("rest parameter must be an identifier")
			}
			pattern, err := p.parsePattern()
			if err != nil {
				return nil, err
			}
			param.Pattern = pattern
		default:
			return nil, p.errorAtCurrent("expected parameter name or pattern")
		}
		if p.peek.Type == lexer.TokenAssign {
			if rest {
				return nil, p.errorAtCurrent("rest parameter cannot have a default value")
			}
			p.nextToken()
			p.nextToken()
			value, err := p.parseExpressionNoComma()
			if err != nil {
				return nil, err
			}
			param.Default = value
		}
		params = append(params, param)
		if rest && p.peek.Type == lexer.TokenComma {
			return nil, p.errorAtPeek("rest parameter must be last")
		}
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, p.errorAtPeek("expected ')' after parameters")
	}
	p.nextToken()
	return params, nil
}

func (p *Parser) parseOptionalTypeParameters() ([]ast.TypeParameter, error) {
	if p.peek.Type != lexer.TokenLt {
		return nil, nil
	}
	p.nextToken()
	p.nextToken()
	params := []ast.TypeParameter{}
	for {
		if p.current.Type != lexer.TokenIdent {
			return nil, p.errorAtCurrent("expected type parameter name")
		}
		param := ast.TypeParameter{Name: p.current.Literal}
		if p.peek.Type == lexer.TokenExtends {
			p.nextToken()
			p.nextToken()
			constraint, err := p.parseTypeExpression(func(token lexer.TokenType) bool {
				return token == lexer.TokenComma || token == lexer.TokenGt
			})
			if err != nil {
				return nil, err
			}
			param.Constraint = constraint
		}
		params = append(params, param)
		if p.peek.Type == lexer.TokenComma {
			p.nextToken()
			p.nextToken()
			continue
		}
		if p.peek.Type != lexer.TokenGt {
			return nil, p.errorAtPeek("expected ',' or '>' after type parameter")
		}
		p.nextToken()
		break
	}
	return params, nil
}

func (p *Parser) parseTypeAnnotation() (string, error) {
	return p.parseTypeAnnotationWithStop(func(token lexer.TokenType) bool {
		switch token {
		case lexer.TokenComma, lexer.TokenRParen, lexer.TokenAssign, lexer.TokenSemicolon, lexer.TokenLBrace:
			return true
		default:
			return false
		}
	})
}

func (p *Parser) parseTypeAnnotationWithStop(stop func(lexer.TokenType) bool) (string, error) {
	if p.peek.Type != lexer.TokenColon {
		return "", nil
	}
	p.nextToken()
	p.nextToken()
	return p.parseTypeExpression(stop)
}

func (p *Parser) parseOptionalReturnType() (string, error) {
	if p.peek.Type != lexer.TokenColon {
		return "", nil
	}
	return p.parseTypeAnnotationWithStop(func(token lexer.TokenType) bool {
		switch token {
		case lexer.TokenLBrace, lexer.TokenArrow:
			return true
		default:
			return false
		}
	})
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
		if !statementConsumesFollowingToken(stmt) {
			p.nextToken()
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, p.errorAtCurrent("expected '}' to close block, got %q", p.current.Literal)
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
		if !statementConsumesFollowingToken(stmt) {
			p.nextToken()
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, p.errorAtCurrent("expected '}' to close block, got %q", p.current.Literal)
	}
	return statements, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.current.Type {
	case lexer.TokenPrivate, lexer.TokenPublic:
		return nil, p.errorAtCurrent("private/public are not supported here; use #private for class members")
	case lexer.TokenLet:
		return nil, p.errorAtCurrent("let is not supported; use var or const")
	case lexer.TokenVar, lexer.TokenConst:
		return p.parseVariableDeclaration()
	case lexer.TokenReturn:
		return p.parseReturn()
	case lexer.TokenIf:
		return p.parseIf()
	case lexer.TokenDo:
		return p.parseDoWhile()
	case lexer.TokenLBrace:
		return p.parseBlockStatement()
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
	case lexer.TokenIdent:
		if p.peek.Type == lexer.TokenColon {
			return p.parseLabeledStatement()
		}
		return p.parseExpressionOrAssignmentStatement()
	default:
		return p.parseExpressionOrAssignmentStatement()
	}
}

func statementConsumesFollowingToken(stmt ast.Statement) bool {
	switch stmt := stmt.(type) {
	case *ast.IfStatement, *ast.WhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement, *ast.SwitchStatement, *ast.TryStatement, *ast.BlockStatement:
		return true
	case *ast.LabeledStatement:
		return statementConsumesFollowingToken(stmt.Statement)
	default:
		return false
	}
}

func (p *Parser) parseVariableDeclaration() (ast.Statement, error) {
	return p.parseVariableDeclarationWithTerminator(true)
}

func (p *Parser) parseVariableDeclarationWithTerminator(consumeTerminator bool) (ast.Statement, error) {
	start := p.currentBase()
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, p.errorAtCurrent("private/public variable declarations are not supported; module visibility is controlled by export")
	}
	var kind ast.DeclarationKind
	switch p.current.Type {
	case lexer.TokenVar:
		kind = ast.DeclarationVar
	case lexer.TokenConst:
		kind = ast.DeclarationConst
	default:
		return nil, p.errorAtCurrent("expected var or const")
	}
	switch p.peek.Type {
	case lexer.TokenIdent:
		p.nextToken()
		name := p.current.Literal
		annotation, err := p.parseTypeAnnotation()
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
		if consumeTerminator {
			if err := p.consumeStatementTerminator(); err != nil {
				return nil, err
			}
		}
		return &ast.VariableDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Kind: kind, Name: name, TypeAnnotation: annotation, Value: value}, nil
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
		if consumeTerminator {
			if err := p.consumeStatementTerminator(); err != nil {
				return nil, err
			}
		}
		return &ast.DestructuringDecl{BaseNode: start, Visibility: ast.VisibilityPublic, Kind: kind, Pattern: pattern, Value: value}, nil
	default:
		return nil, p.errorAtPeek("expected variable name or destructuring pattern")
	}
}

func (p *Parser) parseAssignment(target ast.Expression) (ast.Statement, error) {
	start := ast.BaseNode{Pos: ast.PositionOf(target)}
	operator, err := parseAssignmentOperatorToken(p.peek.Type)
	if err != nil {
		return nil, err
	}
	return p.parseAssignmentStatement(start, target, operator, true)
}

func (p *Parser) parseAssignmentStatement(start ast.BaseNode, target ast.Expression, operator ast.AssignmentOperator, consumeTerminator bool) (ast.Statement, error) {
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if consumeTerminator {
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
	}
	return &ast.AssignmentStatement{BaseNode: start, Target: target, Operator: operator, Value: value}, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type == lexer.TokenSemicolon {
		p.nextToken()
		return &ast.ReturnStatement{BaseNode: start}, nil
	}
	if p.peek.Type == lexer.TokenRBrace || p.peek.Type == lexer.TokenEOF || p.lineBreakBeforePeek() {
		return &ast.ReturnStatement{BaseNode: start}, nil
	}
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ReturnStatement{BaseNode: start, Value: value}, nil
}

func (p *Parser) parseBreak() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type == lexer.TokenIdent && !p.lineBreakBeforePeek() {
		p.nextToken()
		label := p.current.Literal
		if err := p.consumeKeywordTerminator(); err != nil {
			return nil, err
		}
		return &ast.BreakStatement{BaseNode: start, Label: label}, nil
	}
	if err := p.consumeKeywordTerminator(); err != nil {
		return nil, err
	}
	return &ast.BreakStatement{BaseNode: start}, nil
}

func (p *Parser) parseContinue() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type == lexer.TokenIdent && !p.lineBreakBeforePeek() {
		p.nextToken()
		label := p.current.Literal
		if err := p.consumeKeywordTerminator(); err != nil {
			return nil, err
		}
		return &ast.ContinueStatement{BaseNode: start, Label: label}, nil
	}
	if err := p.consumeKeywordTerminator(); err != nil {
		return nil, err
	}
	return &ast.ContinueStatement{BaseNode: start}, nil
}

func (p *Parser) parseBlockStatement() (ast.Statement, error) {
	start := p.currentBase()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.BlockStatement{BaseNode: start, Body: body}, nil
}

func (p *Parser) parseLabeledStatement() (ast.Statement, error) {
	start := p.currentBase()
	label := p.current.Literal
	if err := p.expectPeek(lexer.TokenColon); err != nil {
		return nil, err
	}
	p.nextToken()
	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	return &ast.LabeledStatement{BaseNode: start, Label: label, Statement: stmt}, nil
}

func (p *Parser) parseDelete() (ast.Statement, error) {
	start := p.currentBase()
	p.nextToken()
	target, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.DeleteStatement{BaseNode: start, Target: target}, nil
}

func (p *Parser) parseThrow() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type == lexer.TokenSemicolon || p.peek.Type == lexer.TokenRBrace || p.peek.Type == lexer.TokenEOF || p.lineBreakBeforePeek() {
		return nil, p.errorAtPeek("line break or statement end is not allowed after throw")
	}
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ThrowStatement{BaseNode: start, Value: value}, nil
}

func (p *Parser) parseTry() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after try")
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
				return nil, p.errorAtPeek("expected ')' after catch binding")
			}
			p.nextToken()
		}
		if p.peek.Type != lexer.TokenLBrace {
			return nil, p.errorAtPeek("expected '{' after catch")
		}
		p.nextToken()
		stmt.CatchBody, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	if p.current.Type == lexer.TokenFinally {
		if p.peek.Type != lexer.TokenLBrace {
			return nil, p.errorAtPeek("expected '{' after finally")
		}
		p.nextToken()
		stmt.FinallyBody, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	if len(stmt.CatchBody) == 0 && len(stmt.FinallyBody) == 0 {
		return nil, p.errorAtCurrent("try must include catch, finally, or both")
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
		return nil, p.errorAtPeek("expected ')' after if condition")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after if condition")
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
				return nil, p.errorAtPeek("expected '{' after else")
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
		return nil, p.errorAtPeek("expected ')' after while condition")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after while condition")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStatement{BaseNode: start, Condition: condition, Body: body}, nil
}

func (p *Parser) parseDoWhile() (ast.Statement, error) {
	start := p.currentBase()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after do")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenWhile {
		return nil, p.errorAtCurrent("expected while after do block")
	}
	if p.peek.Type != lexer.TokenLParen {
		return nil, p.errorAtPeek("expected '(' after do while")
	}
	p.nextToken()
	p.nextToken()
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, p.errorAtPeek("expected ')' after do while condition")
	}
	p.nextToken()
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.DoWhileStatement{BaseNode: start, Body: body, Condition: condition}, nil
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
		return nil, p.errorAtCurrent("expected ';' after for initializer")
	}

	condition, err := p.parseForCondition()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenSemicolon {
		return nil, p.errorAtCurrent("expected ';' after for condition")
	}

	update, err := p.parseForUpdate()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenRParen {
		return nil, p.errorAtCurrent("expected ')' after for update")
	}
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after for clause")
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
		return nil, p.errorAtCurrent("let is not supported; use var or const")
	}
	kind, err := p.parseDeclarationKindAtCurrent("for...of and for...in require var or const")
	if err != nil {
		return nil, err
	}
	name, pattern, err := p.parseForEachBinding()
	if err != nil {
		return nil, err
	}
	p.nextToken()
	mode := p.current.Type
	if mode != lexer.TokenOf && mode != lexer.TokenIn {
		return nil, p.errorAtCurrent("expected of or in in for-each loop")
	}
	if pattern != nil && mode == lexer.TokenIn {
		return nil, p.errorAtCurrent("destructuring is not supported in for...in bindings")
	}
	p.nextToken()
	iterable, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenRParen {
		return nil, p.errorAtPeek("expected ')' after for-each iterable")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after for-each clause")
	}
	p.nextToken()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if pattern != nil {
		name, body = p.rewriteForEachDestructuringBinding(kind, pattern, iterable, body)
	}
	if mode == lexer.TokenOf {
		return &ast.ForOfStatement{BaseNode: start, Kind: kind, Name: name, Iterable: iterable, Body: body}, nil
	}
	return &ast.ForInStatement{BaseNode: start, Kind: kind, Name: name, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseDeclarationKindAtCurrent(message string) (ast.DeclarationKind, error) {
	switch p.current.Type {
	case lexer.TokenVar:
		return ast.DeclarationVar, nil
	case lexer.TokenConst:
		return ast.DeclarationConst, nil
	default:
		return "", p.errorAtCurrent(message)
	}
}

func (p *Parser) parseForEachBinding() (string, ast.Pattern, error) {
	if p.peek.Type == lexer.TokenLBrace || p.peek.Type == lexer.TokenLBracket {
		p.nextToken()
		pattern, err := p.parsePattern()
		if err != nil {
			return "", nil, err
		}
		return "", pattern, nil
	}
	if err := p.expectPeek(lexer.TokenIdent); err != nil {
		return "", nil, err
	}
	return p.current.Literal, nil, nil
}

func (p *Parser) rewriteForEachDestructuringBinding(kind ast.DeclarationKind, pattern ast.Pattern, iterable ast.Expression, body []ast.Statement) (string, []ast.Statement) {
	name := p.chooseForEachTempName(pattern, iterable, body)
	patternBase := ast.BaseNode{Pos: ast.PositionOf(pattern)}
	binding := &ast.DestructuringDecl{
		BaseNode:   patternBase,
		Visibility: ast.VisibilityPublic,
		Kind:       kind,
		Pattern:    pattern,
		Value:      &ast.Identifier{BaseNode: patternBase, Name: name},
	}
	return name, append([]ast.Statement{binding}, body...)
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
		return nil, p.errorAtPeek("expected ')' after switch discriminant")
	}
	p.nextToken()
	if p.peek.Type != lexer.TokenLBrace {
		return nil, p.errorAtPeek("expected '{' after switch")
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
				return nil, p.errorAtPeek("expected ':' after switch case")
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
				return nil, p.errorAtPeek("expected ':' after switch default")
			}
			p.nextToken()
			p.nextToken()
			consequent, err := p.parseSwitchConsequent()
			if err != nil {
				return nil, err
			}
			stmt.Default = consequent
		default:
			return nil, p.errorAtCurrent("unexpected token %q in switch", p.current.Literal)
		}
	}
	if p.current.Type != lexer.TokenRBrace {
		return nil, p.errorAtCurrent("expected '}' to close switch")
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
		if !statementConsumesFollowingToken(stmt) {
			p.nextToken()
		}
	}
	return statements, nil
}

func (p *Parser) parseExpressionOrAssignmentStatement() (ast.Statement, error) {
	return p.parseExpressionOrAssignmentStatementWithTerminator(true)
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
		return nil, p.errorAtCurrent("expected binding pattern")
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
				return nil, p.errorAtCurrent("object rest element must be an identifier")
			}
			pattern.Rest = p.current.Literal
			if p.peek.Type != lexer.TokenRBrace {
				return nil, p.errorAtPeek("object rest element must be last")
			}
			break
		}
		if p.current.Type != lexer.TokenIdent {
			return nil, p.errorAtCurrent("expected object pattern key")
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
			value, err := p.parseExpressionNoComma()
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
		return nil, p.errorAtPeek("expected '}' after object pattern")
	}
	p.nextToken()
	return pattern, nil
}

func (p *Parser) parseArrayPattern() (ast.Pattern, error) {
	pattern := &ast.ArrayPattern{BaseNode: p.currentBase()}
	p.nextToken()
	for p.current.Type != lexer.TokenRBracket {
		if p.current.Type == lexer.TokenEOF {
			return nil, p.errorAtCurrent("expected ']' after array pattern")
		}
		if p.current.Type == lexer.TokenComma {
			pattern.Elements = append(pattern.Elements, ast.ArrayPatternElement{})
			p.nextToken()
			continue
		}
		if p.current.Type == lexer.TokenEllipsis {
			p.nextToken()
			if p.current.Type != lexer.TokenIdent {
				return nil, p.errorAtCurrent("array rest element must be an identifier")
			}
			pattern.Elements = append(pattern.Elements, ast.ArrayPatternElement{
				Pattern: &ast.IdentifierPattern{Name: p.current.Literal},
				Rest:    true,
			})
			if p.peek.Type != lexer.TokenRBracket {
				return nil, p.errorAtPeek("array rest element must be last")
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
			value, err := p.parseExpressionNoComma()
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
			return nil, p.errorAtPeek("expected ']' after array pattern")
		}
		p.nextToken()
	}
	return pattern, nil
}

func (p *Parser) parseExpression() (ast.Expression, error) {
	return p.parseSequenceExpression()
}

func (p *Parser) parseSequenceExpression() (ast.Expression, error) {
	left, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenComma {
		p.nextToken()
		p.nextToken()
		right, err := p.parseConditional()
		if err != nil {
			return nil, err
		}
		left = &ast.CommaExpression{
			BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)},
			Left:     left,
			Right:    right,
		}
	}
	return left, nil
}

func (p *Parser) parseExpressionNoComma() (ast.Expression, error) {
	return p.parseConditional()
}

func (p *Parser) parseConditional() (ast.Expression, error) {
	condition, err := p.parseNullish()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != lexer.TokenQuestion {
		return condition, nil
	}
	p.nextToken()
	p.nextToken()
	consequent, err := p.parseExpressionNoComma()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenColon); err != nil {
		return nil, err
	}
	p.nextToken()
	alternative, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	return &ast.ConditionalExpression{
		BaseNode:    ast.BaseNode{Pos: ast.PositionOf(condition)},
		Condition:   condition,
		Consequent:  consequent,
		Alternative: alternative,
	}, nil
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
	left, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenAnd {
		p.nextToken()
		p.nextToken()
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		left = &ast.LogicalExpression{Operator: ast.OperatorAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseOr() (ast.Expression, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenBitOr {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseBitwiseXor()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)}, Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseXor() (ast.Expression, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenBitXor {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)}, Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBitwiseAnd() (ast.Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenBitAnd {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)}, Operator: operator, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseComparison() (ast.Expression, error) {
	left, err := p.parseShift()
	if err != nil {
		return nil, err
	}
	for isComparisonToken(p.peek.Type) {
		p.nextToken()
		tokenType := p.current.Type
		if tokenType == lexer.TokenIs {
			p.nextToken()
			typeAnnotation, err := p.parseTypeExpression(func(tokenType lexer.TokenType) bool {
				switch tokenType {
				case lexer.TokenSemicolon,
					lexer.TokenComma,
					lexer.TokenColon,
					lexer.TokenQuestion,
					lexer.TokenNullish,
					lexer.TokenAnd,
					lexer.TokenOr,
					lexer.TokenRParen,
					lexer.TokenRBracket,
					lexer.TokenRBrace:
					return true
				default:
					return false
				}
			})
			if err != nil {
				return nil, err
			}
			left = &ast.TypeCheckExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)}, Value: left, TypeAnnotation: typeAnnotation}
			continue
		}
		p.nextToken()
		right, err := p.parseShift()
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

func (p *Parser) parseShift() (ast.Expression, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.peek.Type == lexer.TokenShiftLeft || p.peek.Type == lexer.TokenShiftRight || p.peek.Type == lexer.TokenUnsignedShift {
		p.nextToken()
		operator := parseOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(left)}, Operator: operator, Left: left, Right: right}
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
	if p.current.Type == lexer.TokenBitNot {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{BaseNode: p.currentBase(), Operator: ast.OperatorBitNot, Right: right}, nil
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
		lexer.TokenIs,
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
		lexer.TokenAwait,
		lexer.TokenAsync,
		lexer.TokenTrue,
		lexer.TokenFalse,
		lexer.TokenNull,
		lexer.TokenUndefined:
		return true
	default:
		return false
	}
}

func tokenCanBeTypeName(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent, lexer.TokenString, lexer.TokenNumber, lexer.TokenTrue, lexer.TokenFalse, lexer.TokenNull, lexer.TokenUndefined, lexer.TokenLBracket, lexer.TokenLBrace, lexer.TokenLParen:
		return true
	default:
		return false
	}
}

func (p *Parser) parseTypeExpression(stop func(lexer.TokenType) bool) (string, error) {
	if !tokenCanBeTypeName(p.current.Type) {
		return "", p.errorAtCurrent("expected type annotation")
	}
	var builder strings.Builder
	parens := 0
	brackets := 0
	braces := 0
	for {
		if builder.Len() > 0 && needsTypeSpace(builder.String()[builder.Len()-1], p.current.Literal) {
			builder.WriteByte(' ')
		}
		switch p.current.Type {
		case lexer.TokenString:
			builder.WriteString(strconv.Quote(p.current.Literal))
		default:
			builder.WriteString(p.current.Literal)
		}
		switch p.current.Type {
		case lexer.TokenLParen:
			parens++
		case lexer.TokenRParen:
			parens--
		case lexer.TokenLBracket:
			brackets++
		case lexer.TokenRBracket:
			brackets--
		case lexer.TokenLBrace:
			braces++
		case lexer.TokenRBrace:
			braces--
			if braces < 0 {
				return "", p.errorAtCurrent("unexpected '}' in type annotation")
			}
		}
		if p.peek.Type == lexer.TokenEOF {
			break
		}
		if stop(p.peek.Type) && parens == 0 && brackets == 0 && braces == 0 {
			break
		}
		p.nextToken()
	}
	return builder.String(), nil
}

func needsTypeSpace(last byte, next string) bool {
	if len(next) == 0 {
		return false
	}
	nextByte := next[0]
	if isTypeWordChar(last) && isTypeWordChar(nextByte) {
		return true
	}
	return false
}

func isTypeWordChar(ch byte) bool {
	return ch == '_' || ch >= '0' && ch <= '9' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func (p *Parser) parseDotPropertyName() (string, error) {
	if !tokenCanBePropertyName(p.peek.Type) {
		return "", p.errorAtPeek("expected property name after '.', got %s", p.peek.Type)
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
		case lexer.TokenIdent:
			if p.peek.Literal != "as" {
				return expr, nil
			}
			p.nextToken()
			p.nextToken()
			if !tokenCanBeTypeName(p.current.Type) {
				return nil, p.errorAtCurrent("expected type name after as")
			}
			expr = &ast.CastExpression{BaseNode: ast.BaseNode{Pos: ast.PositionOf(expr)}, Value: expr, TypeAnnotation: p.current.Literal}
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
				return nil, p.errorAtPeek("expected ']' after index")
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
					return nil, p.errorAtPeek("expected ']' after optional index")
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
	case lexer.TokenBigInt:
		return &ast.BigIntLiteral{BaseNode: p.currentBase(), Value: p.current.Literal}, nil
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
	case lexer.TokenAsync:
		if p.isAsyncArrowFunctionStart() {
			return p.parseAsyncArrowFunction()
		}
		if p.peek.Type == lexer.TokenFunction {
			return p.parseFunctionExpression()
		}
		return nil, p.errorAtCurrent("async is only supported before function declarations and function expressions")
	case lexer.TokenThis:
		return &ast.ThisExpression{BaseNode: p.currentBase()}, nil
	case lexer.TokenSuper:
		return &ast.SuperExpression{BaseNode: p.currentBase()}, nil
	case lexer.TokenNew:
		return p.parseNewExpression()
	case lexer.TokenAwait:
		start := p.currentBase()
		p.nextToken()
		value, err := p.parsePostfix()
		if err != nil {
			return nil, err
		}
		return &ast.AwaitExpression{BaseNode: start, Value: value}, nil
	case lexer.TokenYield:
		start := p.currentBase()
		if p.peek.Type == lexer.TokenSemicolon || p.peek.Type == lexer.TokenRBrace || p.peek.Type == lexer.TokenEOF {
			return &ast.YieldExpression{BaseNode: start, Value: &ast.UndefinedLiteral{BaseNode: start}}, nil
		}
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.YieldExpression{BaseNode: start, Value: value}, nil
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
			return nil, p.errorAtPeek("expected ')'")
		}
		p.nextToken()
		return expr, nil
	case lexer.TokenLBrace:
		return p.parseObjectLiteral()
	case lexer.TokenLBracket:
		return p.parseArrayLiteral()
	default:
		return nil, p.errorAtCurrent("unexpected expression token %q", p.current.Literal)
	}
}

func (p *Parser) parseFunctionExpression() (ast.Expression, error) {
	start := p.currentBase()
	isAsync := false
	isGenerator := false
	if p.current.Type == lexer.TokenAsync {
		isAsync = true
		if err := p.expectPeek(lexer.TokenFunction); err != nil {
			return nil, err
		}
	}
	if err := p.expectCurrent(lexer.TokenFunction); err != nil {
		return nil, err
	}
	if p.peek.Type == lexer.TokenStar {
		p.nextToken()
		isGenerator = true
	}
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	returnType, err := p.parseOptionalReturnType()
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
	return &ast.FunctionExpression{BaseNode: start, Params: params, ReturnType: returnType, IsAsync: isAsync, IsGenerator: isGenerator, Body: body}, nil
}

func (p *Parser) parseSingleParamArrowFunction() (ast.Expression, error) {
	param := ast.Parameter{Name: p.current.Literal}
	if p.peek.Type == lexer.TokenColon {
		annotation, err := p.parseTypeAnnotation()
		if err != nil {
			return nil, err
		}
		param.TypeAnnotation = annotation
	}
	if err := p.expectPeek(lexer.TokenArrow); err != nil {
		return nil, err
	}
	p.nextToken()
	return p.parseArrowFunctionBody([]ast.Parameter{param}, "")
}

func (p *Parser) parseParenthesizedArrowFunction() (ast.Expression, error) {
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	returnType, err := p.parseOptionalReturnType()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenArrow); err != nil {
		return nil, err
	}
	p.nextToken()
	return p.parseArrowFunctionBody(params, returnType)
}

func (p *Parser) parseAsyncArrowFunction() (ast.Expression, error) {
	start := p.currentBase()
	p.nextToken()
	if p.current.Type == lexer.TokenIdent {
		param := ast.Parameter{Name: p.current.Literal}
		if p.peek.Type == lexer.TokenColon {
			annotation, err := p.parseTypeAnnotation()
			if err != nil {
				return nil, err
			}
			param.TypeAnnotation = annotation
		}
		if err := p.expectPeek(lexer.TokenArrow); err != nil {
			return nil, err
		}
		p.nextToken()
		expr, err := p.parseArrowFunctionBody([]ast.Parameter{param}, "")
		if err != nil {
			return nil, err
		}
		if fn, ok := expr.(*ast.FunctionExpression); ok {
			fn.BaseNode = start
			fn.IsAsync = true
		}
		return expr, nil
	}
	if p.current.Type != lexer.TokenLParen {
		return nil, p.errorAtCurrent("async arrow function expects parameter list")
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	returnType, err := p.parseOptionalReturnType()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenArrow); err != nil {
		return nil, err
	}
	p.nextToken()
	expr, err := p.parseArrowFunctionBody(params, returnType)
	if err != nil {
		return nil, err
	}
	if fn, ok := expr.(*ast.FunctionExpression); ok {
		fn.BaseNode = start
		fn.IsAsync = true
	}
	return expr, nil
}

func (p *Parser) parseArrowFunctionBody(params []ast.Parameter, returnType string) (ast.Expression, error) {
	if p.current.Type == lexer.TokenLBrace {
		body, err := p.parseBlockExpression()
		if err != nil {
			return nil, err
		}
		return &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, ReturnType: returnType, Body: body, IsArrowFunction: true}, nil
	}
	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, ReturnType: returnType, ExpressionBody: body, IsArrowFunction: true}, nil
}

func (p *Parser) parseNewExpression() (ast.Expression, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenDot {
		if err := p.expectPeek(lexer.TokenIdent); err != nil {
			return nil, err
		}
		if p.current.Literal != "target" {
			return nil, p.errorAtCurrent("expected target after new.")
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
		return nil, p.errorAtPeek("expected '}' after object literal")
	}
	p.nextToken()
	return &ast.ObjectLiteral{BaseNode: p.currentBase(), Properties: properties}, nil
}

func (p *Parser) parseObjectProperty() (ast.ObjectProperty, error) {
	if p.current.Type == lexer.TokenEllipsis {
		p.nextToken()
		value, err := p.parseExpressionNoComma()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		return ast.ObjectProperty{Value: value, Spread: true}, nil
	}
	if p.current.Type == lexer.TokenLBracket {
		p.nextToken()
		keyExpr, err := p.parseExpressionNoComma()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		if p.peek.Type != lexer.TokenRBracket {
			return ast.ObjectProperty{}, p.errorAtPeek("expected ']' after computed object key")
		}
		p.nextToken()
		if p.peek.Type != lexer.TokenColon {
			return ast.ObjectProperty{}, p.errorAtPeek("expected ':' after computed object key")
		}
		p.nextToken()
		p.nextToken()
		value, err := p.parseExpressionNoComma()
		if err != nil {
			return ast.ObjectProperty{}, err
		}
		return ast.ObjectProperty{KeyExpr: keyExpr, Value: value, Computed: true}, nil
	}
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenString {
		return ast.ObjectProperty{}, p.errorAtCurrent("expected object property name")
	}
	key := p.current.Literal
	isGetter := false
	isSetter := false
	if p.current.Type == lexer.TokenIdent && (p.current.Literal == "get" || p.current.Literal == "set") && (p.peek.Type == lexer.TokenIdent || p.peek.Type == lexer.TokenString) {
		kind := p.current.Literal
		p.nextToken()
		key = p.current.Literal
		if p.peek.Type == lexer.TokenLParen {
			isGetter = kind == "get"
			isSetter = kind == "set"
		}
	}
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
		return ast.ObjectProperty{
			Key:    key,
			Value:  &ast.FunctionExpression{BaseNode: p.currentBase(), Params: params, Body: body},
			Getter: isGetter,
			Setter: isSetter,
		}, nil
	}
	if p.peek.Type != lexer.TokenColon {
		return ast.ObjectProperty{}, p.errorAtPeek("expected ':' after object property name")
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpressionNoComma()
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
			value, parseErr := p.parseExpressionNoComma()
			if parseErr != nil {
				return nil, parseErr
			}
			element = &ast.SpreadExpression{BaseNode: p.currentBase(), Value: value}
		} else {
			element, err = p.parseExpressionNoComma()
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
		return nil, p.errorAtPeek("expected ']' after array literal")
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
			value, parseErr := p.parseExpressionNoComma()
			if parseErr != nil {
				return nil, parseErr
			}
			arg = &ast.SpreadExpression{BaseNode: p.currentBase(), Value: value}
		} else {
			arg, err = p.parseExpressionNoComma()
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
		return nil, p.errorAtPeek("expected ')' after arguments")
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
		return nil, p.errorAtCurrent("private/public are not supported in for initializers")
	}
	if p.current.Type == lexer.TokenLet {
		return nil, p.errorAtCurrent("let is not supported; use var or const")
	}
	stmt, err := p.parseForClauseStatement(lexer.TokenSemicolon, "for initializer")
	if err != nil {
		return nil, err
	}
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseForClauseStatement(terminator lexer.TokenType, context string) (ast.Statement, error) {
	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		stmt, err := p.parseInlineVariableDeclaration()
		if err != nil {
			return nil, err
		}
		if p.peek.Type != terminator {
			return nil, p.errorAtPeek("expected %q after %s", terminator, context)
		}
		return stmt, nil
	}
	stmt, err := p.parseInlineStatement()
	if err != nil {
		return nil, err
	}
	if p.peek.Type != terminator {
		return nil, p.errorAtPeek("expected %q after %s", terminator, context)
	}
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
		return nil, p.errorAtPeek("expected ';' after for condition")
	}
	p.nextToken()
	return expr, nil
}

func (p *Parser) parseForUpdate() (ast.Statement, error) {
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return nil, nil
	}
	stmt, err := p.parseForClauseStatement(lexer.TokenRParen, "for update")
	if err != nil {
		return nil, err
	}
	p.nextToken()
	return stmt, nil
}

func (p *Parser) parseInlineVariableDeclaration() (ast.Statement, error) {
	return p.parseVariableDeclarationWithTerminator(false)
}

func (p *Parser) parseInlineStatement() (ast.Statement, error) {
	return p.parseExpressionOrAssignmentStatementWithTerminator(false)
}

func (p *Parser) parseExpressionOrAssignmentStatementWithTerminator(consumeTerminator bool) (ast.Statement, error) {
	if p.current.Type == lexer.TokenLBrace || p.current.Type == lexer.TokenLBracket {
		if p.isDestructuringAssignmentStart() {
			stmt, err := p.parseDestructuringAssignment()
			if err != nil {
				return nil, err
			}
			if consumeTerminator {
				if err := p.consumeStatementTerminator(); err != nil {
					return nil, err
				}
			}
			return stmt, nil
		}
	}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if isAssignmentToken(p.peek.Type) {
		if consumeTerminator {
			return p.parseAssignment(expr)
		}
		return p.parseInlineAssignment(expr)
	}
	if consumeTerminator {
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
	}
	return &ast.ExpressionStatement{BaseNode: p.currentBase(), Expression: expr}, nil
}

func (p *Parser) parseDestructuringAssignment() (*ast.DestructuringAssignment, error) {
	pattern, err := p.parsePattern()
	if err != nil {
		return nil, err
	}
	start := ast.BaseNode{Pos: ast.PositionOf(pattern)}
	if p.peek.Type != lexer.TokenAssign {
		return nil, p.errorAtPeek("expected '=' after destructuring pattern")
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.DestructuringAssignment{BaseNode: start, Pattern: pattern, Value: value}, nil
}

func (p *Parser) parseInlineAssignment(target ast.Expression) (ast.Statement, error) {
	start := ast.BaseNode{Pos: ast.PositionOf(target)}
	operator, err := parseAssignmentOperatorToken(p.peek.Type)
	if err != nil {
		return nil, p.errorAtPeek("expected assignment operator after assignment target")
	}
	return p.parseAssignmentStatement(start, target, operator, false)
}

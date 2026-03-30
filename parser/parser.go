package parser

import (
	"fmt"
	"strconv"

	"jayess-go/ast"
	"jayess-go/lexer"
)

type Parser struct {
	lexer   *lexer.Lexer
	current lexer.Token
	peek    lexer.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{lexer: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) ParseProgram() (*ast.Program, error) {
	program := &ast.Program{}
	for p.current.Type != lexer.TokenEOF {
		switch p.current.Type {
		case lexer.TokenPrivate, lexer.TokenPublic:
			return nil, fmt.Errorf("top-level private/public are not supported; module visibility is controlled by export")
		case lexer.TokenLet:
			return nil, fmt.Errorf("let is not supported; use var or const")
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
				return nil, fmt.Errorf("expected top-level variable declaration")
			}
			program.Globals = append(program.Globals, decl)
			p.nextToken()
		default:
			fn, err := p.parseFunction()
			if err != nil {
				return nil, err
			}
			program.Functions = append(program.Functions, fn)
		}
	}
	return program, nil
}

func (p *Parser) parseExternFunction() (*ast.ExternFunctionDecl, error) {
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
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ExternFunctionDecl{Name: name, NativeSymbol: name, Params: params}, nil
}

func (p *Parser) parseFunction() (*ast.FunctionDecl, error) {
	visibility := ast.VisibilityPublic
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
	return &ast.FunctionDecl{Visibility: visibility, Name: name, Params: params, Body: body}, nil
}

func (p *Parser) parseParameters() ([]ast.Parameter, error) {
	var params []ast.Parameter
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return params, nil
	}
	for {
		if p.current.Type != lexer.TokenIdent {
			return nil, fmt.Errorf("expected parameter name at %d:%d", p.current.Line, p.current.Column)
		}
		params = append(params, ast.Parameter{Name: p.current.Literal})
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
		case *ast.IfStatement, *ast.WhileStatement, *ast.ForStatement:
			// These parse their trailing block and leave current positioned on the next token.
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
	case lexer.TokenBreak:
		return p.parseBreak()
	case lexer.TokenContinue:
		return p.parseContinue()
	default:
		if p.current.Type == lexer.TokenIdent && (p.peek.Type == lexer.TokenAssign || p.peek.Type == lexer.TokenDot || p.peek.Type == lexer.TokenLBracket) {
			target, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.peek.Type == lexer.TokenAssign {
				return p.parseAssignment(target)
			}
			return p.parseExpressionStatementFromExpr(target)
		}
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVariableDeclaration() (ast.Statement, error) {
	visibility := ast.VisibilityPublic
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
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.VariableDecl{Visibility: visibility, Kind: kind, Name: name, Value: value}, nil
}

func (p *Parser) parseAssignment(target ast.Expression) (ast.Statement, error) {
	if p.peek.Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after assignment target")
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
	return &ast.AssignmentStatement{Target: target, Value: value}, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ReturnStatement{Value: value}, nil
}

func (p *Parser) parseBreak() (ast.Statement, error) {
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.BreakStatement{}, nil
}

func (p *Parser) parseContinue() (ast.Statement, error) {
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ContinueStatement{}, nil
}

func (p *Parser) parseIf() (ast.Statement, error) {
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
	return &ast.IfStatement{Condition: condition, Consequence: consequence, Alternative: alternative}, nil
}

func (p *Parser) parseWhile() (ast.Statement, error) {
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
	return &ast.WhileStatement{Condition: condition, Body: body}, nil
}

func (p *Parser) parseFor() (ast.Statement, error) {
	if err := p.expectPeek(lexer.TokenLParen); err != nil {
		return nil, err
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
	return &ast.ForStatement{Init: init, Condition: condition, Update: update, Body: body}, nil
}

func (p *Parser) parseExpressionStatement() (ast.Statement, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return p.parseExpressionStatementFromExpr(expr)
}

func (p *Parser) parseExpressionStatementFromExpr(expr ast.Expression) (ast.Statement, error) {
	if err := p.expectPeek(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.ExpressionStatement{Expression: expr}, nil
}

func (p *Parser) parseExpression() (ast.Expression, error) {
	return p.parseLogicalOr()
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
		operator := parseComparisonOperator(p.current.Type)
		p.nextToken()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
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
	if p.current.Type == lexer.TokenBang {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: ast.OperatorNot, Right: right}, nil
	}
	if p.current.Type == lexer.TokenMinus {
		p.nextToken()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Operator: ast.OperatorSub, Left: &ast.NumberLiteral{Value: 0}, Right: right}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek.Type {
		case lexer.TokenLParen:
			ident, ok := expr.(*ast.Identifier)
			if !ok {
				return nil, fmt.Errorf("only named calls are supported")
			}
			p.nextToken()
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			expr = &ast.CallExpression{Callee: ident.Name, Arguments: args}
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
			expr = &ast.IndexExpression{Target: expr, Index: index}
		case lexer.TokenDot:
			p.nextToken()
			if err := p.expectPeek(lexer.TokenIdent); err != nil {
				return nil, err
			}
			expr = &ast.MemberExpression{Target: expr, Property: p.current.Literal}
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
		return &ast.NumberLiteral{Value: value}, nil
	case lexer.TokenTrue:
		return &ast.BooleanLiteral{Value: true}, nil
	case lexer.TokenFalse:
		return &ast.BooleanLiteral{Value: false}, nil
	case lexer.TokenNull:
		return &ast.NullLiteral{}, nil
	case lexer.TokenUndefined:
		return &ast.UndefinedLiteral{}, nil
	case lexer.TokenString:
		return &ast.StringLiteral{Value: p.current.Literal}, nil
	case lexer.TokenIdent:
		return &ast.Identifier{Name: p.current.Literal}, nil
	case lexer.TokenLParen:
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

func (p *Parser) parseObjectLiteral() (ast.Expression, error) {
	var properties []ast.ObjectProperty
	if p.peek.Type == lexer.TokenRBrace {
		p.nextToken()
		return &ast.ObjectLiteral{}, nil
	}
	for {
		p.nextToken()
		if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenString {
			return nil, fmt.Errorf("expected object property name")
		}
		key := p.current.Literal
		if p.peek.Type != lexer.TokenColon {
			return nil, fmt.Errorf("expected ':' after object property name")
		}
		p.nextToken()
		p.nextToken()
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		properties = append(properties, ast.ObjectProperty{Key: key, Value: value})
		if p.peek.Type != lexer.TokenComma {
			break
		}
		p.nextToken()
	}
	if p.peek.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' after object literal")
	}
	p.nextToken()
	return &ast.ObjectLiteral{Properties: properties}, nil
}

func (p *Parser) parseArrayLiteral() (ast.Expression, error) {
	var elements []ast.Expression
	if p.peek.Type == lexer.TokenRBracket {
		p.nextToken()
		return &ast.ArrayLiteral{}, nil
	}
	for {
		p.nextToken()
		element, err := p.parseExpression()
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
	return &ast.ArrayLiteral{Elements: elements}, nil
}

func (p *Parser) parseArguments() ([]ast.Expression, error) {
	var args []ast.Expression
	p.nextToken()
	if p.current.Type == lexer.TokenRParen {
		return args, nil
	}
	for {
		arg, err := p.parseExpression()
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
	var stmt ast.Statement
	var err error
	if p.current.Type == lexer.TokenVar || p.current.Type == lexer.TokenConst {
		stmt, err = p.parseInlineVariableDeclaration()
	} else {
		stmt, err = p.parseInlineStatement()
	}
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
	visibility := ast.VisibilityPublic
	if p.current.Type == lexer.TokenPrivate || p.current.Type == lexer.TokenPublic {
		return nil, fmt.Errorf("private/public variable declarations are not supported here")
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
	return &ast.VariableDecl{Visibility: visibility, Kind: kind, Name: name, Value: value}, nil
}

func (p *Parser) parseInlineStatement() (ast.Statement, error) {
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek.Type == lexer.TokenAssign {
		return p.parseInlineAssignment(expr)
	}
	return &ast.ExpressionStatement{Expression: expr}, nil
}

func (p *Parser) parseInlineAssignment(target ast.Expression) (ast.Statement, error) {
	if p.peek.Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after assignment target")
	}
	p.nextToken()
	p.nextToken()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.AssignmentStatement{Target: target, Value: value}, nil
}

func (p *Parser) expectCurrent(expected lexer.TokenType) error {
	if p.current.Type != expected {
		return fmt.Errorf("expected %s, got %s", expected, p.current.Type)
	}
	return nil
}

func (p *Parser) expectPeek(expected lexer.TokenType) error {
	if p.peek.Type != expected {
		return fmt.Errorf("expected next token %s, got %s", expected, p.peek.Type)
	}
	p.nextToken()
	return nil
}

func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
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
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLte, lexer.TokenGt, lexer.TokenGte:
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

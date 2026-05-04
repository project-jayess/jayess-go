package parser

import (
	"fmt"
	"strings"

	"jayess-go/ast"
	"jayess-go/lexer"
)

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

func (p *Parser) consumeStatementTerminator() error {
	if p.peek.Type == lexer.TokenSemicolon {
		p.nextToken()
		return nil
	}
	if p.peek.Type == lexer.TokenRBrace || p.peek.Type == lexer.TokenEOF || p.lineBreakBeforePeek() {
		return nil
	}
	return p.errorAtPeek("expected statement terminator, got %s", p.peek.Type)
}

func (p *Parser) consumeKeywordTerminator() error {
	if p.peek.Type == lexer.TokenSemicolon {
		p.nextToken()
		return nil
	}
	if p.peek.Type == lexer.TokenRBrace || p.peek.Type == lexer.TokenEOF || p.lineBreakBeforePeek() {
		return nil
	}
	return p.errorAtPeek("expected statement terminator after %s, got %s", p.current.Type, p.peek.Type)
}

func (p *Parser) lineBreakBeforePeek() bool {
	return p.peek.Line > p.current.Line
}

func (p *Parser) errorAtCurrent(format string, args ...any) error {
	if p.current.Type == lexer.TokenIllegal {
		return illegalTokenError(p.current)
	}
	return &DiagnosticError{
		Line:    p.current.Line,
		Column:  p.current.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *Parser) errorAtPeek(format string, args ...any) error {
	if p.peek.Type == lexer.TokenIllegal {
		return illegalTokenError(p.peek)
	}
	return &DiagnosticError{
		Line:    p.peek.Line,
		Column:  p.peek.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func illegalTokenError(token lexer.Token) error {
	message := token.Literal
	switch token.Literal {
	case "unterminated string", "unterminated template":
	default:
		message = fmt.Sprintf("unexpected character %q", token.Literal)
	}
	return &lexer.DiagnosticError{
		Line:    token.Line,
		Column:  token.Column,
		Message: message,
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
	if _, err := p.parseOptionalReturnType(); err != nil {
		return false
	}
	return p.peek.Type == lexer.TokenArrow
}

func (p *Parser) isAsyncArrowFunctionStart() bool {
	saved := p.snapshot()
	defer p.restore(saved)

	if p.current.Type != lexer.TokenAsync {
		return false
	}
	p.nextToken()
	if p.current.Type == lexer.TokenIdent {
		if p.peek.Type == lexer.TokenColon {
			if _, err := p.parseTypeAnnotation(); err != nil {
				return false
			}
		}
		return p.peek.Type == lexer.TokenArrow
	}
	if p.current.Type != lexer.TokenLParen {
		return false
	}
	if _, err := p.parseParameters(); err != nil {
		return false
	}
	if _, err := p.parseOptionalReturnType(); err != nil {
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
	if p.peek.Type == lexer.TokenIdent {
		p.nextToken()
		return p.peek.Type == lexer.TokenOf || p.peek.Type == lexer.TokenIn
	}
	if p.peek.Type != lexer.TokenLBrace && p.peek.Type != lexer.TokenLBracket {
		return false
	}
	p.nextToken()
	if _, err := p.parsePattern(); err != nil {
		return false
	}
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

func (p *Parser) chooseForEachTempName(pattern ast.Pattern, iterable ast.Expression, body []ast.Statement) string {
	for i := 0; ; i++ {
		name := fmt.Sprintf("__jayess_foreach_%d", i)
		if patternContainsIdentifier(pattern, name) {
			continue
		}
		if expressionContainsIdentifier(iterable, name) {
			continue
		}
		if statementsContainIdentifier(body, name) {
			continue
		}
		return name
	}
}

func statementsContainIdentifier(statements []ast.Statement, name string) bool {
	for _, stmt := range statements {
		if statementContainsIdentifier(stmt, name) {
			return true
		}
	}
	return false
}

func statementContainsIdentifier(stmt ast.Statement, name string) bool {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		return stmt.Name == name || expressionContainsIdentifier(stmt.Value, name)
	case *ast.DestructuringDecl:
		return patternContainsIdentifier(stmt.Pattern, name) || expressionContainsIdentifier(stmt.Value, name)
	case *ast.AssignmentStatement:
		return expressionContainsIdentifier(stmt.Target, name) || expressionContainsIdentifier(stmt.Value, name)
	case *ast.DestructuringAssignment:
		return patternContainsIdentifier(stmt.Pattern, name) || expressionContainsIdentifier(stmt.Value, name)
	case *ast.ReturnStatement:
		return expressionContainsIdentifier(stmt.Value, name)
	case *ast.ExpressionStatement:
		return expressionContainsIdentifier(stmt.Expression, name)
	case *ast.DeleteStatement:
		return expressionContainsIdentifier(stmt.Target, name)
	case *ast.ThrowStatement:
		return expressionContainsIdentifier(stmt.Value, name)
	case *ast.IfStatement:
		return expressionContainsIdentifier(stmt.Condition, name) ||
			statementsContainIdentifier(stmt.Consequence, name) ||
			statementsContainIdentifier(stmt.Alternative, name)
	case *ast.WhileStatement:
		return expressionContainsIdentifier(stmt.Condition, name) ||
			statementsContainIdentifier(stmt.Body, name)
	case *ast.DoWhileStatement:
		return statementsContainIdentifier(stmt.Body, name) ||
			expressionContainsIdentifier(stmt.Condition, name)
	case *ast.BlockStatement:
		return statementsContainIdentifier(stmt.Body, name)
	case *ast.ForStatement:
		return (stmt.Init != nil && statementContainsIdentifier(stmt.Init, name)) ||
			(stmt.Condition != nil && expressionContainsIdentifier(stmt.Condition, name)) ||
			(stmt.Update != nil && statementContainsIdentifier(stmt.Update, name)) ||
			statementsContainIdentifier(stmt.Body, name)
	case *ast.ForOfStatement:
		return stmt.Name == name ||
			expressionContainsIdentifier(stmt.Iterable, name) ||
			statementsContainIdentifier(stmt.Body, name)
	case *ast.ForInStatement:
		return stmt.Name == name ||
			expressionContainsIdentifier(stmt.Iterable, name) ||
			statementsContainIdentifier(stmt.Body, name)
	case *ast.SwitchStatement:
		if expressionContainsIdentifier(stmt.Discriminant, name) || statementsContainIdentifier(stmt.Default, name) {
			return true
		}
		for _, switchCase := range stmt.Cases {
			if expressionContainsIdentifier(switchCase.Test, name) || statementsContainIdentifier(switchCase.Consequent, name) {
				return true
			}
		}
		return false
	case *ast.LabeledStatement:
		return statementContainsIdentifier(stmt.Statement, name)
	case *ast.TryStatement:
		return stmt.CatchName == name ||
			statementsContainIdentifier(stmt.TryBody, name) ||
			statementsContainIdentifier(stmt.CatchBody, name) ||
			statementsContainIdentifier(stmt.FinallyBody, name)
	default:
		return false
	}
}

func patternContainsIdentifier(pattern ast.Pattern, name string) bool {
	switch pattern := pattern.(type) {
	case *ast.IdentifierPattern:
		return pattern.Name == name
	case *ast.ObjectPattern:
		if pattern.Rest == name {
			return true
		}
		for _, property := range pattern.Properties {
			if patternContainsIdentifier(property.Pattern, name) || expressionContainsIdentifier(property.Default, name) {
				return true
			}
		}
		return false
	case *ast.ArrayPattern:
		for _, element := range pattern.Elements {
			if element.Pattern != nil && patternContainsIdentifier(element.Pattern, name) {
				return true
			}
			if expressionContainsIdentifier(element.Default, name) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func expressionContainsIdentifier(expr ast.Expression, name string) bool {
	switch expr := expr.(type) {
	case nil:
		return false
	case *ast.Identifier:
		return expr.Name == name
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if expressionContainsIdentifier(property.KeyExpr, name) || expressionContainsIdentifier(property.Value, name) {
				return true
			}
		}
		return false
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if expressionContainsIdentifier(element, name) {
				return true
			}
		}
		return false
	case *ast.TemplateLiteral:
		for _, value := range expr.Values {
			if expressionContainsIdentifier(value, name) {
				return true
			}
		}
		return false
	case *ast.SpreadExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.BinaryExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.ComparisonExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.LogicalExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.NullishCoalesceExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.CommaExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.ConditionalExpression:
		return expressionContainsIdentifier(expr.Condition, name) ||
			expressionContainsIdentifier(expr.Consequent, name) ||
			expressionContainsIdentifier(expr.Alternative, name)
	case *ast.UnaryExpression:
		return expressionContainsIdentifier(expr.Right, name)
	case *ast.TypeofExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.TypeCheckExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.InstanceofExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.IndexExpression:
		return expressionContainsIdentifier(expr.Target, name) || expressionContainsIdentifier(expr.Index, name)
	case *ast.MemberExpression:
		return expressionContainsIdentifier(expr.Target, name)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
		return false
	case *ast.InvokeExpression:
		if expressionContainsIdentifier(expr.Callee, name) {
			return true
		}
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
		return false
	case *ast.NewExpression:
		if expressionContainsIdentifier(expr.Callee, name) {
			return true
		}
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
		return false
	case *ast.AwaitExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.YieldExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.CastExpression:
		return expressionContainsIdentifier(expr.Value, name)
	case *ast.FunctionExpression:
		for _, param := range expr.Params {
			if param.Name == name || (param.Pattern != nil && patternContainsIdentifier(param.Pattern, name)) || expressionContainsIdentifier(param.Default, name) {
				return true
			}
		}
		return statementsContainIdentifier(expr.Body, name) || expressionContainsIdentifier(expr.ExpressionBody, name)
	case *ast.ClosureExpression:
		return expressionContainsIdentifier(expr.Environment, name)
	case *ast.BoundSuperExpression:
		return expressionContainsIdentifier(expr.Receiver, name)
	default:
		return false
	}
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
	case lexer.TokenSlash:
		return ast.OperatorDiv
	case lexer.TokenBitAnd:
		return ast.OperatorBitAnd
	case lexer.TokenBitOr:
		return ast.OperatorBitOr
	case lexer.TokenBitXor:
		return ast.OperatorBitXor
	case lexer.TokenShiftLeft:
		return ast.OperatorShl
	case lexer.TokenShiftRight:
		return ast.OperatorShr
	default:
		return ast.OperatorUShr
	}
}

func isComparisonToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenStrictEq, lexer.TokenStrictNe, lexer.TokenLt, lexer.TokenLte, lexer.TokenGt, lexer.TokenGte, lexer.TokenInstanceof, lexer.TokenIs:
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

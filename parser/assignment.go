package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseExpressionStatement() (ast.Statement, error) {
	expr, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if isAssignmentToken(p.current.Type) {
		return p.parseAssignmentStatementTerminated(expr, true)
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return nil, err
	}
	return &ast.ExpressionStatement{BaseNode: baseOf(expr), Expression: expr}, nil
}

func (p *Parser) parseAssignmentStatement(target ast.Expression) (ast.Statement, error) {
	return p.parseAssignmentStatementTerminated(target, true)
}

func (p *Parser) parseAssignmentStatementTerminated(target ast.Expression, consumeTerminator bool) (ast.Statement, error) {
	if !isAssignmentTarget(target) {
		return nil, errorAtPosition(ast.PositionOf(target), "invalid assignment target")
	}
	operatorToken := p.current
	operator := assignmentOperator(operatorToken.Type)
	p.advance()
	value, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if consumeTerminator {
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
	}
	return &ast.AssignmentStatement{
		BaseNode: baseOf(target),
		Target:   target,
		Operator: operator,
		Value:    value,
	}, nil
}

func isAssignmentTarget(target ast.Expression) bool {
	switch target := target.(type) {
	case *ast.Identifier:
		return true
	case *ast.MemberExpression, *ast.IndexExpression:
		return !hasOptionalChain(target)
	default:
		return false
	}
}

func hasOptionalChain(expr ast.Expression) bool {
	switch expr := expr.(type) {
	case *ast.MemberExpression:
		return expr.Optional || hasOptionalChain(expr.Target)
	case *ast.IndexExpression:
		return expr.Optional || hasOptionalChain(expr.Target) || hasOptionalChain(expr.Index)
	case *ast.InvokeExpression:
		return expr.Optional || hasOptionalChain(expr.Callee)
	default:
		return false
	}
}

func isAssignmentToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenAssign,
		lexer.TokenAddAssign,
		lexer.TokenSubAssign,
		lexer.TokenMulAssign,
		lexer.TokenPowAssign,
		lexer.TokenDivAssign,
		lexer.TokenModAssign,
		lexer.TokenBitAndAssign,
		lexer.TokenBitOrAssign,
		lexer.TokenBitXorAssign,
		lexer.TokenShlAssign,
		lexer.TokenShrAssign,
		lexer.TokenUShrAssign,
		lexer.TokenNullishAssign,
		lexer.TokenOrAssign,
		lexer.TokenAndAssign:
		return true
	default:
		return false
	}
}

func assignmentOperator(tokenType lexer.TokenType) ast.AssignmentOperator {
	switch tokenType {
	case lexer.TokenAddAssign:
		return ast.AssignmentAddAssign
	case lexer.TokenSubAssign:
		return ast.AssignmentSubAssign
	case lexer.TokenMulAssign:
		return ast.AssignmentMulAssign
	case lexer.TokenPowAssign:
		return ast.AssignmentPowAssign
	case lexer.TokenDivAssign:
		return ast.AssignmentDivAssign
	case lexer.TokenModAssign:
		return ast.AssignmentModAssign
	case lexer.TokenBitAndAssign:
		return ast.AssignmentBitAndAssign
	case lexer.TokenBitOrAssign:
		return ast.AssignmentBitOrAssign
	case lexer.TokenBitXorAssign:
		return ast.AssignmentBitXorAssign
	case lexer.TokenShlAssign:
		return ast.AssignmentShlAssign
	case lexer.TokenShrAssign:
		return ast.AssignmentShrAssign
	case lexer.TokenUShrAssign:
		return ast.AssignmentUShrAssign
	case lexer.TokenNullishAssign:
		return ast.AssignmentNullishAssign
	case lexer.TokenOrAssign:
		return ast.AssignmentOrAssign
	case lexer.TokenAndAssign:
		return ast.AssignmentAndAssign
	default:
		return ast.AssignmentAssign
	}
}

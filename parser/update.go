package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parsePrefixUpdate(token lexer.Token) (ast.Expression, error) {
	operator := updateOperator(token.Type)
	p.advance()
	target, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if !isAssignmentTarget(target) {
		return nil, errorAtPosition(ast.PositionOf(target), "invalid update target")
	}
	return &ast.UpdateExpression{
		BaseNode: baseFrom(token),
		Operator: operator,
		Target:   target,
		Prefix:   true,
	}, nil
}

func (p *Parser) parsePostfixUpdate(target ast.Expression) (ast.Expression, error) {
	if !isAssignmentTarget(target) {
		return nil, errorAtPosition(ast.PositionOf(target), "invalid update target")
	}
	token := p.current
	operator := updateOperator(token.Type)
	p.advance()
	return &ast.UpdateExpression{
		BaseNode: baseOf(target),
		Operator: operator,
		Target:   target,
	}, nil
}

func updateOperator(tokenType lexer.TokenType) ast.UpdateOperator {
	if tokenType == lexer.TokenDecrement {
		return ast.UpdateDecrement
	}
	return ast.UpdateIncrement
}

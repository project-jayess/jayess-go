package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseBitwiseOr() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseBitwiseXor, lexer.TokenBitOr)
}

func (p *Parser) parseBitwiseXor() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseBitwiseAnd, lexer.TokenBitXor)
}

func (p *Parser) parseBitwiseAnd() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseComparison, lexer.TokenBitAnd)
}

func (p *Parser) parseComparison() (ast.Expression, error) {
	left, err := p.parseShift()
	if err != nil {
		return nil, err
	}
	for isComparisonToken(p.current.Type) {
		operator := p.current.Type
		p.advance()
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		if operator == lexer.TokenInstanceof {
			left = &ast.InstanceofExpression{BaseNode: baseOf(left), Left: left, Right: right}
			continue
		}
		left = &ast.ComparisonExpression{BaseNode: baseOf(left), Operator: comparisonOperator(operator), Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseShift() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseAdditive, lexer.TokenShiftLeft, lexer.TokenShiftRight, lexer.TokenUnsignedShift)
}

func (p *Parser) parseAdditive() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseMultiplicative, lexer.TokenPlus, lexer.TokenMinus)
}

func (p *Parser) parseMultiplicative() (ast.Expression, error) {
	return p.parseBinaryLeft(p.parseExponentiation, lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent)
}

func (p *Parser) parseExponentiation() (ast.Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if !p.match(lexer.TokenPower) {
		return left, nil
	}
	right, err := p.parseExponentiation()
	if err != nil {
		return nil, err
	}
	return &ast.BinaryExpression{BaseNode: baseOf(left), Operator: ast.OperatorPow, Left: left, Right: right}, nil
}

func (p *Parser) parseBinaryLeft(next func() (ast.Expression, error), tokenTypes ...lexer.TokenType) (ast.Expression, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for tokenMatches(p.current.Type, tokenTypes...) {
		operator := p.current.Type
		p.advance()
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{BaseNode: baseOf(left), Operator: binaryOperator(operator), Left: left, Right: right}
	}
	return left, nil
}

func tokenMatches(tokenType lexer.TokenType, expected ...lexer.TokenType) bool {
	for _, candidate := range expected {
		if tokenType == candidate {
			return true
		}
	}
	return false
}

func isComparisonToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenEq, lexer.TokenNe, lexer.TokenStrictEq, lexer.TokenStrictNe,
		lexer.TokenLt, lexer.TokenLte, lexer.TokenGt, lexer.TokenGte,
		lexer.TokenIn, lexer.TokenInstanceof:
		return true
	default:
		return false
	}
}

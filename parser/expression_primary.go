package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseUnary() (ast.Expression, error) {
	token := p.current
	switch token.Type {
	case lexer.TokenIncrement, lexer.TokenDecrement:
		return p.parsePrefixUpdate(token)
	case lexer.TokenTypeof:
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.TypeofExpression{BaseNode: baseFrom(token), Value: right}, nil
	case lexer.TokenAwait:
		return p.parseAwaitExpression(token)
	case lexer.TokenYield:
		return p.parseYieldExpression(token)
	case lexer.TokenDelete:
		return p.parseUnaryOperator(token, ast.OperatorDelete)
	case lexer.TokenVoid:
		return p.parseUnaryOperator(token, ast.OperatorVoid)
	case lexer.TokenBang:
		return p.parseUnaryOperator(token, ast.OperatorNot)
	case lexer.TokenBitNot:
		return p.parseUnaryOperator(token, ast.OperatorBitNot)
	case lexer.TokenPlus:
		return p.parseUnaryOperator(token, ast.OperatorPositive)
	case lexer.TokenMinus:
		return p.parseUnaryOperator(token, ast.OperatorNegate)
	default:
		return p.parsePostfix()
	}
}

func (p *Parser) parseUnaryOperator(token lexer.Token, operator ast.UnaryOperator) (ast.Expression, error) {
	p.advance()
	right, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return &ast.UnaryExpression{BaseNode: baseFrom(token), Operator: operator, Right: right}, nil
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	token := p.current
	if p.isUnsupportedAsyncLineTerminatorStart() {
		return nil, p.unsupportedAsyncLineTerminatorError()
	}
	if p.isUnsupportedAsyncGenericArrowTypeParametersStart() || p.isUnsupportedGenericArrowTypeParametersStart() {
		return nil, p.unsupportedGenericTypeParametersError()
	}
	if p.isUnsupportedAsyncArrowReturnTypeAnnotationStart() || p.isUnsupportedArrowReturnTypeAnnotationStart() {
		return nil, p.unsupportedArrowReturnAnnotationError()
	}
	if p.isAsyncArrowFunctionStart() {
		return p.parseAsyncArrowFunction()
	}
	if p.isAsyncFunctionStart() {
		return p.parseAsyncFunctionExpression()
	}
	if p.isArrowFunctionStart() {
		return p.parseArrowFunction()
	}
	if p.isNewTargetStart() {
		return p.parseNewTargetExpression()
	}
	if isNewExpressionStart(token) {
		return p.parseNewExpression()
	}
	if p.isImportMetaStart() {
		return p.parseImportMetaExpression()
	}
	if p.isDynamicImportStart() {
		return nil, p.unsupportedDynamicImportError()
	}
	if token.Type == lexer.TokenSlash {
		return nil, p.unsupportedRegularExpressionLiteralError()
	}
	if token.Type == lexer.TokenLt {
		return nil, p.unsupportedJSXOrAngleBracketSyntaxError()
	}
	switch token.Type {
	case lexer.TokenLBracket:
		return p.parseArrayLiteral(baseFrom(token))
	case lexer.TokenLBrace:
		return p.parseObjectLiteral(baseFrom(token))
	case lexer.TokenFunction:
		return p.parseFunctionExpression()
	}
	p.advance()

	switch token.Type {
	case lexer.TokenIdent:
		return &ast.Identifier{BaseNode: baseFrom(token), Name: token.Literal}, nil
	case lexer.TokenThis:
		return &ast.ThisExpression{BaseNode: baseFrom(token)}, nil
	case lexer.TokenSuper:
		return &ast.SuperExpression{BaseNode: baseFrom(token)}, nil
	case lexer.TokenNumber:
		return &ast.NumberLiteral{BaseNode: baseFrom(token), Value: token.Literal}, nil
	case lexer.TokenBigInt:
		return &ast.BigIntLiteral{BaseNode: baseFrom(token), Value: token.Literal}, nil
	case lexer.TokenString:
		return &ast.StringLiteral{BaseNode: baseFrom(token), Value: token.Literal}, nil
	case lexer.TokenTemplate:
		return p.parseTemplateLiteral(token)
	case lexer.TokenTrue:
		return &ast.BooleanLiteral{BaseNode: baseFrom(token), Value: true}, nil
	case lexer.TokenFalse:
		return &ast.BooleanLiteral{BaseNode: baseFrom(token), Value: false}, nil
	case lexer.TokenNull:
		return &ast.NullLiteral{BaseNode: baseFrom(token)}, nil
	case lexer.TokenUndefined:
		return &ast.UndefinedLiteral{BaseNode: baseFrom(token)}, nil
	case lexer.TokenLParen:
		expr, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		if err := p.expect(lexer.TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	default:
		return nil, errorAtToken(token, "expected expression, got %s", token.Type)
	}
}

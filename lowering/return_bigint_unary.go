package lowering

import (
	"math/big"
	"strings"

	"jayess-go/ast"
)

func evaluateBigIntUnary(expression *ast.UnaryExpression, bindings returnScope) (string, bool) {
	switch expression.Operator {
	case ast.OperatorNegate:
		return evaluateBigIntNegate(expression.Right, bindings)
	case ast.OperatorBitNot:
		return evaluateBigIntBitNot(expression.Right, bindings)
	default:
		return "", false
	}
}

func evaluateBigIntNegate(expression ast.Expression, bindings returnScope) (string, bool) {
	value, ok := evaluateBigIntValue(expression, bindings)
	if !ok {
		return "", false
	}
	if value == "0" {
		return value, true
	}
	if strings.HasPrefix(value, "-") {
		return strings.TrimPrefix(value, "-"), true
	}
	return "-" + value, true
}

func evaluateBigIntBitNot(expression ast.Expression, bindings returnScope) (string, bool) {
	value, ok := evaluateBigIntValue(expression, bindings)
	if !ok {
		return "", false
	}
	parsed, ok := parseBigIntValue(value)
	if !ok {
		return "", false
	}
	return new(big.Int).Not(parsed).String(), true
}

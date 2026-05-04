package lowering

import (
	"math/big"

	"jayess-go/ast"
)

func evaluateBigIntUpdateExpression(expression *ast.UpdateExpression, scope returnScope) (string, bool) {
	identifier, ok := expression.Target.(*ast.Identifier)
	if !ok || identifier.Name == "" {
		return "", false
	}
	current, ok := scope.bigints[identifier.Name]
	if !ok {
		return "", false
	}
	next, ok := updatedBigIntValue(expression.Operator, current)
	if !ok {
		return "", false
	}
	scope.bigints[identifier.Name] = next
	if expression.Prefix {
		return next, true
	}
	return current, true
}

func updatedBigIntValue(operator ast.UpdateOperator, current string) (string, bool) {
	value, ok := parseBigIntValue(current)
	if !ok {
		return "", false
	}
	one := big.NewInt(1)
	switch operator {
	case ast.UpdateIncrement:
		value.Add(value, one)
	case ast.UpdateDecrement:
		value.Sub(value, one)
	default:
		return "", false
	}
	return value.String(), true
}

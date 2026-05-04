package lowering

import (
	"math/big"
	"strconv"
	"strings"

	"jayess-go/ast"
)

func evaluateLooseStringBigIntEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseStringBigInt, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseStringBigInt, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func evaluateLooseNumberBigIntEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseNumberBigInt, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseNumberBigInt, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func evaluateLooseBooleanBigIntEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseBooleanBigInt, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseBooleanBigInt, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func compareLooseStringBigInt(stringExpression ast.Expression, bigIntExpression ast.Expression, bindings returnScope) (bool, bool) {
	text, ok := evaluateStringExpression(stringExpression, bindings)
	if !ok {
		return false, false
	}
	bigIntValue, ok := evaluateBigIntValue(bigIntExpression, bindings)
	if !ok {
		return false, false
	}
	stringValue, ok := parseLooseStringBigInt(text)
	if !ok {
		return false, true
	}
	return stringValue == bigIntValue, true
}

func compareLooseNumberBigInt(numberExpression ast.Expression, bigIntExpression ast.Expression, bindings returnScope) (bool, bool) {
	number, ok := evaluateIntExpression(numberExpression, bindings)
	if !ok {
		return false, false
	}
	bigIntValue, ok := evaluateBigIntValue(bigIntExpression, bindings)
	if !ok {
		return false, false
	}
	return strconv.Itoa(number) == bigIntValue, true
}

func compareLooseBooleanBigInt(boolExpression ast.Expression, bigIntExpression ast.Expression, bindings returnScope) (bool, bool) {
	boolean, ok := evaluateBoolValue(boolExpression, bindings)
	if !ok {
		return false, false
	}
	bigIntValue, ok := evaluateBigIntValue(bigIntExpression, bindings)
	if !ok {
		return false, false
	}
	if boolean {
		return bigIntValue == "1", true
	}
	return bigIntValue == "0", true
}

func parseLooseStringBigInt(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "0", true
	}
	value, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return "", false
	}
	return value.String(), true
}

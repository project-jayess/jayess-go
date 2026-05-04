package lowering

import (
	"strconv"
	"strings"

	"jayess-go/ast"
)

func evaluateLooseStringNumberEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseStringNumber, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseStringNumber, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func evaluateLooseStringBooleanEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseStringBoolean, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseStringBoolean, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func evaluateLooseBooleanNumberEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	if value, ok := evaluateLooseProbe(compareLooseBooleanNumber, leftExpression, rightExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := evaluateLooseProbe(compareLooseBooleanNumber, rightExpression, leftExpression, bindings); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func evaluateLooseProbe(probe func(ast.Expression, ast.Expression, returnScope) (bool, bool), leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	value, ok := probe(leftExpression, rightExpression, next)
	if !ok {
		return false, false
	}
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func evaluateLooseNullishPrimitiveMismatch(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorEq && operator != ast.OperatorNe {
		return false, false
	}
	left, leftOK := evaluatePrimitiveKind(leftExpression, bindings)
	right, rightOK := evaluatePrimitiveKind(rightExpression, bindings)
	if !leftOK || !rightOK || left == right {
		return false, false
	}
	if left != "nullish" && right != "nullish" {
		return false, false
	}
	return compareEquality(operator, false), true
}

func compareLooseBooleanNumber(boolExpression ast.Expression, numberExpression ast.Expression, bindings returnScope) (bool, bool) {
	boolean, ok := evaluateBoolValue(boolExpression, bindings)
	if !ok {
		return false, false
	}
	number, ok := evaluateIntExpression(numberExpression, bindings)
	if !ok {
		return false, false
	}
	boolNumber := 0
	if boolean {
		boolNumber = 1
	}
	return boolNumber == number, true
}

func compareLooseStringBoolean(stringExpression ast.Expression, boolExpression ast.Expression, bindings returnScope) (bool, bool) {
	text, ok := evaluateStringExpression(stringExpression, bindings)
	if !ok {
		return false, false
	}
	boolean, ok := evaluateBoolValue(boolExpression, bindings)
	if !ok {
		return false, false
	}
	stringNumber, ok := parseLooseStringNumber(text)
	if !ok {
		return false, true
	}
	boolNumber := 0.0
	if boolean {
		boolNumber = 1
	}
	return stringNumber == boolNumber, true
}

func compareLooseStringNumber(stringExpression ast.Expression, numberExpression ast.Expression, bindings returnScope) (bool, bool) {
	text, ok := evaluateStringExpression(stringExpression, bindings)
	if !ok {
		return false, false
	}
	number, ok := evaluateIntExpression(numberExpression, bindings)
	if !ok {
		return false, false
	}
	stringNumber, ok := parseLooseStringNumber(text)
	if !ok {
		return false, true
	}
	return stringNumber == float64(number), true
}

func parseLooseStringNumber(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, true
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

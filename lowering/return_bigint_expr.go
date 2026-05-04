package lowering

import (
	"math/big"
	"strings"

	"jayess-go/ast"
)

func evaluateBigIntTruthiness(expression ast.Expression) (bool, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return false, false
	}
	return isTruthyBigIntValue(literal.Value), true
}

func evaluateBigIntStringCoercion(expression ast.Expression) (string, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return "", false
	}
	return literal.Value, true
}

func evaluateBigIntLiteralValue(expression ast.Expression) (string, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return "", false
	}
	return normalizeBigIntValue(literal.Value), true
}

func evaluateBigIntValue(expression ast.Expression, bindings returnScope) (string, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.bigints[expr.Name]
		return value, ok
	case *ast.MemberExpression:
		return evaluateBigIntMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		return evaluateBigIntIndexExpression(expr, bindings)
	case *ast.UnaryExpression:
		return evaluateBigIntUnary(expr, bindings)
	case *ast.UpdateExpression:
		return evaluateBigIntUpdateExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateBigIntIIFEExpression(expr, bindings)
	case *ast.BinaryExpression:
		return evaluateBigIntBinary(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.LogicalExpression:
		return evaluateBigIntLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return "", false
		}
		var value string
		if condition {
			value, ok = evaluateBigIntValue(expr.Consequent, next)
		} else {
			value, ok = evaluateBigIntValue(expr.Alternative, next)
		}
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return "", false
		}
		value, ok := evaluateBigIntValue(expr.Right, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateBigIntValue(expr.Right, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateBigIntValue(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return "", false
	default:
		return evaluateBigIntLiteralValue(expression)
	}
}

func evaluateBigIntBinary(operator ast.BinaryOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (string, bool) {
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return "", false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return "", false
	}
	value, ok := evaluateBigIntBinaryValue(operator, left, right)
	if !ok {
		return "", false
	}
	return value, true
}

func evaluateBigIntBinaryValue(operator ast.BinaryOperator, leftValue string, rightValue string) (string, bool) {
	left, ok := parseBigIntValue(leftValue)
	if !ok {
		return "", false
	}
	right, ok := parseBigIntValue(rightValue)
	if !ok {
		return "", false
	}
	result := new(big.Int)
	switch operator {
	case ast.OperatorAdd:
		result.Add(left, right)
	case ast.OperatorSub:
		result.Sub(left, right)
	case ast.OperatorMul:
		result.Mul(left, right)
	case ast.OperatorDiv:
		if right.Sign() == 0 {
			return "", false
		}
		result.Quo(left, right)
	case ast.OperatorMod:
		if right.Sign() == 0 {
			return "", false
		}
		result.Rem(left, right)
	case ast.OperatorPow:
		if right.Sign() < 0 || !right.IsInt64() || right.Int64() > 64 {
			return "", false
		}
		result.Exp(left, right, nil)
	case ast.OperatorBitAnd:
		result.And(left, right)
	case ast.OperatorBitOr:
		result.Or(left, right)
	case ast.OperatorBitXor:
		result.Xor(left, right)
	case ast.OperatorShl:
		count, ok := bigIntShiftCount(right)
		if !ok {
			return "", false
		}
		result.Lsh(left, count)
	case ast.OperatorShr:
		count, ok := bigIntShiftCount(right)
		if !ok {
			return "", false
		}
		result.Rsh(left, count)
	default:
		return "", false
	}
	return result.String(), true
}

func bigIntShiftCount(value *big.Int) (uint, bool) {
	if value.Sign() < 0 || !value.IsUint64() || value.Uint64() > 1024 {
		return 0, false
	}
	return uint(value.Uint64()), true
}

func assignmentBigIntBinaryOperator(operator ast.AssignmentOperator) (ast.BinaryOperator, bool) {
	switch operator {
	case ast.AssignmentAddAssign:
		return ast.OperatorAdd, true
	case ast.AssignmentSubAssign:
		return ast.OperatorSub, true
	case ast.AssignmentMulAssign:
		return ast.OperatorMul, true
	case ast.AssignmentDivAssign:
		return ast.OperatorDiv, true
	case ast.AssignmentModAssign:
		return ast.OperatorMod, true
	case ast.AssignmentPowAssign:
		return ast.OperatorPow, true
	case ast.AssignmentBitAndAssign:
		return ast.OperatorBitAnd, true
	case ast.AssignmentBitOrAssign:
		return ast.OperatorBitOr, true
	case ast.AssignmentBitXorAssign:
		return ast.OperatorBitXor, true
	case ast.AssignmentShlAssign:
		return ast.OperatorShl, true
	case ast.AssignmentShrAssign:
		return ast.OperatorShr, true
	default:
		return "", false
	}
}

func parseBigIntValue(value string) (*big.Int, bool) {
	parsed, ok := new(big.Int).SetString(value, 10)
	return parsed, ok
}

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

func evaluateBigIntLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if left, ok := evaluateBigIntValue(leftExpression, next); ok {
		leftTruthy := isTruthyBigIntValue(left)
		switch operator {
		case ast.OperatorAnd:
			if !leftTruthy {
				replaceReturnScopeBindings(bindings, next)
				return left, true
			}
			right, ok := evaluateBigIntValue(rightExpression, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return right, true
		case ast.OperatorOr:
			if leftTruthy {
				replaceReturnScopeBindings(bindings, next)
				return left, true
			}
			right, ok := evaluateBigIntValue(rightExpression, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return right, true
		default:
			return "", false
		}
	}
	next = bindings.clone()
	leftTruthy, ok := evaluateBoolExpression(leftExpression, next)
	if !ok {
		return "", false
	}
	switch operator {
	case ast.OperatorAnd:
		if !leftTruthy {
			return "", false
		}
		right, ok := evaluateBigIntValue(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	case ast.OperatorOr:
		if leftTruthy {
			return "", false
		}
		right, ok := evaluateBigIntValue(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	default:
		return "", false
	}
}

func normalizeBigIntValue(value string) string {
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign = "-"
		value = strings.TrimPrefix(value, "-")
	}
	normalized := strings.TrimLeft(value, "0")
	if normalized == "" {
		return "0"
	}
	return sign + normalized
}

func isTruthyBigIntValue(value string) bool {
	return normalizeBigIntValue(value) != "0"
}

func evaluateBigIntEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return false, false
	}
	return compareEquality(operator, left == right), true
}

func evaluateBigIntRelationalComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if isEqualityOperator(operator) {
		return false, false
	}
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return false, false
	}
	order := compareBigIntLiteralValues(left, right)
	return compareBigIntRelationalOrder(operator, order), true
}

func compareBigIntLiteralValues(left string, right string) int {
	leftNegative := strings.HasPrefix(left, "-")
	rightNegative := strings.HasPrefix(right, "-")
	if leftNegative && !rightNegative {
		return -1
	}
	if !leftNegative && rightNegative {
		return 1
	}
	if leftNegative {
		return -compareUnsignedBigIntLiteralValues(strings.TrimPrefix(left, "-"), strings.TrimPrefix(right, "-"))
	}
	return compareUnsignedBigIntLiteralValues(left, right)
}

func compareUnsignedBigIntLiteralValues(left string, right string) int {
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

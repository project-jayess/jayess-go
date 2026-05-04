package lowering

import (
	"math"
	"strconv"

	"jayess-go/ast"
)

func evaluateIntExpression(expression ast.Expression, bindings returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.ints[expr.Name]
		return value, ok
	case *ast.NumberLiteral:
		value, err := strconv.ParseFloat(expr.Value, 64)
		if err != nil {
			return 0, false
		}
		return int(value), true
	case *ast.MemberExpression:
		return evaluateIntMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		return evaluateIntIndexExpression(expr, bindings)
	case *ast.UnaryExpression:
		value, ok := evaluateUnaryNumericOperand(expr.Operator, expr.Right, bindings)
		if !ok {
			return 0, false
		}
		switch expr.Operator {
		case ast.OperatorNegate:
			return -value, true
		case ast.OperatorPositive:
			return value, true
		case ast.OperatorBitNot:
			return ^value, true
		default:
			return 0, false
		}
	case *ast.UpdateExpression:
		return evaluateIntUpdateExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateIntIIFEExpression(expr, bindings)
	case *ast.BinaryExpression:
		left, ok := evaluateBinaryNumericOperand(expr.Operator, expr.Left, bindings)
		if !ok {
			return 0, false
		}
		right, ok := evaluateBinaryNumericOperand(expr.Operator, expr.Right, bindings)
		if !ok {
			return 0, false
		}
		return evaluateIntBinary(expr.Operator, left, right)
	case *ast.LogicalExpression:
		return evaluateIntLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var value int
		if condition {
			value, ok = evaluateIntExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateIntExpression(expr.Alternative, next)
		}
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return 0, false
		}
		value, ok := evaluateIntExpression(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateIntExpression(expr.Right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateIntExpression(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return 0, false
	default:
		return 0, false
	}
}

func evaluateBinaryNumericOperand(operator ast.BinaryOperator, expression ast.Expression, bindings returnScope) (int, bool) {
	if operator == ast.OperatorAdd {
		next := bindings.clone()
		if _, ok := evaluateStringExpression(expression, next); ok {
			return 0, false
		}
	}
	return evaluateNumericCoercion(expression, bindings)
}

func evaluateUnaryNumericOperand(operator ast.UnaryOperator, expression ast.Expression, bindings returnScope) (int, bool) {
	switch operator {
	case ast.OperatorNegate, ast.OperatorPositive, ast.OperatorBitNot:
		return evaluateNumericCoercion(expression, bindings)
	default:
		return 0, false
	}
}

func evaluateIntComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateIntExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateIntExpression(rightExpression, bindings)
	if !ok {
		return false, false
	}
	switch operator {
	case ast.OperatorEq, ast.OperatorStrictEq:
		return left == right, true
	case ast.OperatorNe, ast.OperatorStrictNe:
		return left != right, true
	case ast.OperatorLt:
		return left < right, true
	case ast.OperatorLte:
		return left <= right, true
	case ast.OperatorGt:
		return left > right, true
	case ast.OperatorGte:
		return left >= right, true
	default:
		return false, false
	}
}

func evaluateIntAssignment(operator ast.AssignmentOperator, left int, right int) (int, bool) {
	switch operator {
	case ast.AssignmentAddAssign:
		return left + right, true
	case ast.AssignmentSubAssign:
		return left - right, true
	case ast.AssignmentMulAssign:
		return left * right, true
	case ast.AssignmentPowAssign:
		return int(math.Pow(float64(left), float64(right))), true
	case ast.AssignmentDivAssign:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ast.AssignmentModAssign:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ast.AssignmentBitAndAssign:
		return left & right, true
	case ast.AssignmentBitOrAssign:
		return left | right, true
	case ast.AssignmentBitXorAssign:
		return left ^ right, true
	case ast.AssignmentShlAssign:
		return left << shiftCount(right), true
	case ast.AssignmentShrAssign:
		return left >> shiftCount(right), true
	case ast.AssignmentUShrAssign:
		return int(uint32(left) >> shiftCount(right)), true
	default:
		return 0, false
	}
}

func evaluateIntBinary(operator ast.BinaryOperator, left int, right int) (int, bool) {
	switch operator {
	case ast.OperatorAdd:
		return left + right, true
	case ast.OperatorSub:
		return left - right, true
	case ast.OperatorMul:
		return left * right, true
	case ast.OperatorDiv:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ast.OperatorMod:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ast.OperatorPow:
		return int(math.Pow(float64(left), float64(right))), true
	case ast.OperatorBitAnd:
		return left & right, true
	case ast.OperatorBitOr:
		return left | right, true
	case ast.OperatorBitXor:
		return left ^ right, true
	case ast.OperatorShl:
		return left << shiftCount(right), true
	case ast.OperatorShr:
		return left >> shiftCount(right), true
	case ast.OperatorUShr:
		return int(uint32(left) >> shiftCount(right)), true
	default:
		return 0, false
	}
}

func shiftCount(value int) uint {
	return uint(value) & 31
}

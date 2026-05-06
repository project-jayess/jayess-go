package lowering

import "jayess-go/ast"

func evaluateBoolLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateBoolExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	switch operator {
	case ast.OperatorAnd:
		if !left {
			return false, true
		}
		right, ok := evaluateBoolExpression(rightExpression, bindings)
		return right, ok
	case ast.OperatorOr:
		if left {
			return true, true
		}
		right, ok := evaluateBoolExpression(rightExpression, bindings)
		return right, ok
	default:
		return false, false
	}
}

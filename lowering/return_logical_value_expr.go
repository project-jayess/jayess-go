package lowering

import "jayess-go/ast"

func evaluateIntLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (int, bool) {
	next := bindings.clone()
	if left, ok := evaluateIntExpression(leftExpression, next); ok {
		leftTruthy := left != 0
		switch operator {
		case ast.OperatorAnd:
			if !leftTruthy {
				replaceReturnScopeBindings(bindings, next)
				return left, true
			}
			right, ok := evaluateIntExpression(rightExpression, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(bindings, next)
			return right, true
		case ast.OperatorOr:
			if leftTruthy {
				replaceReturnScopeBindings(bindings, next)
				return left, true
			}
			right, ok := evaluateIntExpression(rightExpression, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(bindings, next)
			return right, true
		default:
			return 0, false
		}
	}
	next = bindings.clone()
	leftTruthy, ok := evaluateBoolExpression(leftExpression, next)
	if !ok {
		return 0, false
	}
	switch operator {
	case ast.OperatorAnd:
		if !leftTruthy {
			return 0, false
		}
		right, ok := evaluateIntExpression(rightExpression, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	case ast.OperatorOr:
		if leftTruthy {
			return 0, false
		}
		right, ok := evaluateIntExpression(rightExpression, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	default:
		return 0, false
	}
}

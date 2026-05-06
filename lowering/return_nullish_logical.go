package lowering

import "jayess-go/ast"

func evaluateNullishLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (returnNullishKind, bool) {
	next := bindings.clone()
	if left, ok := evaluateNullishExpression(leftExpression, next); ok {
		switch operator {
		case ast.OperatorAnd:
			replaceReturnScopeBindings(bindings, next)
			return left, true
		case ast.OperatorOr:
			right, ok := evaluateNullishExpression(rightExpression, next)
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
		right, ok := evaluateNullishExpression(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	case ast.OperatorOr:
		if leftTruthy {
			return "", false
		}
		right, ok := evaluateNullishExpression(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	default:
		return "", false
	}
}

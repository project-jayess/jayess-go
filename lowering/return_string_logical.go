package lowering

import "jayess-go/ast"

func evaluateStringLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if left, ok := evaluateStringExpression(leftExpression, next); ok {
		leftTruthy := left != ""
		switch operator {
		case ast.OperatorAnd:
			if !leftTruthy {
				replaceReturnScopeBindings(bindings, next)
				return left, true
			}
			right, ok := evaluateStringExpression(rightExpression, next)
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
			right, ok := evaluateStringExpression(rightExpression, next)
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
		right, ok := evaluateStringExpression(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	case ast.OperatorOr:
		if leftTruthy {
			return "", false
		}
		right, ok := evaluateStringExpression(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return right, true
	default:
		return "", false
	}
}

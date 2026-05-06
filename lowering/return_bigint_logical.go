package lowering

import "jayess-go/ast"

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

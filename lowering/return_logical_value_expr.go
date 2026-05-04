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

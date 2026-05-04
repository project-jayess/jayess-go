package lowering

import "jayess-go/ast"

func evaluateNewObjectExpression(expression *ast.NewExpression, bindings returnScope) bool {
	next := bindings.clone()
	if !evaluateEmptyConstructorExpression(expression.Callee, next) {
		return false
	}
	if !evaluateArgumentList(expression.Arguments, next) {
		return false
	}
	replaceReturnScopeBindings(bindings, next)
	return true
}

func evaluateEmptyConstructorExpression(expression ast.Expression, bindings returnScope) bool {
	switch expr := expression.(type) {
	case *ast.FunctionExpression:
		return len(expr.Body) == 0 && expr.ExpressionBody == nil
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return false
		}
		if !evaluateEmptyConstructorExpression(expr.Right, next) {
			return false
		}
		replaceReturnScopeBindings(bindings, next)
		return true
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false
		}
		var matched bool
		if condition {
			matched = evaluateEmptyConstructorExpression(expr.Consequent, next)
		} else {
			matched = evaluateEmptyConstructorExpression(expr.Alternative, next)
		}
		if !matched {
			return false
		}
		replaceReturnScopeBindings(bindings, next)
		return true
	default:
		return false
	}
}

func evaluateArgumentList(arguments []ast.Expression, bindings returnScope) bool {
	for _, argument := range arguments {
		if !evaluateArgumentExpression(argument, bindings) {
			return false
		}
	}
	return true
}

func evaluateArgumentExpression(argument ast.Expression, bindings returnScope) bool {
	spread, ok := argument.(*ast.SpreadExpression)
	if !ok {
		return evaluateDiscardExpression(argument, bindings)
	}
	_, ok = evaluateArrayLiteralElements(spread.Value, bindings)
	return ok
}

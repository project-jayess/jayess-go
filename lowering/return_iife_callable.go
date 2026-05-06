package lowering

import "jayess-go/ast"

func evaluateCallableReturnExpression(expression ast.Expression, bindings returnScope) (iifeBody, bool) {
	switch expr := expression.(type) {
	case *ast.FunctionExpression:
		return evaluateFunctionCallableReturnExpression(expr, bindings)
	case *ast.CommaExpression:
		return evaluateCommaCallableReturnExpression(expr, bindings)
	case *ast.ConditionalExpression:
		return evaluateConditionalCallableReturnExpression(expr, bindings)
	default:
		return iifeBody{}, false
	}
}

func evaluateFunctionCallableReturnExpression(expr *ast.FunctionExpression, bindings returnScope) (iifeBody, bool) {
	if expr.ExpressionBody != nil {
		return iifeBody{expression: expr.ExpressionBody, selfName: expr.Name, params: expr.Params}, true
	}
	if len(expr.Body) < 1 {
		return iifeBody{expression: &ast.UndefinedLiteral{}, selfName: expr.Name, params: expr.Params}, true
	}
	prefix := expr.Body
	expression := ast.Expression(&ast.UndefinedLiteral{})
	if statement, ok := expr.Body[len(expr.Body)-1].(*ast.ReturnStatement); ok {
		prefix = expr.Body[:len(expr.Body)-1]
		if statement.Value != nil {
			expression = statement.Value
		}
	}
	locals, shadows, funcLocals, funcShadows, ok := collectIIFEPrefixBindings(prefix, expr.Name, expr.Params, bindings)
	if !ok {
		return iifeBody{}, false
	}
	return iifeBody{
		expression:  expression,
		selfName:    expr.Name,
		params:      expr.Params,
		prefix:      prefix,
		locals:      locals,
		shadows:     shadows,
		funcLocals:  funcLocals,
		funcShadows: funcShadows,
	}, true
}

func evaluateCommaCallableReturnExpression(expr *ast.CommaExpression, bindings returnScope) (iifeBody, bool) {
	next := bindings.clone()
	if !evaluateDiscardExpression(expr.Left, next) {
		return iifeBody{}, false
	}
	body, ok := evaluateCallableReturnExpression(expr.Right, next)
	if !ok {
		return iifeBody{}, false
	}
	replaceReturnScopeBindings(bindings, next)
	return body, true
}

func evaluateConditionalCallableReturnExpression(expr *ast.ConditionalExpression, bindings returnScope) (iifeBody, bool) {
	next := bindings.clone()
	condition, ok := evaluateBoolExpression(expr.Condition, next)
	if !ok {
		return iifeBody{}, false
	}
	var body iifeBody
	if condition {
		body, ok = evaluateCallableReturnExpression(expr.Consequent, next)
	} else {
		body, ok = evaluateCallableReturnExpression(expr.Alternative, next)
	}
	if !ok {
		return iifeBody{}, false
	}
	replaceReturnScopeBindings(bindings, next)
	return body, true
}

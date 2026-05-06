package lowering

import "jayess-go/ast"

type iifeBody struct {
	expression  ast.Expression
	selfName    string
	params      []ast.Parameter
	prefix      []ast.Statement
	locals      []string
	shadows     []string
	funcLocals  []string
	funcShadows []string
}

func evaluateIntIIFEExpression(expression ast.Expression, bindings returnScope) (int, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateIntExpression)
}

func evaluateStringIIFEExpression(expression ast.Expression, bindings returnScope) (string, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateStringExpression)
}

func evaluateBigIntIIFEExpression(expression ast.Expression, bindings returnScope) (string, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateBigIntValue)
}

func evaluateBoolIIFEExpression(expression ast.Expression, bindings returnScope) (bool, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateBoolExpression)
}

func evaluateNullishIIFEExpression(expression ast.Expression, bindings returnScope) (returnNullishKind, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateNullishExpression)
}

func evaluateFunctionIIFEExpression(expression ast.Expression, bindings returnScope) (int, bool) {
	return evaluateIIFEValue(expression, bindings, materializeFunctionIdentity)
}

func evaluateObjectIIFEExpression(expression ast.Expression, bindings returnScope) (int, bool) {
	return evaluateIIFEValue(expression, bindings, materializeObjectIdentity)
}

func evaluateArrayIIFELiteralElements(expression ast.Expression, bindings returnScope) ([]returnArrayElement, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateArrayLiteralElements)
}

func evaluateObjectIIFELiteralProperties(expression ast.Expression, bindings returnScope) (map[string]returnArrayElement, bool) {
	return evaluateIIFEValue(expression, bindings, evaluateObjectLiteralProperties)
}

func evaluateIIFEValue[T any](expression ast.Expression, bindings returnScope, evaluate func(ast.Expression, returnScope) (T, bool)) (T, bool) {
	var zero T
	call, ok := expression.(*ast.InvokeExpression)
	if !ok {
		return zero, false
	}
	next := bindings.clone()
	body, ok := evaluateIIFEBody(call, next)
	if !ok {
		return zero, false
	}
	shadowScope, ok := bindIIFEParameters(body, call.Arguments, next)
	if !ok {
		return zero, false
	}
	hoistIIFEVarBindings(body, next)
	if len(body.prefix) > 0 {
		if statementsContainReturn(body.prefix) {
			return zero, false
		}
		if applyNonReturnStatements(body.prefix, next) != nonReturnFlowNone {
			return zero, false
		}
	}
	value, ok := evaluate(body.expression, next)
	if !ok {
		return zero, false
	}
	clearIIFELocals(body.locals, next)
	clearIIFELocals(body.funcLocals, next)
	restoreIIFEShadows(body.shadows, shadowScope, next)
	restoreIIFEShadows(body.funcShadows, shadowScope, next)
	restoreIIFESelfName(body, shadowScope, next)
	clearIIFEParameters(body.params, next)
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func evaluateIIFEBody(call *ast.InvokeExpression, bindings returnScope) (iifeBody, bool) {
	if call.Optional {
		return iifeBody{}, false
	}
	body, ok := evaluateCallableReturnExpression(call.Callee, bindings)
	if !ok {
		return iifeBody{}, false
	}
	return body, true
}

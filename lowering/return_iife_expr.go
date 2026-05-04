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

func evaluateCallableReturnExpression(expression ast.Expression, bindings returnScope) (iifeBody, bool) {
	switch expr := expression.(type) {
	case *ast.FunctionExpression:
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
		return iifeBody{expression: expression, selfName: expr.Name, params: expr.Params, prefix: prefix, locals: locals, shadows: shadows, funcLocals: funcLocals, funcShadows: funcShadows}, true
	case *ast.CommaExpression:
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
	case *ast.ConditionalExpression:
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
	default:
		return iifeBody{}, false
	}
}

func bindIIFEParameters(body iifeBody, args []ast.Expression, bindings returnScope) (returnScope, bool) {
	params := body.params
	for _, param := range params {
		if !canBindIIFEParameter(param, bindings) {
			return returnScope{}, false
		}
	}
	defaults := make([]bool, len(params))
	for index, param := range params {
		needsDefault, ok := bindIIFEParameter(param, args, index, bindings)
		if !ok {
			return returnScope{}, false
		}
		defaults[index] = needsDefault
	}
	if len(args) > len(params) {
		if !evaluateArgumentList(args[len(params):], bindings) {
			return returnScope{}, false
		}
	}
	shadowScope := bindings.clone()
	bindIIFESelfName(body, bindings)
	for index, param := range params {
		if defaults[index] && !applyScalarVariableDecl(param.Name, param.Default, bindings) {
			return returnScope{}, false
		}
	}
	return shadowScope, true
}

func bindIIFEParameter(param ast.Parameter, args []ast.Expression, index int, bindings returnScope) (bool, bool) {
	if index >= len(args) {
		return bindIIFEMissingParameter(param, bindings)
	}
	if param.Default != nil {
		nullish, ok := evaluateIIFEArgumentNullish(args[index], bindings)
		if ok {
			clearReturnScopeBinding(bindings, param.Name)
			bindings.nullish[param.Name] = nullish
			return nullish == returnUndefinedKind, true
		}
	}
	return false, applyScalarVariableDecl(param.Name, args[index], bindings)
}

func bindIIFEMissingParameter(param ast.Parameter, bindings returnScope) (bool, bool) {
	clearReturnScopeBinding(bindings, param.Name)
	bindings.nullish[param.Name] = returnUndefinedKind
	return param.Default != nil, true
}

func evaluateIIFEArgumentNullish(argument ast.Expression, bindings returnScope) (returnNullishKind, bool) {
	next := bindings.clone()
	value, ok := evaluateNullishExpression(argument, next)
	if !ok {
		return "", false
	}
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func canBindIIFEParameter(param ast.Parameter, bindings returnScope) bool {
	if param.Name == "" || param.Rest || param.Pattern == nil {
		return false
	}
	if _, ok := param.Pattern.(*ast.BindingName); !ok {
		return false
	}
	return !bindingKnown(bindings, param.Name)
}

func bindIIFESelfName(body iifeBody, bindings returnScope) {
	if body.selfName == "" {
		return
	}
	if iifeParameterNames(body.params)[body.selfName] {
		return
	}
	assignFunctionIdentity(bindings, body.selfName, allocateFunctionIdentity(bindings))
}

func restoreIIFESelfName(body iifeBody, source returnScope, target returnScope) {
	if body.selfName == "" {
		return
	}
	if bindingKnown(source, body.selfName) {
		restoreIIFEShadow(body.selfName, source, target)
		return
	}
	clearReturnScopeBinding(target, body.selfName)
	delete(target.funcSeq, body.selfName)
	delete(target.objectSeq, body.selfName)
}

func iifeParameterNames(params []ast.Parameter) map[string]bool {
	names := make(map[string]bool, len(params))
	for _, param := range params {
		if param.Name != "" {
			names[param.Name] = true
		}
	}
	return names
}

func collectIIFEPrefixBindings(prefix []ast.Statement, selfName string, params []ast.Parameter, bindings returnScope) ([]string, []string, []string, []string, bool) {
	paramNames := iifeParameterNames(params)
	seenLocals := make(map[string]bool)
	var locals []string
	var shadows []string
	var funcLocals []string
	var funcShadows []string
	for _, statement := range prefix {
		decl, ok := statement.(*ast.FunctionDecl)
		if !ok || decl.Name == "" {
			continue
		}
		if paramNames[decl.Name] || seenLocals[decl.Name] {
			continue
		}
		seenLocals[decl.Name] = true
		if bindingKnown(bindings, decl.Name) {
			funcShadows = append(funcShadows, decl.Name)
		} else {
			funcLocals = append(funcLocals, decl.Name)
		}
	}
	for _, statement := range prefix {
		decl, ok := statement.(*ast.VariableDecl)
		if !ok || decl.Name == "" {
			continue
		}
		if paramNames[decl.Name] {
			continue
		}
		if decl.Name == selfName {
			continue
		}
		if seenLocals[decl.Name] {
			continue
		}
		seenLocals[decl.Name] = true
		if bindingKnown(bindings, decl.Name) {
			shadows = append(shadows, decl.Name)
		} else {
			locals = append(locals, decl.Name)
		}
	}
	return locals, shadows, funcLocals, funcShadows, true
}

func hoistIIFEVarBindings(body iifeBody, bindings returnScope) {
	for _, name := range body.locals {
		hoistIIFEVarBinding(name, bindings)
	}
	for _, name := range body.shadows {
		hoistIIFEVarBinding(name, bindings)
	}
	for _, name := range body.funcLocals {
		hoistIIFEFunctionBinding(name, bindings)
	}
	for _, name := range body.funcShadows {
		hoistIIFEFunctionBinding(name, bindings)
	}
}

func hoistIIFEVarBinding(name string, bindings returnScope) {
	clearReturnScopeBinding(bindings, name)
	delete(bindings.funcSeq, name)
	delete(bindings.objectSeq, name)
	bindings.nullish[name] = returnUndefinedKind
}

func hoistIIFEFunctionBinding(name string, bindings returnScope) {
	assignFunctionIdentity(bindings, name, allocateFunctionIdentity(bindings))
}

func clearIIFELocals(locals []string, bindings returnScope) {
	for _, name := range locals {
		clearReturnScopeBinding(bindings, name)
		delete(bindings.funcSeq, name)
		delete(bindings.objectSeq, name)
	}
}

func restoreIIFEShadows(shadows []string, source returnScope, target returnScope) {
	for _, name := range shadows {
		restoreIIFEShadow(name, source, target)
	}
}

func restoreIIFEShadow(name string, source returnScope, target returnScope) {
	clearReturnScopeBinding(target, name)
	delete(target.funcSeq, name)
	delete(target.objectSeq, name)
	if value, ok := source.ints[name]; ok {
		target.ints[name] = value
	}
	if value, ok := source.bigints[name]; ok {
		target.bigints[name] = value
	}
	if value, ok := source.bools[name]; ok {
		target.bools[name] = value
	}
	if value, ok := source.strings[name]; ok {
		target.strings[name] = value
	}
	if value, ok := source.nullish[name]; ok {
		target.nullish[name] = value
	}
	if value, ok := source.funcs[name]; ok {
		target.funcs[name] = value
	}
	if value, ok := source.funcSeq[name]; ok {
		target.funcSeq[name] = value
	}
	if value, ok := source.objects[name]; ok {
		target.objects[name] = value
	}
	if value, ok := source.objectSeq[name]; ok {
		target.objectSeq[name] = value
	}
}

func clearIIFEParameters(params []ast.Parameter, bindings returnScope) {
	for _, param := range params {
		clearReturnScopeBinding(bindings, param.Name)
	}
}

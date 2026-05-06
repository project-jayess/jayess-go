package lowering

import "jayess-go/ast"

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

package lowering

import "jayess-go/ast"

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
		restoreReturnScopeBinding(name, source, target)
	}
}

func restoreIIFEShadow(name string, source returnScope, target returnScope) {
	restoreReturnScopeBinding(name, source, target)
}

func clearIIFEParameters(params []ast.Parameter, bindings returnScope) {
	for _, param := range params {
		clearReturnScopeBinding(bindings, param.Name)
	}
}

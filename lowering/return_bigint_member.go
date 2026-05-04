package lowering

import "jayess-go/ast"

func evaluateBigIntMemberExpression(expression *ast.MemberExpression, bindings returnScope) (string, bool) {
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayBigIntKind {
		return "", false
	}
	return value.bigIntValue, true
}

func evaluateBigIntIndexExpression(expression *ast.IndexExpression, bindings returnScope) (string, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayBigIntKind {
		return value.bigIntValue, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayBigIntKind {
		return "", false
	}
	return value.bigIntValue, true
}

package lowering

import "jayess-go/ast"

func evaluateIntUpdateExpression(expression *ast.UpdateExpression, scope returnScope) (int, bool) {
	identifier, ok := expression.Target.(*ast.Identifier)
	if !ok || identifier.Name == "" {
		return 0, false
	}
	current, ok := scope.ints[identifier.Name]
	if !ok {
		return 0, false
	}
	next, ok := updatedIntValue(expression.Operator, current)
	if !ok {
		return 0, false
	}
	scope.ints[identifier.Name] = next
	if expression.Prefix {
		return next, true
	}
	return current, true
}

func updatedIntValue(operator ast.UpdateOperator, current int) (int, bool) {
	switch operator {
	case ast.UpdateIncrement:
		return current + 1, true
	case ast.UpdateDecrement:
		return current - 1, true
	default:
		return 0, false
	}
}

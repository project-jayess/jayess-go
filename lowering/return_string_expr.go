package lowering

import "jayess-go/ast"

func evaluateStringExpression(expression ast.Expression, bindings returnScope) (string, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.strings[expr.Name]
		return value, ok
	case *ast.StringLiteral:
		return expr.Value, true
	case *ast.TemplateLiteral:
		return evaluateTemplateString(expr, bindings)
	case *ast.TypeofExpression:
		return evaluateTypeofString(expr.Value, bindings)
	case *ast.MemberExpression:
		return evaluateStringMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		if value, ok := evaluateArrayIndexElement(expr, bindings); ok && value.kind == returnArrayStringKind {
			return value.stringValue, true
		}
		if value, ok := evaluateObjectIndexElement(expr, bindings); ok && value.kind == returnArrayStringKind {
			return value.stringValue, true
		}
		return evaluateStringIndexExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateStringIIFEExpression(expr, bindings)
	case *ast.BinaryExpression:
		return evaluateStringBinary(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.LogicalExpression:
		return evaluateStringLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return "", false
		}
		var value string
		if condition {
			value, ok = evaluateStringExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateStringExpression(expr.Alternative, next)
		}
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return "", false
		}
		value, ok := evaluateStringExpression(expr.Right, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateStringExpression(expr.Right, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateStringExpression(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return "", false
	default:
		return "", false
	}
}

func evaluateStringBinary(operator ast.BinaryOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (string, bool) {
	if operator != ast.OperatorAdd {
		return "", false
	}

	next := bindings.clone()
	if left, ok := evaluateStringExpression(leftExpression, next); ok {
		right, ok := evaluateStringCoercion(rightExpression, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return left + right, true
	}

	next = bindings.clone()
	left, ok := evaluateStringCoercion(leftExpression, next)
	if !ok {
		return "", false
	}
	right, ok := evaluateStringExpression(rightExpression, next)
	if !ok {
		return "", false
	}
	replaceReturnScopeBindings(bindings, next)
	return left + right, true
}

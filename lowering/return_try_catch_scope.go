package lowering

import "jayess-go/ast"

func applyCatchBodySideEffects(statement *ast.TryStatement, scope returnScope) nonReturnFlow {
	return withCatchBindingScope(statement, scope, func() nonReturnFlow {
		return applyNonReturnStatements(statement.CatchBody, scope)
	})
}

func tryCatchBodyUntilThrowFlow(statement *ast.TryStatement, scope returnScope) tryThrowFlow {
	return withCatchBindingScope(statement, scope, func() tryThrowFlow {
		return tryBodyUntilThrowFlow(statement.CatchBody, scope)
	})
}

func withCatchBindingScope[T any](statement *ast.TryStatement, scope returnScope, evaluate func() T) T {
	names := tryCatchBindingNames(statement)
	if len(names) == 0 {
		return evaluate()
	}
	beforeCatch := scope.clone()
	for _, name := range names {
		clearReturnScopeBinding(scope, name)
	}
	value := evaluate()
	for _, name := range names {
		restoreReturnScopeBinding(name, beforeCatch, scope)
	}
	return value
}

func tryCatchBindingNames(statement *ast.TryStatement) []string {
	if statement.CatchPattern != nil {
		return bindingNames(statement.CatchPattern)
	}
	if statement.CatchName != "" {
		return []string{statement.CatchName}
	}
	return nil
}

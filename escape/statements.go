package escape

import "jayess-go/ast"

func analyzeStatements(report *Report, scope *scope, statements []ast.Statement) {
	for _, statement := range statements {
		analyzeStatement(report, scope, statement)
	}
}

func analyzeStatement(report *Report, scope *scope, statement ast.Statement) {
	if statement == nil {
		return
	}
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		analyzeExpression(report, scope, stmt.Value)
		recordDeclarationAlias(report, stmt)
		declarePatternInScope(scope, stmt.Pattern)
	case *ast.FunctionDecl:
		scope.declare(stmt.Name)
		functionScope := newScope(scope)
		declareParametersInScope(functionScope, stmt.Params)
		analyzeStatements(report, functionScope, stmt.Body)
	case *ast.BlockStatement:
		analyzeStatements(report, newScope(scope), stmt.Statements)
	case *ast.ExpressionStatement:
		analyzeExpression(report, scope, stmt.Expression)
	case *ast.AssignmentStatement:
		markOuterAssignmentEscaping(report, scope, stmt)
		analyzeExpression(report, scope, stmt.Target)
		analyzeExpression(report, scope, stmt.Value)
	case *ast.ReturnStatement:
		markReturnedExpressionEscaping(report, stmt.Value)
	case *ast.IfStatement:
		analyzeExpression(report, scope, stmt.Condition)
		analyzeStatements(report, newScope(scope), stmt.Consequence)
		analyzeStatements(report, newScope(scope), stmt.Alternative)
	case *ast.WhileStatement:
		analyzeExpression(report, scope, stmt.Condition)
		analyzeStatements(report, newScope(scope), stmt.Body)
	case *ast.DoWhileStatement:
		analyzeStatements(report, newScope(scope), stmt.Body)
		analyzeExpression(report, scope, stmt.Condition)
	case *ast.ForStatement:
		loopScope := newScope(scope)
		analyzeStatement(report, loopScope, stmt.Init)
		analyzeExpression(report, loopScope, stmt.Condition)
		analyzeStatement(report, loopScope, stmt.Update)
		analyzeStatements(report, newScope(loopScope), stmt.Body)
	case *ast.ForOfStatement:
		loopScope := newScope(scope)
		declarePatternInScope(loopScope, stmt.Pattern)
		analyzeExpression(report, loopScope, stmt.Target)
		analyzeExpression(report, loopScope, stmt.Iterable)
		analyzeStatements(report, newScope(loopScope), stmt.Body)
	case *ast.ForInStatement:
		loopScope := newScope(scope)
		declarePatternInScope(loopScope, stmt.Pattern)
		analyzeExpression(report, loopScope, stmt.Target)
		analyzeExpression(report, loopScope, stmt.Object)
		analyzeStatements(report, newScope(loopScope), stmt.Body)
	case *ast.LabeledStatement:
		analyzeStatement(report, scope, stmt.Statement)
	case *ast.SwitchStatement:
		switchScope := newScope(scope)
		analyzeExpression(report, switchScope, stmt.Discriminant)
		for _, switchCase := range stmt.Cases {
			analyzeExpression(report, switchScope, switchCase.Test)
			analyzeStatements(report, newScope(switchScope), switchCase.Consequent)
		}
		analyzeStatements(report, newScope(switchScope), stmt.Default)
	case *ast.ThrowStatement:
		analyzeExpression(report, scope, stmt.Value)
	case *ast.TryStatement:
		analyzeStatements(report, newScope(scope), stmt.TryBody)
		catchScope := newScope(scope)
		declarePatternInScope(catchScope, stmt.CatchPattern)
		analyzeStatements(report, catchScope, stmt.CatchBody)
		analyzeStatements(report, newScope(scope), stmt.FinallyBody)
	}
}

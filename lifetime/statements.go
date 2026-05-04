package lifetime

import (
	"jayess-go/ast"
	"jayess-go/escape"
)

func collectStatementCleanups(report *escape.Report, statements []ast.Statement, plan *Plan, depth int) {
	for _, statement := range statements {
		collectStatementCleanup(report, statement, plan, depth)
	}
}

func collectStatementCleanup(report *escape.Report, statement ast.Statement, plan *Plan, depth int) {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		addDeclarationLifetimeActions(report, stmt, stmt.Pattern, plan, depth)
	case *ast.FunctionDecl:
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.BlockStatement:
		collectStatementCleanups(report, stmt.Statements, plan, depth+1)
	case *ast.IfStatement:
		collectStatementCleanups(report, stmt.Consequence, plan, depth+1)
		collectStatementCleanups(report, stmt.Alternative, plan, depth+1)
	case *ast.WhileStatement:
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.DoWhileStatement:
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.ForStatement:
		collectStatementCleanup(report, stmt.Init, plan, depth+1)
		collectStatementCleanup(report, stmt.Update, plan, depth+1)
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.ForOfStatement:
		addDeclarationLifetimeActions(report, stmt, stmt.Pattern, plan, depth+1)
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.ForInStatement:
		addDeclarationLifetimeActions(report, stmt, stmt.Pattern, plan, depth+1)
		collectStatementCleanups(report, stmt.Body, plan, depth+1)
	case *ast.LabeledStatement:
		collectStatementCleanup(report, stmt.Statement, plan, depth)
	case *ast.SwitchStatement:
		for _, switchCase := range stmt.Cases {
			collectStatementCleanups(report, switchCase.Consequent, plan, depth+1)
		}
		collectStatementCleanups(report, stmt.Default, plan, depth+1)
	case *ast.TryStatement:
		collectStatementCleanups(report, stmt.TryBody, plan, depth+1)
		addDeclarationLifetimeActions(report, stmt, stmt.CatchPattern, plan, depth+1)
		collectStatementCleanups(report, stmt.CatchBody, plan, depth+1)
		collectStatementCleanups(report, stmt.FinallyBody, plan, depth+1)
	}
}

func addDeclarationLifetimeActions(report *escape.Report, node ast.Node, pattern ast.BindingPattern, plan *Plan, depth int) {
	pos := ast.PositionOf(node)
	for _, name := range bindingNames(pattern) {
		if depth == 0 {
			plan.ExtendedLifetimes = append(plan.ExtendedLifetimes, ExtendedLifetime{
				Binding:    name,
				Line:       pos.Line,
				Column:     pos.Column,
				ScopeDepth: depth,
			})
			continue
		}
		if report.EligibleForScopeCleanup(name) {
			plan.ScopeExitCleanups = append(plan.ScopeExitCleanups, Cleanup{
				Binding:    name,
				Line:       pos.Line,
				Column:     pos.Column,
				ScopeDepth: depth,
			})
			continue
		}
		if report.MustSurviveScopeExit(name) {
			plan.ExtendedLifetimes = append(plan.ExtendedLifetimes, ExtendedLifetime{
				Binding:    name,
				Line:       pos.Line,
				Column:     pos.Column,
				ScopeDepth: depth,
			})
		}
	}
}

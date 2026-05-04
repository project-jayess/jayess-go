package escape

import (
	"fmt"

	"jayess-go/ast"
)

// Diagnostic describes a lifetime decision for a declared binding.
type Diagnostic struct {
	Line     int
	Column   int
	Binding  string
	Escaping bool
	Message  string
}

// LifetimeDiagnostics reports scope-cleanup decisions for declarations.
func LifetimeDiagnostics(program *ast.Program) []Diagnostic {
	report := Analyze(program)
	if program == nil {
		return nil
	}
	var diagnostics []Diagnostic
	collectStatementDiagnostics(report, program.Statements, &diagnostics)
	return diagnostics
}

func collectStatementDiagnostics(report *Report, statements []ast.Statement, diagnostics *[]Diagnostic) {
	for _, statement := range statements {
		collectStatementDiagnostic(report, statement, diagnostics)
	}
}

func collectStatementDiagnostic(report *Report, statement ast.Statement, diagnostics *[]Diagnostic) {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		addPatternDiagnostics(report, stmt, stmt.Pattern, diagnostics)
	case *ast.FunctionDecl:
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.BlockStatement:
		collectStatementDiagnostics(report, stmt.Statements, diagnostics)
	case *ast.IfStatement:
		collectStatementDiagnostics(report, stmt.Consequence, diagnostics)
		collectStatementDiagnostics(report, stmt.Alternative, diagnostics)
	case *ast.WhileStatement:
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.DoWhileStatement:
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.ForStatement:
		collectStatementDiagnostic(report, stmt.Init, diagnostics)
		collectStatementDiagnostic(report, stmt.Update, diagnostics)
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.ForOfStatement:
		addPatternDiagnostics(report, stmt, stmt.Pattern, diagnostics)
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.ForInStatement:
		addPatternDiagnostics(report, stmt, stmt.Pattern, diagnostics)
		collectStatementDiagnostics(report, stmt.Body, diagnostics)
	case *ast.LabeledStatement:
		collectStatementDiagnostic(report, stmt.Statement, diagnostics)
	case *ast.SwitchStatement:
		for _, switchCase := range stmt.Cases {
			collectStatementDiagnostics(report, switchCase.Consequent, diagnostics)
		}
		collectStatementDiagnostics(report, stmt.Default, diagnostics)
	case *ast.TryStatement:
		collectStatementDiagnostics(report, stmt.TryBody, diagnostics)
		addPatternDiagnostics(report, stmt, stmt.CatchPattern, diagnostics)
		collectStatementDiagnostics(report, stmt.CatchBody, diagnostics)
		collectStatementDiagnostics(report, stmt.FinallyBody, diagnostics)
	}
}

func addPatternDiagnostics(report *Report, node ast.Node, pattern ast.BindingPattern, diagnostics *[]Diagnostic) {
	names := map[string]bool{}
	declarePattern(names, pattern)
	pos := ast.PositionOf(node)
	for name := range names {
		escaping := report.MustSurviveScopeExit(name)
		*diagnostics = append(*diagnostics, Diagnostic{
			Line:     pos.Line,
			Column:   pos.Column,
			Binding:  name,
			Escaping: escaping,
			Message:  lifetimeDiagnosticMessage(name, escaping),
		})
	}
}

func lifetimeDiagnosticMessage(name string, escaping bool) string {
	if escaping {
		return fmt.Sprintf("%s escapes lexical scope; skip scope-exit cleanup", name)
	}
	return fmt.Sprintf("%s does not escape lexical scope; eligible for scope-exit cleanup", name)
}

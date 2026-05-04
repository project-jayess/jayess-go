package escape

import "jayess-go/ast"

// Analyze conservatively classifies values that escape their lexical scope.
func Analyze(program *ast.Program) *Report {
	report := newReport()
	if program == nil {
		return report
	}
	analyzeStatements(report, newScope(nil), program.Statements)
	return report
}

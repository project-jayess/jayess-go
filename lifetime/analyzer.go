package lifetime

import "jayess-go/ast"

type Report struct {
	EscapesDetected bool
}

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(program *ast.Program) Report {
	// The MVP language subset cannot express escaping values yet.
	return Report{EscapesDetected: false}
}

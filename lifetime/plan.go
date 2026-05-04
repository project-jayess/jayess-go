package lifetime

import (
	"jayess-go/ast"
	"jayess-go/escape"
)

// Cleanup describes a binding that should be destroyed at lexical scope exit.
type Cleanup struct {
	Binding    string
	Line       int
	Column     int
	ScopeDepth int
}

// ExtendedLifetime describes a binding whose lifetime must extend past scope exit.
type ExtendedLifetime struct {
	Binding    string
	Line       int
	Column     int
	ScopeDepth int
}

// ClosureCapture describes a binding captured by a closure environment.
type ClosureCapture struct {
	Binding          string
	ByReference      bool
	LifetimeExtended bool
	SharedSlot       int
	Mutated          bool
	NonDangling      bool
}

// ClosureEnvironment describes a closure's captured bindings.
type ClosureEnvironment struct {
	Line       int
	Column     int
	Allocation string
	Captures   []ClosureCapture
}

// Plan contains lifetime actions derived from escape analysis.
type Plan struct {
	ScopeExitCleanups   []Cleanup
	ExtendedLifetimes   []ExtendedLifetime
	ClosureEnvironments []ClosureEnvironment
}

// BuildScopeExitPlan schedules cleanup for non-escaping local declarations.
func BuildScopeExitPlan(program *ast.Program) Plan {
	report := escape.Analyze(program)
	var plan Plan
	if program == nil {
		return plan
	}
	collectStatementCleanups(report, program.Statements, &plan, 0)
	collectClosureEnvironments(program.Statements, &plan)
	return plan
}

package lowering

import "jayess-go/lifetime"

// CleanupOp is a lowered destructor/cleanup action for a binding.
type CleanupOp struct {
	Binding    string
	Line       int
	Column     int
	ScopeDepth int
}

// PreserveOp records an escaping binding that must not be cleaned up at scope exit.
type PreserveOp struct {
	Binding    string
	Line       int
	Column     int
	ScopeDepth int
}

// LowerCleanupOps converts a lifetime plan into cleanup operations.
func LowerCleanupOps(plan lifetime.Plan) []CleanupOp {
	ops := make([]CleanupOp, 0, len(plan.ScopeExitCleanups))
	for _, cleanup := range plan.ScopeExitCleanups {
		ops = append(ops, CleanupOp{
			Binding:    cleanup.Binding,
			Line:       cleanup.Line,
			Column:     cleanup.Column,
			ScopeDepth: cleanup.ScopeDepth,
		})
	}
	return ops
}

// LowerPreserveOps converts extended lifetimes into cleanup suppression operations.
func LowerPreserveOps(plan lifetime.Plan) []PreserveOp {
	ops := make([]PreserveOp, 0, len(plan.ExtendedLifetimes))
	for _, extended := range plan.ExtendedLifetimes {
		ops = append(ops, PreserveOp{
			Binding:    extended.Binding,
			Line:       extended.Line,
			Column:     extended.Column,
			ScopeDepth: extended.ScopeDepth,
		})
	}
	return ops
}

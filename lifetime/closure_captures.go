package lifetime

import "jayess-go/ast"

func closureCaptures(names []string, mutated map[string]bool, plan *Plan, slots map[string]int) []ClosureCapture {
	captures := make([]ClosureCapture, 0, len(names))
	for _, name := range names {
		extended := hasExtendedLifetime(plan, name)
		captures = append(captures, ClosureCapture{
			Binding:          name,
			ByReference:      true,
			LifetimeExtended: extended,
			SharedSlot:       sharedCaptureSlot(slots, name),
			Mutated:          mutated[name],
			NonDangling:      extended,
		})
	}
	return captures
}

func mutatedCapturedNames(scope map[string]bool, fn *ast.FunctionExpression) map[string]bool {
	locals := map[string]bool{}
	if fn.Name != "" {
		locals[fn.Name] = true
	}
	declareParameters(locals, fn.Params)
	declareStatementNames(locals, fn.Body)
	mutated := map[string]bool{}
	collectMutatedCapturedStatementNames(scope, locals, fn.Body, mutated)
	return mutated
}

func sharedCaptureSlot(slots map[string]int, binding string) int {
	if slot, ok := slots[binding]; ok {
		return slot
	}
	slot := len(slots)
	slots[binding] = slot
	return slot
}

func hasExtendedLifetime(plan *Plan, binding string) bool {
	if plan == nil {
		return false
	}
	for _, extended := range plan.ExtendedLifetimes {
		if extended.Binding == binding {
			return true
		}
	}
	return false
}

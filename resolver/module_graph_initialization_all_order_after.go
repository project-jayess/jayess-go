package resolver

// InitializesAfterAll reports whether one module initializes after another in the full graph.
func (g *ModuleGraph) InitializesAfterAll(after string, before string) (bool, error) {
	return g.InitializesBeforeAll(before, after)
}

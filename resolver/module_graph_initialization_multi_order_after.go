package resolver

// InitializesAfterFor reports whether one module initializes after another for entries.
func (g *ModuleGraph) InitializesAfterFor(entries []string, after string, before string) (bool, error) {
	return g.InitializesBeforeFor(entries, before, after)
}

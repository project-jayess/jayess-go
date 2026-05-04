package resolver

// InitializesAfter reports whether one module initializes after another for an entry.
func (g *ModuleGraph) InitializesAfter(entry string, after string, before string) (bool, error) {
	return g.InitializesBefore(entry, before, after)
}

package resolver

// InitializesBeforeAll reports whether one module initializes before another in the full graph.
func (g *ModuleGraph) InitializesBeforeAll(before string, after string) (bool, error) {
	positions, err := g.InitializationOrderPositionsAll()
	if err != nil {
		return false, err
	}
	return initializationOrderPositionsBefore(positions, before, after), nil
}

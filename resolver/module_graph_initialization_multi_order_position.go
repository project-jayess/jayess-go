package resolver

// InitializationOrderPositionFor returns one module position within a multi-entry initialization order.
func (g *ModuleGraph) InitializationOrderPositionFor(entries []string, module string) (int, bool, error) {
	positions, err := g.InitializationOrderPositionsFor(entries)
	if err != nil {
		return 0, false, err
	}
	if positions == nil {
		return 0, false, nil
	}

	position, ok := positions[module]
	return position, ok, nil
}

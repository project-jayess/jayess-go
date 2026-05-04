package resolver

// InitializationOrderPositionAll returns one module position within the full-graph initialization order.
func (g *ModuleGraph) InitializationOrderPositionAll(module string) (int, bool, error) {
	positions, err := g.InitializationOrderPositionsAll()
	if err != nil {
		return 0, false, err
	}
	if positions == nil {
		return 0, false, nil
	}

	position, ok := positions[module]
	return position, ok, nil
}

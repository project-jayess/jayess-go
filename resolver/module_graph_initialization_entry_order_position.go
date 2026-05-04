package resolver

// InitializationOrderPosition returns one module position within an entry initialization order.
func (g *ModuleGraph) InitializationOrderPosition(entry string, module string) (int, bool, error) {
	positions, err := g.InitializationOrderPositions(entry)
	if err != nil {
		return 0, false, err
	}
	if positions == nil {
		return 0, false, nil
	}

	position, ok := positions[module]
	return position, ok, nil
}

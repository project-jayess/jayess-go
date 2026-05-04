package resolver

// InitializationOrderPositionsFor returns module positions within a multi-entry initialization order.
func (g *ModuleGraph) InitializationOrderPositionsFor(entries []string) (map[string]int, error) {
	order, err := g.InitializationOrderFor(entries)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}

	positions := make(map[string]int, len(order))
	for index, module := range order {
		positions[module] = index
	}
	return positions, nil
}

package resolver

// InitializationOrderPositions returns module positions within an entry initialization order.
func (g *ModuleGraph) InitializationOrderPositions(entry string) (map[string]int, error) {
	order, err := g.InitializationOrder(entry)
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

package resolver

// ReachableModuleOrderMap returns reachable initialization orders by entry module.
func (g *ModuleGraph) ReachableModuleOrderMap() (map[string][]string, error) {
	if g.imports == nil {
		return nil, nil
	}

	orders := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		order, err := g.ReachableModuleOrder(module)
		if err != nil {
			return nil, err
		}
		orders[module] = order
	}
	return orders, nil
}

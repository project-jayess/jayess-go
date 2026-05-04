package resolver

// ReachableModuleCountMap returns reachable module counts by entry module.
func (g *ModuleGraph) ReachableModuleCountMap() (map[string]int, error) {
	if g.imports == nil {
		return nil, nil
	}

	counts := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		count, err := g.ReachableModuleCount(module)
		if err != nil {
			return nil, err
		}
		counts[module] = count
	}
	return counts, nil
}

package resolver

// ReachableModuleSetMap returns reachable module sets by entry module.
func (g *ModuleGraph) ReachableModuleSetMap() (map[string]map[string]bool, error) {
	if g.imports == nil {
		return nil, nil
	}

	sets := make(map[string]map[string]bool, len(g.imports))
	for _, module := range g.Modules() {
		set, err := g.ReachableModuleSet(module)
		if err != nil {
			return nil, err
		}
		sets[module] = set
	}
	return sets, nil
}

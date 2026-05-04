package resolver

func (g *ModuleGraph) DependencyCountMap() map[string]int {
	if g.imports == nil {
		return nil
	}
	counts := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		counts[module] = g.DependencyCount(module)
	}
	return counts
}

func (g *ModuleGraph) DependentCountMap() map[string]int {
	if g.imports == nil {
		return nil
	}
	counts := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		counts[module] = g.DependentCount(module)
	}
	return counts
}

package resolver

// TransitiveDependentCountMap returns transitive dependent counts by module.
func (g *ModuleGraph) TransitiveDependentCountMap() map[string]int {
	if g.imports == nil {
		return nil
	}

	counts := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		counts[module] = g.TransitiveDependentCount(module)
	}
	return counts
}

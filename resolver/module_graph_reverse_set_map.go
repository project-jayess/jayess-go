package resolver

// TransitiveDependentSetMap returns transitive dependent sets by module.
func (g *ModuleGraph) TransitiveDependentSetMap() map[string]map[string]bool {
	if g.imports == nil {
		return nil
	}

	sets := make(map[string]map[string]bool, len(g.imports))
	for _, module := range g.Modules() {
		sets[module] = g.TransitiveDependentSet(module)
	}
	return sets
}

package resolver

// TransitiveDependencyCountMap returns transitive dependency counts by module.
func (g *ModuleGraph) TransitiveDependencyCountMap() (map[string]int, error) {
	if g.imports == nil {
		return nil, nil
	}

	counts := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		count, err := g.TransitiveDependencyCount(module)
		if err != nil {
			return nil, err
		}
		counts[module] = count
	}
	return counts, nil
}

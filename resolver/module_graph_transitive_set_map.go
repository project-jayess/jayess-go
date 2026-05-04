package resolver

// TransitiveDependencySetMap returns transitive dependency sets by module.
func (g *ModuleGraph) TransitiveDependencySetMap() (map[string]map[string]bool, error) {
	if g.imports == nil {
		return nil, nil
	}

	sets := make(map[string]map[string]bool, len(g.imports))
	for _, module := range g.Modules() {
		set, err := g.TransitiveDependencySet(module)
		if err != nil {
			return nil, err
		}
		sets[module] = set
	}
	return sets, nil
}

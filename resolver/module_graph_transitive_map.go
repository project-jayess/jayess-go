package resolver

func (g *ModuleGraph) TransitiveDependencyMap() (map[string][]string, error) {
	if g.imports == nil {
		return nil, nil
	}
	dependencies := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		moduleDependencies, err := g.TransitiveDependencies(module)
		if err != nil {
			return nil, err
		}
		dependencies[module] = moduleDependencies
	}
	return dependencies, nil
}

func (g *ModuleGraph) TransitiveDependentMap() map[string][]string {
	if g.imports == nil {
		return nil
	}
	dependents := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		dependents[module] = g.TransitiveDependents(module)
	}
	return dependents
}

package resolver

func (g *ModuleGraph) TransitiveDependencyCount(module string) (int, error) {
	dependencies, err := g.TransitiveDependencies(module)
	if err != nil {
		return 0, err
	}
	return len(dependencies), nil
}

func (g *ModuleGraph) TransitiveDependentCount(module string) int {
	return len(g.TransitiveDependents(module))
}

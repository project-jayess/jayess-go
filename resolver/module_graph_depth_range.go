package resolver

func (g *ModuleGraph) ModulesWithinDependencyDepth(maxDepth int) ([]string, error) {
	if maxDepth < 0 {
		return nil, nil
	}
	depths, err := g.DependencyDepthMap()
	if err != nil {
		return nil, err
	}
	var modules []string
	for _, module := range g.Modules() {
		if depths[module] <= maxDepth {
			modules = append(modules, module)
		}
	}
	return modules, nil
}

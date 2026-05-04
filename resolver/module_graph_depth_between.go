package resolver

func (g *ModuleGraph) ModulesBetweenDependencyDepth(minDepth int, maxDepth int) ([]string, error) {
	if maxDepth < minDepth {
		return nil, nil
	}
	depths, err := g.DependencyDepthMap()
	if err != nil {
		return nil, err
	}
	var modules []string
	for _, module := range g.Modules() {
		depth := depths[module]
		if depth >= minDepth && depth <= maxDepth {
			modules = append(modules, module)
		}
	}
	return modules, nil
}

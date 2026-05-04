package resolver

func (g *ModuleGraph) ModulesAtDependentDepth(targetDepth int) ([]string, error) {
	if targetDepth < 0 {
		return nil, nil
	}
	depths, err := g.DependentDepthMap()
	if err != nil {
		return nil, err
	}
	var modules []string
	for _, module := range g.Modules() {
		if depths[module] == targetDepth {
			modules = append(modules, module)
		}
	}
	return modules, nil
}

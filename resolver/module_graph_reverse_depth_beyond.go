package resolver

func (g *ModuleGraph) ModulesBeyondDependentDepth(minDepth int) ([]string, error) {
	depths, err := g.DependentDepthMap()
	if err != nil {
		return nil, err
	}
	var modules []string
	for _, module := range g.Modules() {
		if depths[module] > minDepth {
			modules = append(modules, module)
		}
	}
	return modules, nil
}

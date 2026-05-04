package resolver

func (g *ModuleGraph) DeepestDependentModules() ([]string, int, error) {
	depths, err := g.DependentDepthMap()
	if err != nil {
		return nil, 0, err
	}
	if len(depths) == 0 {
		return nil, 0, nil
	}
	maxDepth := 0
	var modules []string
	for _, module := range g.Modules() {
		depth := depths[module]
		if depth > maxDepth {
			maxDepth = depth
			modules = []string{module}
			continue
		}
		if depth == maxDepth {
			modules = append(modules, module)
		}
	}
	return modules, maxDepth, nil
}

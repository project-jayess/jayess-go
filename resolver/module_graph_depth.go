package resolver

func (g *ModuleGraph) DependencyDepth(module string) (int, error) {
	if g.imports == nil || !g.HasModule(module) {
		return 0, nil
	}
	memo := map[string]int{}
	active := map[string]int{}
	var stack []string
	return g.dependencyDepth(module, memo, active, &stack)
}

func (g *ModuleGraph) DependencyDepthMap() (map[string]int, error) {
	if g.imports == nil {
		return nil, nil
	}
	depths := make(map[string]int, len(g.imports))
	for _, module := range g.Modules() {
		depth, err := g.DependencyDepth(module)
		if err != nil {
			return nil, err
		}
		depths[module] = depth
	}
	return depths, nil
}

func (g *ModuleGraph) dependencyDepth(module string, memo map[string]int, active map[string]int, stack *[]string) (int, error) {
	if depth, ok := memo[module]; ok {
		return depth, nil
	}
	if index, ok := active[module]; ok {
		cycle := append([]string(nil), (*stack)[index:]...)
		cycle = append(cycle, module)
		return 0, &ImportCycleError{Cycle: cycle}
	}
	active[module] = len(*stack)
	*stack = append(*stack, module)
	depth := 0
	for _, imported := range g.imports[module] {
		importDepth, err := g.dependencyDepth(imported, memo, active, stack)
		if err != nil {
			return 0, err
		}
		if importDepth+1 > depth {
			depth = importDepth + 1
		}
	}
	*stack = (*stack)[:len(*stack)-1]
	delete(active, module)
	memo[module] = depth
	return depth, nil
}

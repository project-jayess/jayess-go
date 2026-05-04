package resolver

func (g *ModuleGraph) LongestDependentPath(module string) ([]string, error) {
	if g.imports == nil || !g.HasModule(module) {
		return nil, nil
	}
	memo := map[string][]string{}
	active := map[string]int{}
	var stack []string
	return g.longestDependentPath(module, memo, active, &stack)
}

func (g *ModuleGraph) longestDependentPath(module string, memo map[string][]string, active map[string]int, stack *[]string) ([]string, error) {
	if path, ok := memo[module]; ok {
		return append([]string(nil), path...), nil
	}
	if index, ok := active[module]; ok {
		cycle := append([]string(nil), (*stack)[index:]...)
		cycle = append(cycle, module)
		return nil, &ImportCycleError{Cycle: cycle}
	}
	active[module] = len(*stack)
	*stack = append(*stack, module)
	path := []string{module}
	for _, dependent := range g.Dependents(module) {
		dependentPath, err := g.longestDependentPath(dependent, memo, active, stack)
		if err != nil {
			return nil, err
		}
		if len(dependentPath)+1 > len(path) {
			path = append([]string{module}, dependentPath...)
		}
	}
	*stack = (*stack)[:len(*stack)-1]
	delete(active, module)
	memo[module] = append([]string(nil), path...)
	return path, nil
}

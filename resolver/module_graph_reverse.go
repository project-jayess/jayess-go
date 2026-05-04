package resolver

func (g *ModuleGraph) TransitiveDependents(module string) []string {
	if g.imports == nil {
		return nil
	}
	visited := map[string]bool{module: true}
	var dependents []string
	var visit func(string)
	visit = func(current string) {
		for _, dependent := range g.Dependents(current) {
			if visited[dependent] {
				continue
			}
			visited[dependent] = true
			dependents = append(dependents, dependent)
			visit(dependent)
		}
	}
	visit(module)
	return dependents
}

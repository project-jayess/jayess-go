package resolver

func (g *ModuleGraph) DependentMap() map[string][]string {
	if g.imports == nil {
		return nil
	}
	dependents := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		dependents[module] = g.Dependents(module)
	}
	return dependents
}

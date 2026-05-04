package resolver

func (g *ModuleGraph) IsolatedModules() []string {
	if g.imports == nil {
		return nil
	}
	var isolated []string
	for _, module := range g.Modules() {
		if g.IsIsolatedModule(module) {
			isolated = append(isolated, module)
		}
	}
	return isolated
}

package resolver

func (g *ModuleGraph) IsRootModule(module string) bool {
	return g.HasModule(module) && len(g.Dependents(module)) == 0
}

func (g *ModuleGraph) IsLeafModule(module string) bool {
	if g.imports == nil {
		return false
	}
	imports, ok := g.imports[module]
	return ok && len(imports) == 0
}

func (g *ModuleGraph) IsIsolatedModule(module string) bool {
	return g.IsRootModule(module) && g.IsLeafModule(module)
}

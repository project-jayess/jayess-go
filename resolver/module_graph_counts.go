package resolver

func (g *ModuleGraph) ModuleCount() int {
	if g.imports == nil {
		return 0
	}
	return len(g.imports)
}

func (g *ModuleGraph) ImportEdgeCount() int {
	if g.imports == nil {
		return 0
	}
	count := 0
	for _, imports := range g.imports {
		count += len(imports)
	}
	return count
}

func (g *ModuleGraph) DependencyCount(module string) int {
	if g.imports == nil {
		return 0
	}
	return len(g.imports[module])
}

func (g *ModuleGraph) DependentCount(module string) int {
	return len(g.Dependents(module))
}

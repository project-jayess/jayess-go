package resolver

func (g *ModuleGraph) RemoveModule(module string) bool {
	if g.imports == nil {
		return false
	}
	if _, ok := g.imports[module]; !ok {
		return false
	}
	delete(g.imports, module)
	for importer := range g.imports {
		g.RemoveAllImports(importer, module)
	}
	return true
}

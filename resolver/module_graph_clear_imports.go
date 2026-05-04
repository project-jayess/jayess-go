package resolver

func (g *ModuleGraph) ClearImports(module string) bool {
	if g.imports == nil {
		return false
	}
	if _, ok := g.imports[module]; !ok {
		return false
	}
	g.imports[module] = nil
	return true
}

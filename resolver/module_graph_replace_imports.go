package resolver

func (g *ModuleGraph) ReplaceImports(module string, imports []string) {
	g.AddModule(module, imports)
}

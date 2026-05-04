package resolver

func (g *ModuleGraph) AddImport(module string, imported string) {
	if g.imports == nil {
		g.imports = map[string][]string{}
	}
	g.imports[module] = append(g.imports[module], imported)
	if _, ok := g.imports[imported]; !ok {
		g.imports[imported] = nil
	}
}

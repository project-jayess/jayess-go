package resolver

func (g *ModuleGraph) reachableSubgraph(order []string) *ModuleGraph {
	included := map[string]bool{}
	for _, module := range order {
		included[module] = true
	}

	graph := NewModuleGraph()
	for _, module := range order {
		var imports []string
		for _, imported := range g.imports[module] {
			if included[imported] {
				imports = append(imports, imported)
			}
		}
		graph.AddModule(module, imports)
	}
	return graph
}

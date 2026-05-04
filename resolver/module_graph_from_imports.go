package resolver

func NewModuleGraphFromImports(imports map[string][]string) *ModuleGraph {
	graph := NewModuleGraph()
	for module, moduleImports := range imports {
		graph.AddModule(module, moduleImports)
	}
	return graph
}

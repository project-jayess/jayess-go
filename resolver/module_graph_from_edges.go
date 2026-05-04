package resolver

func NewModuleGraphFromEdges(edges []ModuleImportEdge) *ModuleGraph {
	graph := NewModuleGraph()
	for _, edge := range edges {
		graph.AddImport(edge.From, edge.To)
	}
	return graph
}

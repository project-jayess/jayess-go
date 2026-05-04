package resolver

// ReachableSubgraph returns a module graph containing only an entry module and its imports.
func (g *ModuleGraph) ReachableSubgraph(entry string) (*ModuleGraph, error) {
	order, err := g.InitializationOrder(entry)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return NewModuleGraph(), nil
	}
	return g.reachableSubgraph(order), nil
}

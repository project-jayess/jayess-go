package resolver

// ReachableSubgraphFor returns a module graph containing only modules reachable from entries.
func (g *ModuleGraph) ReachableSubgraphFor(entries []string) (*ModuleGraph, error) {
	order, err := g.InitializationOrderFor(entries)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return NewModuleGraph(), nil
	}
	return g.reachableSubgraph(order), nil
}

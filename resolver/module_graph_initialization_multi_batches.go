package resolver

// InitializationBatchesFor returns dependency-ready module batches for multiple entries.
func (g *ModuleGraph) InitializationBatchesFor(entries []string) ([][]string, error) {
	order, err := g.InitializationOrderFor(entries)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}

	return g.reachableSubgraph(order).DependencyDepthLayers()
}

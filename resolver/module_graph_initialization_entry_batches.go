package resolver

// InitializationBatches returns dependency-ready module batches for an entry module.
func (g *ModuleGraph) InitializationBatches(entry string) ([][]string, error) {
	order, err := g.InitializationOrder(entry)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}

	return g.reachableSubgraph(order).DependencyDepthLayers()
}

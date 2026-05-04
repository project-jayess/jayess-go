package resolver

// InitializationBatchesAll returns dependency-ready module batches for the whole graph.
func (g *ModuleGraph) InitializationBatchesAll() ([][]string, error) {
	return g.DependencyDepthLayers()
}

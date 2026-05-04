package resolver

// InitializationBatchWidthsAll returns dependency-ready batch sizes for the whole graph.
func (g *ModuleGraph) InitializationBatchWidthsAll() ([]int, error) {
	return g.DependencyDepthLayerWidths()
}

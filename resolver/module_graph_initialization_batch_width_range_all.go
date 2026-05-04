package resolver

// InitializationBatchWidthRangeAll returns the smallest and largest dependency-ready batch widths for the whole graph.
func (g *ModuleGraph) InitializationBatchWidthRangeAll() (InitializationBatchWidthRange, error) {
	widths, err := g.InitializationBatchWidthsAll()
	if err != nil {
		return InitializationBatchWidthRange{}, err
	}
	return initializationBatchWidthRange(widths), nil
}

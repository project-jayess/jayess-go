package resolver

// InitializationBatchWidthsFor returns dependency-ready batch sizes for multiple entries.
func (g *ModuleGraph) InitializationBatchWidthsFor(entries []string) ([]int, error) {
	batches, err := g.InitializationBatchesFor(entries)
	if err != nil {
		return nil, err
	}
	return initializationBatchWidths(batches), nil
}

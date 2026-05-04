package resolver

// InitializationBatchWidths returns dependency-ready batch sizes for an entry module.
func (g *ModuleGraph) InitializationBatchWidths(entry string) ([]int, error) {
	batches, err := g.InitializationBatches(entry)
	if err != nil {
		return nil, err
	}
	return initializationBatchWidths(batches), nil
}

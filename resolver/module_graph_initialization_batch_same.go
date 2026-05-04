package resolver

// InitializesInSameBatchAll reports whether two modules share a full-graph initialization batch.
func (g *ModuleGraph) InitializesInSameBatchAll(first string, second string) (bool, error) {
	batches, err := g.InitializationBatchesAll()
	if err != nil {
		return false, err
	}
	return initializationBatchesContainSameBatch(batches, first, second), nil
}

package resolver

// InitializesInSameBatchFor reports whether two modules share a multi-entry initialization batch.
func (g *ModuleGraph) InitializesInSameBatchFor(entries []string, first string, second string) (bool, error) {
	batches, err := g.InitializationBatchesFor(entries)
	if err != nil {
		return false, err
	}
	return initializationBatchesContainSameBatch(batches, first, second), nil
}

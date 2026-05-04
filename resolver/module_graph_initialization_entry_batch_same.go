package resolver

// InitializesInSameBatch reports whether two modules share an entry initialization batch.
func (g *ModuleGraph) InitializesInSameBatch(entry string, first string, second string) (bool, error) {
	batches, err := g.InitializationBatches(entry)
	if err != nil {
		return false, err
	}
	return initializationBatchesContainSameBatch(batches, first, second), nil
}

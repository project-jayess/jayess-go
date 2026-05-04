package resolver

// InitializationBatchIndexes returns dependency-ready batch indexes by module for an entry.
func (g *ModuleGraph) InitializationBatchIndexes(entry string) (map[string]int, error) {
	batches, err := g.InitializationBatches(entry)
	if err != nil {
		return nil, err
	}
	return initializationBatchIndexes(batches), nil
}

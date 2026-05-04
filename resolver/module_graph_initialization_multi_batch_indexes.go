package resolver

// InitializationBatchIndexesFor returns dependency-ready batch indexes by module for entries.
func (g *ModuleGraph) InitializationBatchIndexesFor(entries []string) (map[string]int, error) {
	batches, err := g.InitializationBatchesFor(entries)
	if err != nil {
		return nil, err
	}
	return initializationBatchIndexes(batches), nil
}

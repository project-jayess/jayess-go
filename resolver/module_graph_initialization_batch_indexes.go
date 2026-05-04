package resolver

// InitializationBatchIndexesAll returns dependency-ready batch indexes by module for the whole graph.
func (g *ModuleGraph) InitializationBatchIndexesAll() (map[string]int, error) {
	batches, err := g.InitializationBatchesAll()
	if err != nil {
		return nil, err
	}
	return initializationBatchIndexes(batches), nil
}

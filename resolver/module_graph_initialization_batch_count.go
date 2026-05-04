package resolver

// InitializationBatchCountAll returns the number of dependency-ready batches for the whole graph.
func (g *ModuleGraph) InitializationBatchCountAll() (int, error) {
	batches, err := g.InitializationBatchesAll()
	if err != nil {
		return 0, err
	}
	return len(batches), nil
}

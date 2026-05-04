package resolver

// InitializationBatchCount returns the number of dependency-ready batches for an entry module.
func (g *ModuleGraph) InitializationBatchCount(entry string) (int, error) {
	batches, err := g.InitializationBatches(entry)
	if err != nil {
		return 0, err
	}
	return len(batches), nil
}

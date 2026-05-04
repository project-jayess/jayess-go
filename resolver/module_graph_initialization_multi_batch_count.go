package resolver

// InitializationBatchCountFor returns the number of dependency-ready batches for multiple entries.
func (g *ModuleGraph) InitializationBatchCountFor(entries []string) (int, error) {
	batches, err := g.InitializationBatchesFor(entries)
	if err != nil {
		return 0, err
	}
	return len(batches), nil
}

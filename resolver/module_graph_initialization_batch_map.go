package resolver

// InitializationBatchMap returns dependency-ready batches by entry module.
func (g *ModuleGraph) InitializationBatchMap() (map[string][][]string, error) {
	return initializationEntryMap(g, g.InitializationBatches)
}

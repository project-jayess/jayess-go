package resolver

// InitializationBatchCountMap returns dependency-ready batch counts by entry module.
func (g *ModuleGraph) InitializationBatchCountMap() (map[string]int, error) {
	return initializationEntryMap(g, g.InitializationBatchCount)
}

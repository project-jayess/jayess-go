package resolver

// InitializationBatchWidthMap returns dependency-ready batch widths by entry module.
func (g *ModuleGraph) InitializationBatchWidthMap() (map[string][]int, error) {
	return initializationEntryMap(g, g.InitializationBatchWidths)
}

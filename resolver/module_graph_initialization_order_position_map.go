package resolver

// InitializationOrderPositionMap returns module positions within each entry initialization order.
func (g *ModuleGraph) InitializationOrderPositionMap() (map[string]map[string]int, error) {
	return initializationEntryMap(g, g.InitializationOrderPositions)
}

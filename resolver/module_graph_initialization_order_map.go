package resolver

// InitializationOrderMap returns dependency-first initialization orders by entry module.
func (g *ModuleGraph) InitializationOrderMap() (map[string][]string, error) {
	return initializationEntryMap(g, g.InitializationOrder)
}

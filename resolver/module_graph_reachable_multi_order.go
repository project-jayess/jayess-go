package resolver

// ReachableModuleOrderFor returns modules reachable from multiple entries in initialization order.
func (g *ModuleGraph) ReachableModuleOrderFor(entries []string) ([]string, error) {
	return g.InitializationOrderFor(entries)
}

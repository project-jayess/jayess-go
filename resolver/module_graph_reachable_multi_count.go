package resolver

// ReachableModuleCountFor returns the number of modules reachable from multiple entries.
func (g *ModuleGraph) ReachableModuleCountFor(entries []string) (int, error) {
	subgraph, err := g.ReachableSubgraphFor(entries)
	if err != nil {
		return 0, err
	}
	return subgraph.ModuleCount(), nil
}

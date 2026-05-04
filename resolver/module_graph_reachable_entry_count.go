package resolver

// ReachableModuleCount returns the number of modules reachable from an entry module.
func (g *ModuleGraph) ReachableModuleCount(entry string) (int, error) {
	subgraph, err := g.ReachableSubgraph(entry)
	if err != nil {
		return 0, err
	}
	return subgraph.ModuleCount(), nil
}

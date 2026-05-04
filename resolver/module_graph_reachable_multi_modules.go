package resolver

// ReachableModulesFor returns modules reachable from multiple entries in deterministic order.
func (g *ModuleGraph) ReachableModulesFor(entries []string) ([]string, error) {
	subgraph, err := g.ReachableSubgraphFor(entries)
	if err != nil {
		return nil, err
	}
	return subgraph.Modules(), nil
}

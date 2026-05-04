package resolver

// ReachableModules returns modules reachable from an entry module in deterministic order.
func (g *ModuleGraph) ReachableModules(entry string) ([]string, error) {
	subgraph, err := g.ReachableSubgraph(entry)
	if err != nil {
		return nil, err
	}
	return subgraph.Modules(), nil
}

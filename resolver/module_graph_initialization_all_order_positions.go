package resolver

// InitializationOrderPositionsAll returns module positions within the full-graph initialization order.
func (g *ModuleGraph) InitializationOrderPositionsAll() (map[string]int, error) {
	if g.imports == nil {
		return nil, nil
	}
	return g.InitializationOrderPositionsFor(g.Modules())
}

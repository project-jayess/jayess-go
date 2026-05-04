package resolver

// DependentDepthLayers returns module groups ordered by ascending dependent depth.
func (g *ModuleGraph) DependentDepthLayers() ([][]string, error) {
	groups, err := g.DependentDepthGroups()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	levels, err := g.DependentDepthLevels()
	if err != nil {
		return nil, err
	}
	layers := make([][]string, 0, len(levels))
	for _, level := range levels {
		layers = append(layers, groups[level])
	}
	return layers, nil
}

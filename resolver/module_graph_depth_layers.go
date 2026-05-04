package resolver

// DependencyDepthLayers returns module groups ordered by ascending dependency depth.
func (g *ModuleGraph) DependencyDepthLayers() ([][]string, error) {
	groups, err := g.DependencyDepthGroups()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	levels, err := g.DependencyDepthLevels()
	if err != nil {
		return nil, err
	}
	layers := make([][]string, 0, len(levels))
	for _, level := range levels {
		layers = append(layers, groups[level])
	}
	return layers, nil
}

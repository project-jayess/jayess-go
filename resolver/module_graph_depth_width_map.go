package resolver

// DependencyDepthWidthMap returns the number of modules at each dependency depth.
func (g *ModuleGraph) DependencyDepthWidthMap() (map[int]int, error) {
	groups, err := g.DependencyDepthGroups()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	widths := make(map[int]int, len(groups))
	for depth, modules := range groups {
		widths[depth] = len(modules)
	}
	return widths, nil
}

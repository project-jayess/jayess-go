package resolver

// DependentDepthWidthMap returns the number of modules at each dependent depth.
func (g *ModuleGraph) DependentDepthWidthMap() (map[int]int, error) {
	groups, err := g.DependentDepthGroups()
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

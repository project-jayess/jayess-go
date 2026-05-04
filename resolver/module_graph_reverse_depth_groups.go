package resolver

func (g *ModuleGraph) DependentDepthGroups() (map[int][]string, error) {
	depths, err := g.DependentDepthMap()
	if err != nil {
		return nil, err
	}
	if len(depths) == 0 {
		return nil, nil
	}
	groups := map[int][]string{}
	for _, module := range g.Modules() {
		depth := depths[module]
		groups[depth] = append(groups[depth], module)
	}
	return groups, nil
}

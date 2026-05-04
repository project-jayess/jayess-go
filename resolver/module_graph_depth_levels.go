package resolver

import "sort"

// DependencyDepthLevels returns dependency depth values in ascending order.
func (g *ModuleGraph) DependencyDepthLevels() ([]int, error) {
	groups, err := g.DependencyDepthGroups()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	levels := make([]int, 0, len(groups))
	for depth := range groups {
		levels = append(levels, depth)
	}
	sort.Ints(levels)
	return levels, nil
}

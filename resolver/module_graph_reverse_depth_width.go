package resolver

func (g *ModuleGraph) WidestDependentDepths() ([]int, int, error) {
	groups, err := g.DependentDepthGroups()
	if err != nil {
		return nil, 0, err
	}
	if len(groups) == 0 {
		return nil, 0, nil
	}
	maxWidth := 0
	var depths []int
	for depth := 0; depth <= g.ModuleCount(); depth++ {
		width := len(groups[depth])
		if width == 0 {
			continue
		}
		if width > maxWidth {
			maxWidth = width
			depths = []int{depth}
			continue
		}
		if width == maxWidth {
			depths = append(depths, depth)
		}
	}
	return depths, maxWidth, nil
}

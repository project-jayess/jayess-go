package resolver

// DependencyDepthLayerWidths returns module counts ordered by ascending dependency depth.
func (g *ModuleGraph) DependencyDepthLayerWidths() ([]int, error) {
	layers, err := g.DependencyDepthLayers()
	if err != nil {
		return nil, err
	}
	if len(layers) == 0 {
		return nil, nil
	}

	widths := make([]int, 0, len(layers))
	for _, layer := range layers {
		widths = append(widths, len(layer))
	}
	return widths, nil
}

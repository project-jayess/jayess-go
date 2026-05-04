package resolver

// DependentDepthLayerWidths returns module counts ordered by ascending dependent depth.
func (g *ModuleGraph) DependentDepthLayerWidths() ([]int, error) {
	layers, err := g.DependentDepthLayers()
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

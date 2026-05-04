package resolver

// WidestInitializationBatchesAll returns widest dependency-ready batch indexes and width.
func (g *ModuleGraph) WidestInitializationBatchesAll() ([]int, int, error) {
	widths, err := g.InitializationBatchWidthsAll()
	if err != nil {
		return nil, 0, err
	}
	indexes, maxWidth := widestInitializationBatchIndexes(widths)
	return indexes, maxWidth, nil
}

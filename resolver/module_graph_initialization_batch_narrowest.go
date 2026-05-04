package resolver

// NarrowestInitializationBatchesAll returns narrowest dependency-ready batch indexes and width.
func (g *ModuleGraph) NarrowestInitializationBatchesAll() ([]int, int, error) {
	widths, err := g.InitializationBatchWidthsAll()
	if err != nil {
		return nil, 0, err
	}
	indexes, minWidth := narrowestInitializationBatchIndexes(widths)
	return indexes, minWidth, nil
}

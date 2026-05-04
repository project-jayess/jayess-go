package resolver

// NarrowestInitializationBatchesFor returns narrowest dependency-ready batch indexes and width for entries.
func (g *ModuleGraph) NarrowestInitializationBatchesFor(entries []string) ([]int, int, error) {
	widths, err := g.InitializationBatchWidthsFor(entries)
	if err != nil {
		return nil, 0, err
	}
	indexes, minWidth := narrowestInitializationBatchIndexes(widths)
	return indexes, minWidth, nil
}

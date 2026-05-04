package resolver

// NarrowestInitializationBatches returns narrowest dependency-ready batch indexes and width for an entry.
func (g *ModuleGraph) NarrowestInitializationBatches(entry string) ([]int, int, error) {
	widths, err := g.InitializationBatchWidths(entry)
	if err != nil {
		return nil, 0, err
	}
	indexes, minWidth := narrowestInitializationBatchIndexes(widths)
	return indexes, minWidth, nil
}

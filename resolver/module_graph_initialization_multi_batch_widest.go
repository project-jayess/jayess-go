package resolver

// WidestInitializationBatchesFor returns widest dependency-ready batch indexes and width for entries.
func (g *ModuleGraph) WidestInitializationBatchesFor(entries []string) ([]int, int, error) {
	widths, err := g.InitializationBatchWidthsFor(entries)
	if err != nil {
		return nil, 0, err
	}
	indexes, maxWidth := widestInitializationBatchIndexes(widths)
	return indexes, maxWidth, nil
}

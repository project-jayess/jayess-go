package resolver

// WidestInitializationBatches returns widest dependency-ready batch indexes and width for an entry.
func (g *ModuleGraph) WidestInitializationBatches(entry string) ([]int, int, error) {
	widths, err := g.InitializationBatchWidths(entry)
	if err != nil {
		return nil, 0, err
	}
	indexes, maxWidth := widestInitializationBatchIndexes(widths)
	return indexes, maxWidth, nil
}

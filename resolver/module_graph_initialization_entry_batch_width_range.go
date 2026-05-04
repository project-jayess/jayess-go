package resolver

// InitializationBatchWidthRange returns the smallest and largest dependency-ready batch widths for an entry.
func (g *ModuleGraph) InitializationBatchWidthRange(entry string) (InitializationBatchWidthRange, error) {
	widths, err := g.InitializationBatchWidths(entry)
	if err != nil {
		return InitializationBatchWidthRange{}, err
	}
	return initializationBatchWidthRange(widths), nil
}

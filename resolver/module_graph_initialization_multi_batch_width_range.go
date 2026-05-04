package resolver

// InitializationBatchWidthRangeFor returns the smallest and largest dependency-ready batch widths for entries.
func (g *ModuleGraph) InitializationBatchWidthRangeFor(entries []string) (InitializationBatchWidthRange, error) {
	widths, err := g.InitializationBatchWidthsFor(entries)
	if err != nil {
		return InitializationBatchWidthRange{}, err
	}
	return initializationBatchWidthRange(widths), nil
}

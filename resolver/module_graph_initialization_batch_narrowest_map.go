package resolver

// NarrowestInitializationBatchSummary describes the narrowest dependency-ready batch.
type NarrowestInitializationBatchSummary struct {
	Indexes []int
	Width   int
}

// NarrowestInitializationBatchMap returns narrowest dependency-ready batch summaries by entry module.
func (g *ModuleGraph) NarrowestInitializationBatchMap() (map[string]NarrowestInitializationBatchSummary, error) {
	return initializationBatchSummaryMap(
		g,
		g.NarrowestInitializationBatches,
		func(indexes []int, width int) NarrowestInitializationBatchSummary {
			return NarrowestInitializationBatchSummary{
				Indexes: indexes,
				Width:   width,
			}
		},
	)
}

package resolver

// WidestInitializationBatchSummary describes the widest dependency-ready batch.
type WidestInitializationBatchSummary struct {
	Indexes []int
	Width   int
}

// WidestInitializationBatchMap returns widest dependency-ready batch summaries by entry module.
func (g *ModuleGraph) WidestInitializationBatchMap() (map[string]WidestInitializationBatchSummary, error) {
	return initializationBatchSummaryMap(
		g,
		g.WidestInitializationBatches,
		func(indexes []int, width int) WidestInitializationBatchSummary {
			return WidestInitializationBatchSummary{
				Indexes: indexes,
				Width:   width,
			}
		},
	)
}

package resolver

func initializationBatchSummaryMap[T any](
	g *ModuleGraph,
	resolve func(string) ([]int, int, error),
	summarize func([]int, int) T,
) (map[string]T, error) {
	if g.imports == nil {
		return nil, nil
	}

	summaries := make(map[string]T, len(g.imports))
	for _, module := range g.Modules() {
		indexes, width, err := resolve(module)
		if err != nil {
			return nil, err
		}
		summaries[module] = summarize(indexes, width)
	}
	return summaries, nil
}

package resolver

func initializationBatchIndexes(batches [][]string) map[string]int {
	if len(batches) == 0 {
		return nil
	}

	indexes := map[string]int{}
	for batchIndex, batch := range batches {
		for _, module := range batch {
			indexes[module] = batchIndex
		}
	}
	return indexes
}

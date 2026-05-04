package resolver

func initializationBatchWidths(batches [][]string) []int {
	if len(batches) == 0 {
		return nil
	}

	widths := make([]int, 0, len(batches))
	for _, batch := range batches {
		widths = append(widths, len(batch))
	}
	return widths
}

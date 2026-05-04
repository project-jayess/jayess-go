package resolver

func widestInitializationBatchIndexes(widths []int) ([]int, int) {
	return initializationBatchExtremaIndexes(widths, func(width int, selectedWidth int) bool {
		return width > selectedWidth
	})
}

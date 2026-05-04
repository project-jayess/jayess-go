package resolver

func initializationBatchExtremaIndexes(widths []int, better func(int, int) bool) ([]int, int) {
	if len(widths) == 0 {
		return nil, 0
	}

	selectedWidth := 0
	var indexes []int
	for index, width := range widths {
		if indexes == nil || better(width, selectedWidth) {
			selectedWidth = width
			indexes = []int{index}
			continue
		}
		if width == selectedWidth {
			indexes = append(indexes, index)
		}
	}
	return indexes, selectedWidth
}

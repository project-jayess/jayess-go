package resolver

// InitializationBatchWidthRange describes the smallest and largest dependency-ready batch widths.
type InitializationBatchWidthRange struct {
	Min int
	Max int
}

func initializationBatchWidthRange(widths []int) InitializationBatchWidthRange {
	if len(widths) == 0 {
		return InitializationBatchWidthRange{}
	}

	minWidth := widths[0]
	maxWidth := widths[0]
	for _, width := range widths[1:] {
		if width < minWidth {
			minWidth = width
		}
		if width > maxWidth {
			maxWidth = width
		}
	}
	return InitializationBatchWidthRange{
		Min: minWidth,
		Max: maxWidth,
	}
}

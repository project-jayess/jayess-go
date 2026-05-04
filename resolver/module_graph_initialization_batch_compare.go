package resolver

func initializationBatchesContainSameBatch(batches [][]string, first string, second string) bool {
	for _, batch := range batches {
		firstSeen := false
		secondSeen := false
		for _, module := range batch {
			if module == first {
				firstSeen = true
			}
			if module == second {
				secondSeen = true
			}
		}
		if firstSeen && secondSeen {
			return true
		}
	}
	return false
}

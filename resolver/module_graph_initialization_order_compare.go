package resolver

func initializationOrderPositionsBefore(positions map[string]int, before string, after string) bool {
	if positions == nil {
		return false
	}

	beforePosition, beforeOK := positions[before]
	afterPosition, afterOK := positions[after]
	if !beforeOK || !afterOK {
		return false
	}
	return beforePosition < afterPosition
}

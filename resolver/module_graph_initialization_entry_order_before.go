package resolver

// InitializesBefore reports whether one module initializes before another for an entry.
func (g *ModuleGraph) InitializesBefore(entry string, before string, after string) (bool, error) {
	positions, err := g.InitializationOrderPositions(entry)
	if err != nil {
		return false, err
	}
	return initializationOrderPositionsBefore(positions, before, after), nil
}

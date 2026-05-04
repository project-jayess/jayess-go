package resolver

// InitializesBeforeFor reports whether one module initializes before another for entries.
func (g *ModuleGraph) InitializesBeforeFor(entries []string, before string, after string) (bool, error) {
	positions, err := g.InitializationOrderPositionsFor(entries)
	if err != nil {
		return false, err
	}
	return initializationOrderPositionsBefore(positions, before, after), nil
}

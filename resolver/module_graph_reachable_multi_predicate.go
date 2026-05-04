package resolver

// ReachesModuleFor reports whether a module is reachable from multiple entries.
func (g *ModuleGraph) ReachesModuleFor(entries []string, module string) (bool, error) {
	set, err := g.ReachableModuleSetFor(entries)
	if err != nil {
		return false, err
	}
	return set[module], nil
}

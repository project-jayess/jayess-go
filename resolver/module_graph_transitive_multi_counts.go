package resolver

// TransitiveDependencyCountFor returns the number of dependency modules reachable from multiple entries.
func (g *ModuleGraph) TransitiveDependencyCountFor(entries []string) (int, error) {
	dependencies, err := g.TransitiveDependenciesFor(entries)
	if err != nil {
		return 0, err
	}
	return len(dependencies), nil
}

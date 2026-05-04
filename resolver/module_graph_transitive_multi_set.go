package resolver

// TransitiveDependencySetFor returns dependency modules reachable from multiple entries as a set.
func (g *ModuleGraph) TransitiveDependencySetFor(entries []string) (map[string]bool, error) {
	dependencies, err := g.TransitiveDependenciesFor(entries)
	if err != nil {
		return nil, err
	}
	if len(dependencies) == 0 {
		return nil, nil
	}

	set := make(map[string]bool, len(dependencies))
	for _, dependency := range dependencies {
		set[dependency] = true
	}
	return set, nil
}

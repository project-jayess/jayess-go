package resolver

// TransitiveDependencySet returns dependency modules reachable from an entry as a set.
func (g *ModuleGraph) TransitiveDependencySet(entry string) (map[string]bool, error) {
	dependencies, err := g.TransitiveDependencies(entry)
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

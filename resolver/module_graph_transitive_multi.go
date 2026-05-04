package resolver

// TransitiveDependenciesFor returns dependency modules reachable from multiple entries.
func (g *ModuleGraph) TransitiveDependenciesFor(entries []string) ([]string, error) {
	order, err := g.InitializationOrderFor(entries)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}

	entrySet := map[string]bool{}
	for _, entry := range entries {
		entrySet[entry] = true
	}

	dependencies := make([]string, 0, len(order))
	for _, module := range order {
		if !entrySet[module] {
			dependencies = append(dependencies, module)
		}
	}
	return dependencies, nil
}

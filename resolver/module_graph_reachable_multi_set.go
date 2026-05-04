package resolver

// ReachableModuleSetFor returns modules reachable from multiple entries as a set.
func (g *ModuleGraph) ReachableModuleSetFor(entries []string) (map[string]bool, error) {
	modules, err := g.ReachableModulesFor(entries)
	if err != nil {
		return nil, err
	}
	if len(modules) == 0 {
		return nil, nil
	}

	set := make(map[string]bool, len(modules))
	for _, module := range modules {
		set[module] = true
	}
	return set, nil
}

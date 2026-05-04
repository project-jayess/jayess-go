package resolver

// ReachableModuleSet returns modules reachable from an entry module as a set.
func (g *ModuleGraph) ReachableModuleSet(entry string) (map[string]bool, error) {
	modules, err := g.ReachableModules(entry)
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

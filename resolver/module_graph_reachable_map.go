package resolver

// ReachableModuleMap returns reachable modules by entry module.
func (g *ModuleGraph) ReachableModuleMap() (map[string][]string, error) {
	if g.imports == nil {
		return nil, nil
	}

	reachable := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		modules, err := g.ReachableModules(module)
		if err != nil {
			return nil, err
		}
		reachable[module] = modules
	}
	return reachable, nil
}

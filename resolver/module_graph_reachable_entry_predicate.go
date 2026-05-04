package resolver

// ReachesModule reports whether a module is reachable from an entry module.
func (g *ModuleGraph) ReachesModule(entry string, module string) (bool, error) {
	set, err := g.ReachableModuleSet(entry)
	if err != nil {
		return false, err
	}
	return set[module], nil
}

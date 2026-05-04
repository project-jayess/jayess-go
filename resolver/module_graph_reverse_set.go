package resolver

// TransitiveDependentSet returns modules that directly or indirectly import a module as a set.
func (g *ModuleGraph) TransitiveDependentSet(module string) map[string]bool {
	dependents := g.TransitiveDependents(module)
	if len(dependents) == 0 {
		return nil
	}

	set := make(map[string]bool, len(dependents))
	for _, dependent := range dependents {
		set[dependent] = true
	}
	return set
}

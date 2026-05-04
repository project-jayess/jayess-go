package resolver

// TransitiveDependentSetFor returns modules that directly or indirectly import any module as a set.
func (g *ModuleGraph) TransitiveDependentSetFor(modules []string) map[string]bool {
	dependents := g.TransitiveDependentsFor(modules)
	if len(dependents) == 0 {
		return nil
	}

	set := make(map[string]bool, len(dependents))
	for _, dependent := range dependents {
		set[dependent] = true
	}
	return set
}

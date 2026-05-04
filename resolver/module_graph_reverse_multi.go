package resolver

import "sort"

// TransitiveDependentsFor returns modules that directly or indirectly import any module.
func (g *ModuleGraph) TransitiveDependentsFor(modules []string) []string {
	if len(modules) == 0 {
		return nil
	}

	dependentSet := map[string]bool{}
	for _, module := range modules {
		for _, dependent := range g.TransitiveDependents(module) {
			dependentSet[dependent] = true
		}
	}
	if len(dependentSet) == 0 {
		return nil
	}

	dependents := make([]string, 0, len(dependentSet))
	for dependent := range dependentSet {
		dependents = append(dependents, dependent)
	}
	sort.Strings(dependents)
	return dependents
}

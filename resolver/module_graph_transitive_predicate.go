package resolver

func (g *ModuleGraph) TransitivelyDependsOn(module string, dependency string) (bool, error) {
	dependencies, err := g.TransitiveDependencies(module)
	if err != nil {
		return false, err
	}
	for _, current := range dependencies {
		if current == dependency {
			return true, nil
		}
	}
	return false, nil
}

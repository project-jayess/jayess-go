package resolver

func (g *ModuleGraph) TransitiveDependencies(entry string) ([]string, error) {
	order, err := g.InitializationOrder(entry)
	if err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}
	dependencies := make([]string, 0, len(order)-1)
	for _, module := range order {
		if module != entry {
			dependencies = append(dependencies, module)
		}
	}
	return dependencies, nil
}

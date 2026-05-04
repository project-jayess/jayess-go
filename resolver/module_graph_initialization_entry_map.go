package resolver

func initializationEntryMap[T any](g *ModuleGraph, resolve func(string) (T, error)) (map[string]T, error) {
	if g.imports == nil {
		return nil, nil
	}

	values := make(map[string]T, len(g.imports))
	for _, module := range g.Modules() {
		value, err := resolve(module)
		if err != nil {
			return nil, err
		}
		values[module] = value
	}
	return values, nil
}

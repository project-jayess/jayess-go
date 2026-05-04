package ast

func CompactModuleDependencies(dependencies []ModuleDependency) []ModuleDependency {
	if len(dependencies) == 0 {
		return nil
	}
	indexBySource := map[string]int{}
	compacted := make([]ModuleDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		if index, ok := indexBySource[dependency.Source]; ok {
			compacted[index].ReExport = compacted[index].ReExport || dependency.ReExport
			compacted[index].SideEffect = compacted[index].SideEffect || dependency.SideEffect
			continue
		}
		indexBySource[dependency.Source] = len(compacted)
		compacted = append(compacted, dependency)
	}
	return compacted
}

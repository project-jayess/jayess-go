package resolver

func CompactResolvedModuleDependencies(dependencies []ResolvedModuleDependency) []ResolvedModuleDependency {
	if len(dependencies) == 0 {
		return nil
	}
	indexByPath := map[string]int{}
	compacted := make([]ResolvedModuleDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		if index, ok := indexByPath[dependency.Path]; ok {
			compacted[index].ReExport = compacted[index].ReExport || dependency.ReExport
			compacted[index].SideEffect = compacted[index].SideEffect || dependency.SideEffect
			continue
		}
		indexByPath[dependency.Path] = len(compacted)
		compacted = append(compacted, dependency)
	}
	return compacted
}
